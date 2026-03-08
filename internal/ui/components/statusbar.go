package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/ui/theme"
)

// ViewHints contains view-specific keyboard hints
type ViewHints struct {
	Pods        string
	Deployments string
	Services    string
	ConfigMaps  string
	Secrets     string
	Nodes       string
	Events      string
	Default     string
}

// DefaultViewHints returns the default view-specific hints
var DefaultViewHints = ViewHints{
	Pods:        "L logs  d describe  ctrl+d delete",
	Deployments: "r restart  s scale  d describe",
	Services:    "d describe  ctrl+d delete",
	ConfigMaps:  "d describe  ctrl+d delete",
	Secrets:     "d describe  ctrl+d delete",
	Nodes:       "d describe  c cordon",
	Events:      "d describe",
	Default:     "d describe  ctrl+d delete",
}

// StatusBar is the bottom status bar component
type StatusBar struct {
	width         int
	message       string
	isError       bool
	resourceCount int
	filteredCount int
	selectedIndex int
	helpText      string
	viewType      string
	resourceName  string
}

// NewStatusBar creates a new status bar component
func NewStatusBar() *StatusBar {
	return &StatusBar{
		width:    80,
		helpText: "↑↓ Navigate  Enter Select  / Filter  : Command  ? Help  q Quit",
		viewType: "default",
	}
}

// SetWidth sets the status bar width
func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

// SetMessage sets the status message
func (s *StatusBar) SetMessage(message string, isError bool) {
	s.message = message
	s.isError = isError
}

// ClearMessage clears the status message
func (s *StatusBar) ClearMessage() {
	s.message = ""
	s.isError = false
}

// SetResourceCount sets the resource count display
func (s *StatusBar) SetResourceCount(total, filtered, selected int) {
	s.resourceCount = total
	s.filteredCount = filtered
	s.selectedIndex = selected
}

// SetHelpText sets the help text
func (s *StatusBar) SetHelpText(text string) {
	s.helpText = text
}

// SetViewType sets the current view type for contextual hints
func (s *StatusBar) SetViewType(viewType string) {
	s.viewType = viewType
}

// SetResourceName sets the name of the current resource type (e.g., "Pods")
func (s *StatusBar) SetResourceName(name string) {
	s.resourceName = name
}

// GetContextualHints returns view-specific hints
func (s *StatusBar) GetContextualHints() string {
	switch s.viewType {
	case "pods":
		return DefaultViewHints.Pods
	case "deployments":
		return DefaultViewHints.Deployments
	case "services":
		return DefaultViewHints.Services
	case "configmaps":
		return DefaultViewHints.ConfigMaps
	case "secrets":
		return DefaultViewHints.Secrets
	case "nodes":
		return DefaultViewHints.Nodes
	case "events":
		return DefaultViewHints.Events
	default:
		return DefaultViewHints.Default
	}
}

// View renders the status bar
func (s *StatusBar) View() string {
	style := theme.Styles.StatusBar.Width(s.width)

	// Left side: message or resource count with resource name
	var leftContent string
	if s.message != "" {
		msgStyle := theme.Styles.Base
		if s.isError {
			msgStyle = theme.Styles.StatusError
		}
		leftContent = msgStyle.Render(s.message)
	} else {
		// Show position and count with resource name
		resourceLabel := s.resourceName
		if resourceLabel == "" {
			resourceLabel = "items"
		}

		if s.filteredCount != s.resourceCount && s.filteredCount > 0 {
			leftContent = fmt.Sprintf("[%d/%d %s] filtered from %d", s.selectedIndex+1, s.filteredCount, resourceLabel, s.resourceCount)
		} else if s.resourceCount > 0 {
			leftContent = fmt.Sprintf("[%d/%d %s]", s.selectedIndex+1, s.resourceCount, resourceLabel)
		} else {
			leftContent = fmt.Sprintf("[0 %s]", resourceLabel)
		}
	}

	// Middle: contextual hints
	contextHints := s.formatContextualHints()

	// Right side: basic help
	rightContent := s.formatBasicHelp()

	// Calculate padding
	leftWidth := lipgloss.Width(leftContent)
	middleWidth := lipgloss.Width(contextHints)
	rightWidth := lipgloss.Width(rightContent)
	totalContent := leftWidth + middleWidth + rightWidth

	// Distribute space
	availableSpace := s.width - totalContent - 4
	if availableSpace < 2 {
		availableSpace = 2
	}

	leftPadding := availableSpace / 2
	rightPadding := availableSpace - leftPadding

	content := leftContent + strings.Repeat(" ", leftPadding) + contextHints + strings.Repeat(" ", rightPadding) + rightContent
	return style.Render(content)
}

// formatContextualHints formats the view-specific hints with styling
func (s *StatusBar) formatContextualHints() string {
	hints := s.GetContextualHints()
	if hints == "" {
		return ""
	}

	// Parse and style the hints (format: "key action  key action")
	parts := strings.Split(hints, "  ")
	var styled []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Split on first space to get key and description
		fields := strings.SplitN(part, " ", 2)
		if len(fields) == 2 {
			key := theme.Styles.HelpKey.Render(fields[0])
			desc := theme.Styles.HelpDesc.Render(fields[1])
			styled = append(styled, key+" "+desc)
		} else {
			styled = append(styled, theme.Styles.HelpDesc.Render(part))
		}
	}
	return strings.Join(styled, "  ")
}

// formatBasicHelp formats the basic help hints
func (s *StatusBar) formatBasicHelp() string {
	hints := []struct{ key, desc string }{
		{"?", "Help"},
		{"q", "Quit"},
	}

	var parts []string
	for _, h := range hints {
		key := theme.Styles.HelpKey.Render(h.key)
		desc := theme.Styles.HelpDesc.Render(h.desc)
		parts = append(parts, key+" "+desc)
	}
	return strings.Join(parts, "  ")
}

// ShortcutsView renders a minimal shortcuts-only footer
// Format: <:command  /filter  ?help  ctrl+p:palette  q:quit>
func (s *StatusBar) ShortcutsView() string {
	shortcuts := []struct{ key, desc string }{
		{":", "command"},
		{"/", "filter"},
		{"?", "help"},
		{"ctrl+p", "palette"},
		{"q", "quit"},
	}

	var parts []string
	for _, sc := range shortcuts {
		key := theme.Styles.ShortcutKey.Render(sc.key)
		desc := theme.Styles.ShortcutDesc.Render(sc.desc)
		parts = append(parts, key+":"+desc)
	}

	content := "<" + strings.Join(parts, "  ") + ">"
	return theme.Styles.ShortcutsBar.Width(s.width).Render(content)
}

// HelpBar provides a compact help display
type HelpBar struct {
	width    int
	bindings []HelpBinding
}

// HelpBinding represents a single key binding for help display
type HelpBinding struct {
	Key  string
	Desc string
}

// NewHelpBar creates a new help bar
func NewHelpBar() *HelpBar {
	return &HelpBar{
		width: 80,
		bindings: []HelpBinding{
			{Key: "up/dn", Desc: "Navigate"},
			{Key: "Enter", Desc: "Select"},
			{Key: "/", Desc: "Filter"},
			{Key: ":", Desc: "Command"},
			{Key: "?", Desc: "Help"},
			{Key: "q", Desc: "Quit"},
		},
	}
}

// SetWidth sets the help bar width
func (h *HelpBar) SetWidth(width int) {
	h.width = width
}

// SetBindings sets the key bindings to display
func (h *HelpBar) SetBindings(bindings []HelpBinding) {
	h.bindings = bindings
}

// View renders the help bar
func (h *HelpBar) View() string {
	var parts []string
	for _, b := range h.bindings {
		key := theme.Styles.HelpKey.Render(b.Key)
		desc := theme.Styles.HelpDesc.Render(b.Desc)
		parts = append(parts, key+" "+desc)
	}

	content := strings.Join(parts, "  ")

	// Truncate if too long
	if lipgloss.Width(content) > h.width {
		// Show fewer bindings
		truncated := make([]string, 0)
		totalWidth := 0
		for _, b := range h.bindings {
			part := theme.Styles.HelpKey.Render(b.Key) + " " + theme.Styles.HelpDesc.Render(b.Desc)
			partWidth := lipgloss.Width(part) + 2
			if totalWidth+partWidth > h.width-3 {
				break
			}
			truncated = append(truncated, part)
			totalWidth += partWidth
		}
		content = strings.Join(truncated, "  ")
	}

	return theme.Styles.Help.Width(h.width).Render(content)
}
