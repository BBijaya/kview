package components

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/ui/theme"
)

// Spinner wraps the bubbles spinner with additional functionality
type Spinner struct {
	spinner spinner.Model
	message string
	visible bool
	style   lipgloss.Style
}

// NewSpinner creates a new spinner with a default message
func NewSpinner() *Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorPrimary)

	return &Spinner{
		spinner: s,
		message: "Loading...",
		visible: false,
		style: lipgloss.NewStyle().
			Foreground(theme.ColorMuted),
	}
}

// NewSpinnerWithStyle creates a spinner with a custom spinner style
func NewSpinnerWithStyle(spinnerType spinner.Spinner) *Spinner {
	s := spinner.New()
	s.Spinner = spinnerType
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorPrimary)

	return &Spinner{
		spinner: s,
		message: "Loading...",
		visible: false,
		style: lipgloss.NewStyle().
			Foreground(theme.ColorMuted),
	}
}

// Show makes the spinner visible and starts animation
func (s *Spinner) Show() tea.Cmd {
	s.visible = true
	return s.spinner.Tick
}

// Hide makes the spinner invisible
func (s *Spinner) Hide() {
	s.visible = false
}

// SetMessage updates the spinner message
func (s *Spinner) SetMessage(msg string) {
	s.message = msg
}

// IsVisible returns whether the spinner is visible
func (s *Spinner) IsVisible() bool {
	return s.visible
}

// Tick handles spinner animation updates
func (s *Spinner) Tick() tea.Cmd {
	if !s.visible {
		return nil
	}
	return s.spinner.Tick
}

// Update handles spinner messages
func (s *Spinner) Update(msg tea.Msg) (*Spinner, tea.Cmd) {
	if !s.visible {
		return s, nil
	}

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	}

	return s, nil
}

// View renders the spinner
func (s *Spinner) View() string {
	if !s.visible {
		return ""
	}

	return s.spinner.View() + " " + s.style.Render(s.message)
}

// ViewCentered renders the spinner centered in the given width with background.
// Uses lipgloss.Place() instead of style.Width().Height() to ensure all
// padding (including vertical) gets proper background styling.
func (s *Spinner) ViewCentered(width, height int) string {
	if !s.visible {
		return ""
	}

	content := s.View()

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		content,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(theme.ColorBackground)),
	)
}

// SpinnerType represents different spinner animation styles
type SpinnerType int

const (
	SpinnerDot SpinnerType = iota
	SpinnerLine
	SpinnerMiniDot
	SpinnerJump
	SpinnerPulse
	SpinnerPoints
	SpinnerGlobe
	SpinnerMoon
	SpinnerMonkey
)

// GetSpinnerStyle returns the bubbles spinner style for the given type
func GetSpinnerStyle(t SpinnerType) spinner.Spinner {
	switch t {
	case SpinnerLine:
		return spinner.Line
	case SpinnerMiniDot:
		return spinner.MiniDot
	case SpinnerJump:
		return spinner.Jump
	case SpinnerPulse:
		return spinner.Pulse
	case SpinnerPoints:
		return spinner.Points
	case SpinnerGlobe:
		return spinner.Globe
	case SpinnerMoon:
		return spinner.Moon
	case SpinnerMonkey:
		return spinner.Monkey
	default:
		return spinner.Dot
	}
}
