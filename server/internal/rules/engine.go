package rules

import (
	"strings"

	"github.com/bis-code/claude-toolkit/server/internal/db"
)

const DefaultTokenBudget = 2000

// approx tokens: ~4 chars per token (rough estimate)
const charsPerToken = 4

// MergeContext holds the context for rule merging.
type MergeContext struct {
	Project     string
	Workspace   string
	Task        string
	TechStack   []string
	TokenBudget int
	Variables   map[string]string
}

// Engine handles rule merging, variable substitution, and filtering.
type Engine struct {
	store *db.Store
}

// NewEngine creates a new rule engine backed by the given store.
func NewEngine(store *db.Store) *Engine {
	return &Engine{store: store}
}

// GetActiveRules merges rules across 4 scopes, substitutes variables,
// filters by tech stack, and respects the token budget.
func (e *Engine) GetActiveRules(ctx MergeContext) ([]db.Rule, error) {
	if ctx.TokenBudget <= 0 {
		ctx.TokenBudget = DefaultTokenBudget
	}

	// Collect rules from all applicable scopes
	var allRules []db.Rule

	// 1. Global rules (no tech_stack filter yet — we filter after collecting)
	globals, err := e.store.ListRules("global", "", "")
	if err != nil {
		return nil, err
	}
	allRules = append(allRules, globals...)

	// 2. Workspace rules
	if ctx.Workspace != "" {
		workspaceRules, err := e.store.ListRules("workspace", "", "")
		if err != nil {
			return nil, err
		}
		for _, r := range workspaceRules {
			if r.Workspace == ctx.Workspace {
				allRules = append(allRules, r)
			}
		}
	}

	// 3. Project rules
	if ctx.Project != "" {
		projectRules, err := e.store.ListRules("project", ctx.Project, "")
		if err != nil {
			return nil, err
		}
		allRules = append(allRules, projectRules...)
	}

	// 4. Task rules
	if ctx.Task != "" {
		taskRules, err := e.store.ListRules("task", ctx.Project, "")
		if err != nil {
			return nil, err
		}
		for _, r := range taskRules {
			allRules = append(allRules, r)
		}
	}

	// Filter by tech stack
	filtered := filterByTechStack(allRules, ctx.TechStack)

	// Apply variable substitution
	for i := range filtered {
		filtered[i].Content = substituteVariables(filtered[i].Content, ctx.Variables)
	}

	// Apply token budget (rules are already sorted by effectiveness from ListRules)
	budgeted := applyTokenBudget(filtered, ctx.TokenBudget)

	return budgeted, nil
}

// filterByTechStack removes rules that have tech_stack tags not matching the project's stack.
// Rules without tech_stack tags are always included (universal rules).
func filterByTechStack(rules []db.Rule, projectStack []string) []db.Rule {
	if len(projectStack) == 0 {
		return rules
	}

	stackSet := make(map[string]bool)
	for _, s := range projectStack {
		stackSet[s] = true
	}

	var result []db.Rule
	for _, r := range rules {
		stacks, hasTechStack := r.Tags["tech_stack"]
		if !hasTechStack || len(stacks) == 0 {
			// Universal rule — always include
			result = append(result, r)
			continue
		}

		// Include if ANY of the rule's tech stacks match the project
		for _, s := range stacks {
			if stackSet[s] {
				result = append(result, r)
				break
			}
		}
	}

	return result
}

// substituteVariables replaces {{variable}} placeholders in rule content.
func substituteVariables(content string, vars map[string]string) string {
	if len(vars) == 0 {
		return content
	}

	result := content
	for key, value := range vars {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}

// applyTokenBudget keeps rules until the token budget is exhausted.
// Rules are assumed to already be sorted by effectiveness (highest first).
func applyTokenBudget(rules []db.Rule, budget int) []db.Rule {
	var result []db.Rule
	usedTokens := 0

	for _, r := range rules {
		ruleTokens := len(r.Content) / charsPerToken
		if ruleTokens == 0 {
			ruleTokens = 1
		}

		if usedTokens+ruleTokens > budget {
			continue
		}

		result = append(result, r)
		usedTokens += ruleTokens
	}

	return result
}
