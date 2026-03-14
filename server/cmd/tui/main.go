package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bis-code/claude-toolkit/server/internal/db"
)

// Tokyo Night colors
var (
	colorBg      = lipgloss.Color("#1a1b26")
	colorSurface = lipgloss.Color("#24283b")
	colorBorder  = lipgloss.Color("#3b4261")
	colorText    = lipgloss.Color("#c0caf5")
	colorMuted   = lipgloss.Color("#565f89")
	colorAccent  = lipgloss.Color("#7aa2f7")
	colorSuccess = lipgloss.Color("#9ece6a")
	colorWarning = lipgloss.Color("#e0af68")
	colorDanger  = lipgloss.Color("#f7768e")
	colorInfo    = lipgloss.Color("#7dcfff")
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorWarning)

	dangerStyle = lipgloss.NewStyle().
			Foreground(colorDanger)

	infoStyle = lipgloss.NewStyle().
			Foreground(colorInfo)

	accentStyle = lipgloss.NewStyle().
			Foreground(colorAccent)
)

type tab int

const (
	tabSessions tab = iota
	tabRules
	tabPatrol
	tabLearning
)

type model struct {
	store      *db.Store
	activeTab  tab
	width      int
	height     int
	sessions   []*db.Session
	events     []db.Event
	rules      []db.Rule
	skills     []db.SkillScore
	workflows  []db.WorkflowEvent
	lastUpdate time.Time
	err        error
}

type tickMsg time.Time
type dataMsg struct {
	sessions  []*db.Session
	events    []db.Event
	rules     []db.Rule
	skills    []db.SkillScore
	workflows []db.WorkflowEvent
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) fetchData() tea.Msg {
	sessions, _ := m.store.ListSessions("", 10)
	rules, _ := m.store.ListRules("", "", "")
	skills, _ := m.store.ListSkillScores()

	var events []db.Event
	if len(sessions) > 0 {
		evts, _ := m.store.ListEvents(sessions[0].ID)
		if len(evts) > 10 {
			evts = evts[len(evts)-10:]
		}
		for _, e := range evts {
			events = append(events, *e)
		}
	}

	var workflows []db.WorkflowEvent
	wes, _ := m.store.ListWorkflowEvents("")
	for _, we := range wes {
		if we.Confidence > 0.5 {
			workflows = append(workflows, we)
		}
	}

	return dataMsg{sessions: sessions, events: events, rules: rules, skills: skills, workflows: workflows}
}

func initialModel(store *db.Store) model {
	return model{store: store, activeTab: tabSessions}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), m.fetchData)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "x", "ctrl+c":
			return m, tea.Quit
		case "1", "s":
			m.activeTab = tabSessions
		case "2", "r":
			m.activeTab = tabRules
		case "3", "p":
			m.activeTab = tabPatrol
		case "4", "l":
			m.activeTab = tabLearning
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		return m, tea.Batch(tickCmd(), m.fetchData)
	case dataMsg:
		m.sessions = msg.sessions
		m.events = msg.events
		m.rules = msg.rules
		m.skills = msg.skills
		m.workflows = msg.workflows
		m.lastUpdate = time.Now()
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	w := m.width - 4
	if w < 40 {
		w = 40
	}

	var sb strings.Builder

	// Header
	header := titleStyle.Render("CLAUDE TOOLKIT") + " " +
		mutedStyle.Render("v5.0.0") + "  " +
		successStyle.Render("● Live")
	sb.WriteString(header + "\n\n")

	// Active session — consider active if latest event is within 5 minutes
	if len(m.sessions) > 0 {
		s := m.sessions[0]
		dur := time.Since(s.StartedAt).Round(time.Second).String()
		active := len(m.events) > 0 && time.Since(m.events[len(m.events)-1].Timestamp).Minutes() < 5
		if !active && s.EndedAt == nil {
			// No recent events but no ended_at — check if started recently
			active = time.Since(s.StartedAt).Minutes() < 5
		}
		status := successStyle.Render("● active")
		if !active {
			status = mutedStyle.Render("○ idle")
		}
		eventCount := len(m.events)
		sessionLine := fmt.Sprintf("%s  %s  %s  %s  events: %d",
			status,
			accentStyle.Render(shortID(s.ID)),
			headerStyle.Render(s.Project),
			mutedStyle.Render(dur),
			eventCount)
		sb.WriteString(panelStyle.Width(w).Render(sessionLine) + "\n\n")
	}

	// Tab content
	switch m.activeTab {
	case tabSessions:
		sb.WriteString(m.renderSessions(w))
	case tabRules:
		sb.WriteString(m.renderSkills(w))
	case tabPatrol:
		sb.WriteString(m.renderPatrol(w))
	case tabLearning:
		sb.WriteString(m.renderLearning(w))
	}

	// Tab bar
	tabs := []struct {
		key   string
		label string
		t     tab
	}{
		{"1", "sessions", tabSessions},
		{"2", "rules/skills", tabRules},
		{"3", "patrol", tabPatrol},
		{"4", "learning", tabLearning},
	}

	var tabParts []string
	for _, t := range tabs {
		if t.t == m.activeTab {
			tabParts = append(tabParts, accentStyle.Render(fmt.Sprintf("[%s]%s", t.key, t.label)))
		} else {
			tabParts = append(tabParts, mutedStyle.Render(fmt.Sprintf("[%s]%s", t.key, t.label)))
		}
	}
	tabParts = append(tabParts, dangerStyle.Render("[x]exit"))
	sb.WriteString("\n" + strings.Join(tabParts, "  "))

	return sb.String()
}

func (m model) renderSessions(w int) string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Recent Sessions") + "\n")

	if len(m.sessions) == 0 {
		sb.WriteString(mutedStyle.Render("  No sessions yet") + "\n")
		return panelStyle.Width(w).Render(sb.String())
	}

	for i, s := range m.sessions {
		if i >= 8 {
			break
		}
		dur := time.Since(s.StartedAt).Round(time.Second).String()
		line := fmt.Sprintf("  %s  %-15s  %s",
			mutedStyle.Render(shortID(s.ID)),
			s.Project,
			dur)
		sb.WriteString(line + "\n")
	}

	// Recent events
	if len(m.events) > 0 {
		sb.WriteString("\n" + headerStyle.Render("Live Events") + "\n")
		for _, e := range m.events {
			ts := e.Timestamp.Format("15:04:05")
			result := successStyle.Render("✓")
			if e.Result == "failure" || e.Result == "error" {
				result = dangerStyle.Render("✗")
			}
			line := fmt.Sprintf("  %s  %-8s %-30s %s",
				mutedStyle.Render(ts),
				infoStyle.Render(e.Type),
				truncate(e.Details, 30),
				result)
			sb.WriteString(line + "\n")
		}
	}

	return panelStyle.Width(w).Render(sb.String())
}

func (m model) renderSkills(w int) string {
	var sb strings.Builder

	// Rules section
	sb.WriteString(headerStyle.Render("Rules") + "\n")
	if len(m.rules) == 0 {
		sb.WriteString(mutedStyle.Render("  No rules yet") + "\n")
	} else {
		for i, r := range m.rules {
			if i >= 10 {
				sb.WriteString(mutedStyle.Render(fmt.Sprintf("  ... and %d more", len(m.rules)-10)) + "\n")
				break
			}
			// Effectiveness indicator
			eff := ""
			if r.Effectiveness >= 0.7 {
				eff = successStyle.Render("●")
			} else if r.Effectiveness >= 0.4 {
				eff = warningStyle.Render("●")
			} else if r.Effectiveness > 0 {
				eff = dangerStyle.Render("●")
			} else {
				eff = mutedStyle.Render("○")
			}
			scope := mutedStyle.Render(fmt.Sprintf("[%s]", r.Scope))
			content := truncate(r.Content, w-30)
			sb.WriteString(fmt.Sprintf("  %s %s %s\n", eff, scope, content))
		}
	}

	sb.WriteString("\n" + headerStyle.Render("Skill Effectiveness") + "\n")
	if len(m.skills) == 0 {
		sb.WriteString(mutedStyle.Render("  No skill data yet") + "\n")
	} else {
		for _, s := range m.skills {
			barLen := int(s.Effectiveness * 10)
			bar := strings.Repeat("█", barLen) + strings.Repeat("░", 10-barLen)

			style := successStyle
			warn := ""
			if s.Effectiveness < 0.5 {
				style = dangerStyle
				warn = " ⚠"
			} else if s.Effectiveness < 0.7 {
				style = warningStyle
				warn = " ⚠"
			}

			line := fmt.Sprintf("  %-20s %s %.2f%s",
				s.Name,
				style.Render(bar),
				s.Effectiveness,
				warn)
			sb.WriteString(line + "\n")
		}
	}

	return panelStyle.Width(w).Render(sb.String())
}

func (m model) renderPatrol(w int) string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Patrol Status") + "\n")
	sb.WriteString("  " + successStyle.Render("●") + " No active alerts\n")
	sb.WriteString(mutedStyle.Render("  Patrol checks run automatically on session end") + "\n")
	return panelStyle.Width(w).Render(sb.String())
}

func (m model) renderLearning(w int) string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Learned Patterns") + "\n")

	if len(m.workflows) == 0 {
		sb.WriteString(mutedStyle.Render("  No patterns learned yet. Use the toolkit for a few sessions.") + "\n")
		return panelStyle.Width(w).Render(sb.String())
	}

	categories := map[string][]db.WorkflowEvent{}
	for _, we := range m.workflows {
		categories[we.Category] = append(categories[we.Category], we)
	}

	catOrder := []string{"coding_pattern", "task_execution", "problem_solving", "preference"}
	catLabels := map[string]string{
		"coding_pattern":  "Coding",
		"task_execution":  "Execution",
		"problem_solving": "Problem Solving",
		"preference":      "Preferences",
	}

	for _, cat := range catOrder {
		entries := categories[cat]
		if len(entries) == 0 {
			continue
		}
		sb.WriteString("  " + accentStyle.Render(catLabels[cat]) + "\n")
		for _, we := range entries {
			confBar := int(we.Confidence * 5)
			bar := strings.Repeat("●", confBar) + strings.Repeat("○", 5-confBar)
			line := fmt.Sprintf("    %-25s %s  %.2f  (%d)",
				truncate(we.Pattern, 25),
				infoStyle.Render(bar),
				we.Confidence,
				we.Occurrences)
			sb.WriteString(line + "\n")
		}
	}

	return panelStyle.Width(w).Render(sb.String())
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func main() {
	store, err := db.NewStore("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	p := tea.NewProgram(initialModel(store), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
