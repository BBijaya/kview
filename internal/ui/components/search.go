package components

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/ui/theme"
)

// FilterChangedMsg is sent when the filter changes
type FilterChangedMsg struct {
	Value string
}

// FilterClosedMsg is sent when the filter input is closed
type FilterClosedMsg struct {
	Submitted bool // true = Enter, false = Esc
}

// SearchInput is a search/filter input component
type SearchInput struct {
	input   textinput.Model
	width   int
	visible bool
}

// NewSearchInput creates a new search input
func NewSearchInput() *SearchInput {
	ti := textinput.New()
	ti.Placeholder = "Type to filter (! -f -l)..."
	ti.CharLimit = 100
	ti.SetWidth(30)
	// Apply background to textinput styles (same as CommandInput)
	styles := textinput.DefaultDarkStyles()
	styles.Focused.Text = lipgloss.NewStyle().Foreground(theme.ColorText).Background(theme.ColorBackground)
	styles.Focused.Prompt = lipgloss.NewStyle().Background(theme.ColorBackground)
	styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(theme.ColorMuted).Background(theme.ColorBackground)
	ti.SetStyles(styles)

	return &SearchInput{
		input:   ti,
		width:   40,
		visible: false,
	}
}

// SetWidth sets the input width
func (s *SearchInput) SetWidth(width int) {
	s.width = width
	s.input.SetWidth(width - 4) // Account for "/ " prefix and padding
}

// Show shows the input and focuses it
func (s *SearchInput) Show() {
	s.visible = true
	s.input.Reset()
	s.input.Focus()
}

// Hide hides the input
func (s *SearchInput) Hide() {
	s.visible = false
	s.input.Blur()
}

// IsVisible returns whether the input is visible
func (s *SearchInput) IsVisible() bool {
	return s.visible
}

// Value returns the current input value
func (s *SearchInput) Value() string {
	return s.input.Value()
}

// Clear clears the input
func (s *SearchInput) Clear() {
	s.input.SetValue("")
}

// SetValue sets the input value
func (s *SearchInput) SetValue(val string) {
	s.input.SetValue(val)
}

// Update handles input events
func (s *SearchInput) Update(msg tea.Msg) (*SearchInput, tea.Cmd) {
	if !s.visible {
		return s, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			s.Hide()
			return s, func() tea.Msg { return FilterClosedMsg{Submitted: false} }

		case key.Matches(msg, theme.DefaultKeyMap().Enter):
			s.Hide()
			return s, func() tea.Msg { return FilterClosedMsg{Submitted: true} }
		}
	}

	prevValue := s.input.Value()
	s.input, cmd = s.input.Update(msg)

	// Check if value changed
	if s.input.Value() != prevValue {
		return s, tea.Batch(cmd, func() tea.Msg {
			return FilterChangedMsg{Value: s.input.Value()}
		})
	}

	return s, cmd
}

// View renders the search input
func (s *SearchInput) View() string {
	if !s.visible {
		return ""
	}

	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	// Styled prefix with accent color (teal/cyan to differentiate from command's ":")
	prefixStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Background(theme.ColorBackground).
		Bold(true)
	prefix := prefixStyle.Render("/")
	space := bgStyle.Render(" ")
	input := s.input.View()

	content := prefix + space + input

	// Pad to full width with styled background
	contentWidth := lipgloss.Width(content)
	if contentWidth < s.width {
		padding := bgStyle.Render(strings.Repeat(" ", s.width-contentWidth))
		content = content + padding
	}

	return content
}
