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
	commands   []string // Available commands for completion
	history    []string
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

	return &CommandInput{
		width:   80,
		visible: false,
		input:   ti,
		commands: []string{
			"q", "quit",
			"delete", "del",
			"describe", "desc",
			"logs", "log",
			"scale",
			"ns", "namespace",
			"ctx", "context",
			"refresh", "r",
			"help", "h",
			"pods", "deployments", "services",
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
	c.historyIdx = len(c.history)
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

		case msg.Type == tea.KeyTab:
			// Tab completion
			c.completeCommand()
			return c, nil
		}
	}

	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	return c, cmd
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

// completeCommand provides tab completion for commands
func (c *CommandInput) completeCommand() {
	current := strings.ToLower(c.input.Value())
	if current == "" {
		return
	}

	// Find first matching command
	for _, cmd := range c.commands {
		if strings.HasPrefix(cmd, current) {
			c.input.SetValue(cmd)
			c.input.CursorEnd()
			return
		}
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
