package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/ui/theme"
)

// CommandExecuteMsg is sent when a command is executed
type CommandExecuteMsg struct {
	Command string
	Args    []string
}

// CommandCancelMsg is sent when command mode is cancelled
type CommandCancelMsg struct{}

// CommandInput provides a vim-style command line input
type CommandInput struct {
	width      int
	visible    bool
	input      textinput.Model
	commands      []string // Available commands for completion
	namespaces    []string // Dynamic namespace completions
	contexts      []string // Dynamic context completions
	pfSessionIDs  []string // Dynamic port-forward session ID completions
	history       []string
	historyIdx int
}

// NewCommandInput creates a new command input component
func NewCommandInput() *CommandInput {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.CharLimit = 256
	ti.Width = 60
	// Apply background to textinput styles
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.ColorText).Background(theme.ColorBackground)
	ti.PromptStyle = lipgloss.NewStyle().Background(theme.ColorBackground)
	ti.Cursor.Style = lipgloss.NewStyle().Background(theme.ColorHighlight)
	ti.ShowSuggestions = true
	ti.CompletionStyle = lipgloss.NewStyle().Foreground(theme.ColorMuted).Background(theme.ColorBackground)

	return &CommandInput{
		width:   80,
		visible: false,
		input:   ti,
		commands: []string{
			// Resource views (canonical names first)
			"api-resources", "configmaps", "cronjobs", "daemonsets", "deployments",
			"events", "helm", "hpa", "ingresses", "jobs",
			"nodes", "pods", "portforwards", "pvcs",
			"replicasets", "rolebindings", "secrets", "services", "statefulsets",
			// Command-only views (no keybinding equivalent)
			"diagnosis", "health", "pulse", "themes", "timeline", "xray",
			// Navigation & utility
			"help", "namespace", "quit", "refresh",
			// Short aliases
			"ar", "cj", "cm", "ctx", "deploy", "diag",
			"ds", "ev", "graph", "h", "hr", "ing",
			"no", "ns", "pf", "pf-stop", "pv", "pvc", "q",
			"r", "rb", "rel", "releases", "rs", "sec",
			"sts", "svc", "tl",
		},
		history:    []string{},
		historyIdx: -1,
	}
}

// SetWidth sets the command input width
func (c *CommandInput) SetWidth(width int) {
	c.width = width
	c.input.Width = width - 4 // Account for ": " prefix and padding
}

// IsVisible returns whether the command input is visible
func (c *CommandInput) IsVisible() bool {
	return c.visible
}

// Show shows the command input and focuses it
func (c *CommandInput) Show() {
	c.visible = true
	c.input.Reset()
	c.input.Focus()
	c.input.SetSuggestions(c.commands)
	c.historyIdx = len(c.history)
}

// SetNamespaces updates the namespace list for dynamic `:ns` completion
func (c *CommandInput) SetNamespaces(namespaces []string) {
	c.namespaces = namespaces
}

// SetContexts updates the context list for dynamic `:ctx` completion
func (c *CommandInput) SetContexts(contexts []string) {
	c.contexts = contexts
}

// SetPortForwardIDs updates the port-forward session IDs for `:pf-stop` completion
func (c *CommandInput) SetPortForwardIDs(ids []string) {
	c.pfSessionIDs = ids
}

// Hide hides the command input
func (c *CommandInput) Hide() {
	c.visible = false
	c.input.Blur()
}

// Value returns the current input value
func (c *CommandInput) Value() string {
	return c.input.Value()
}

// Update handles input events
func (c *CommandInput) Update(msg tea.Msg) (*CommandInput, tea.Cmd) {
	if !c.visible {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			c.Hide()
			return c, func() tea.Msg {
				return CommandCancelMsg{}
			}

		case msg.Type == tea.KeyEnter:
			cmd := strings.TrimSpace(c.input.Value())
			if cmd != "" {
				// Add to history
				c.history = append(c.history, cmd)
				c.Hide()
				return c, func() tea.Msg {
					return c.parseCommand(cmd)
				}
			}
			c.Hide()
			return c, func() tea.Msg {
				return CommandCancelMsg{}
			}

		case msg.Type == tea.KeyUp:
			// Navigate history up
			if len(c.history) > 0 && c.historyIdx > 0 {
				c.historyIdx--
				c.input.SetValue(c.history[c.historyIdx])
				c.input.CursorEnd()
			}
			return c, nil

		case msg.Type == tea.KeyDown:
			// Navigate history down
			if c.historyIdx < len(c.history)-1 {
				c.historyIdx++
				c.input.SetValue(c.history[c.historyIdx])
				c.input.CursorEnd()
			} else if c.historyIdx == len(c.history)-1 {
				c.historyIdx = len(c.history)
				c.input.SetValue("")
			}
			return c, nil
		}
	}

	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	c.updateDynamicSuggestions()
	return c, cmd
}

// xrayKinds are the valid kind aliases for :xray completion (from resolveKindAlias).
var xrayKinds = []string{
	"deploy", "pod", "svc", "sts", "ds", "job", "cj", "rs",
	"cm", "sec", "ing", "pvc", "pv", "hpa", "node",
}

// updateDynamicSuggestions switches suggestions based on current input prefix.
func (c *CommandInput) updateDynamicSuggestions() {
	val := c.input.Value()

	// :ns / :namespace → namespace names (with "all" first)
	if len(c.namespaces) > 0 {
		if prefix, ok := matchPrefix(val, "ns ", "namespace "); ok {
			suggestions := make([]string, 0, len(c.namespaces)+1)
			suggestions = append(suggestions, prefix+"all")
			for _, ns := range c.namespaces {
				suggestions = append(suggestions, prefix+ns)
			}
			c.input.SetSuggestions(suggestions)
			return
		}
	}

	// :ctx / :context → context names
	if len(c.contexts) > 0 {
		if prefix, ok := matchPrefix(val, "ctx ", "context "); ok {
			suggestions := make([]string, 0, len(c.contexts))
			for _, ctx := range c.contexts {
				suggestions = append(suggestions, prefix+ctx)
			}
			c.input.SetSuggestions(suggestions)
			return
		}
	}

	// :xray → kind aliases
	if strings.HasPrefix(val, "xray ") {
		suggestions := make([]string, 0, len(xrayKinds))
		for _, kind := range xrayKinds {
			suggestions = append(suggestions, "xray "+kind)
		}
		c.input.SetSuggestions(suggestions)
		return
	}

	// :pf-stop → "all" + active session IDs
	if strings.HasPrefix(val, "pf-stop ") {
		suggestions := []string{"pf-stop all"}
		for _, id := range c.pfSessionIDs {
			suggestions = append(suggestions, "pf-stop "+id)
		}
		c.input.SetSuggestions(suggestions)
		return
	}

	c.input.SetSuggestions(c.commands)
}

// matchPrefix checks if val starts with any of the given prefixes and returns the matched one.
func matchPrefix(val string, prefixes ...string) (string, bool) {
	for _, p := range prefixes {
		if strings.HasPrefix(val, p) {
			return p, true
		}
	}
	return "", false
}

// View renders the command input
func (c *CommandInput) View() string {
	if !c.visible {
		return ""
	}

	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	// Styled prefix with accent color
	prefix := theme.Styles.CommandPrefix.Render(":")
	space := bgStyle.Render(" ")
	input := c.input.View()

	content := prefix + space + input

	// Pad to full width with styled background
	contentWidth := lipgloss.Width(content)
	if contentWidth < c.width {
		padding := bgStyle.Render(strings.Repeat(" ", c.width-contentWidth))
		content = content + padding
	}

	return content
}

// ViewEmpty renders an empty command line placeholder
func (c *CommandInput) ViewEmpty() string {
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	prefix := theme.Styles.HelpDesc.Render(":")
	padding := bgStyle.Render(strings.Repeat(" ", c.width-1))
	content := prefix + padding
	return theme.Styles.CommandLine.Width(c.width).Render(content)
}

// parseCommand parses the command string and returns the appropriate message
func (c *CommandInput) parseCommand(cmd string) tea.Msg {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return CommandCancelMsg{}
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	return CommandExecuteMsg{
		Command: command,
		Args:    args,
	}
}

// GetCommands returns all available commands with descriptions
func (c *CommandInput) GetCommands() map[string]string {
	return map[string]string{
		"q":           "Quit the application",
		"quit":        "Quit the application",
		"delete":      "Delete the selected resource",
		"del":         "Delete the selected resource",
		"describe":    "Describe the selected resource",
		"desc":        "Describe the selected resource",
		"logs":        "View logs for the selected pod",
		"log":         "View logs for the selected pod",
		"scale":       "Scale a deployment (usage: scale <name> <replicas>)",
		"ns":          "Switch namespace (usage: ns <namespace>)",
		"namespace":   "Switch namespace (usage: namespace <namespace>)",
		"ctx":         "Switch context (usage: ctx <context>)",
		"context":     "Switch context (usage: context <context>)",
		"refresh":     "Refresh the current view",
		"r":           "Refresh the current view",
		"help":        "Show help",
		"h":           "Show help",
		"pods":        "Switch to pods view",
		"deployments": "Switch to deployments view",
		"services":    "Switch to services view",
	}
}
