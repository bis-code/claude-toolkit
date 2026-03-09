package workspace

// Config represents a .claude-workspace.json file.
type Config struct {
	Name            string   `json:"name"`
	Repos           []Repo   `json:"repos"`
	Shared          []string `json:"shared,omitempty"`
	PlanningRepo    string   `json:"planning_repo,omitempty"`
	CrossRepoRules  []string `json:"cross_repo_rules,omitempty"`
	DependencyOrder []string `json:"dependency_order,omitempty"`
	DomainLabels    []string `json:"domain_labels,omitempty"`
}

// Repo represents a single repository in the workspace.
type Repo struct {
	Path   string `json:"path"`
	Branch string `json:"branch,omitempty"`
	Type   string `json:"type,omitempty"`
}
