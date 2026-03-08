package components

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/ui/theme"
)

// ScalePickerConfirmMsg is sent when the scale action is confirmed
type ScalePickerConfirmMsg struct {
	Namespace string
	Name      string
	Kind      string // "deployments" or "statefulsets"
	Replicas  int
}

// ScalePickerCancelMsg is sent when the scale picker is cancelled
type ScalePickerCancelMsg struct{}

// ScalePicker is an overlay form for scaling deployments and statefulsets.
// Single editable field for replica count.
type ScalePicker struct {
	visible         bool
	width, height   int
	namespace       string
	name            string
	kind            string // "deployments" or "statefulsets"
	currentReplicas int32
	replicaInput    textinput.Model
}

// NewScalePicker creates a new scale picker
func NewScalePicker() *ScalePicker {
	replicaInput := newStyledInput("replicas", 5, 10)
	replicaInput.Validate = digitsOnly
	return &ScalePicker{
		replicaInput: replicaInput,
	}
}

// Show shows the picker with the given resource info
func (p *ScalePicker) Show(namespace, name, kind string, currentReplicas int32) {
	p.visible = true
	p.namespace = namespace
	p.name = name
	p.kind = kind
	p.currentReplicas = currentReplicas
	p.replicaInput.SetValue(fmt.Sprintf("%d", currentReplicas))
	p.replicaInput.Focus()
}

// Hide hides the picker
func (p *ScalePicker) Hide() {
	p.visible = false
	p.replicaInput.Blur()
}

// IsVisible returns whether the picker is visible
func (p *ScalePicker) IsVisible() bool {
	return p.visible
}

// SetSize sets the overlay dimensions
func (p *ScalePicker) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Update handles messages for the picker
func (p *ScalePicker) Update(msg tea.Msg) (*ScalePicker, tea.Cmd) {
	if !p.visible {
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			p.Hide()
			return p, func() tea.Msg { return ScalePickerCancelMsg{} }

		case "enter":
			val := p.replicaInput.Value()
			if val == "" {
				return p, nil
			}
			replicas, err := strconv.Atoi(val)
			if err != nil || replicas < 0 {
				return p, nil
			}
			p.Hide()
			return p, func() tea.Msg {
				return ScalePickerConfirmMsg{
					Namespace: p.namespace,
					Name:      p.name,
					Kind:      p.kind,
					Replicas:  replicas,
				}
			}

		default:
			var cmd tea.Cmd
			p.replicaInput, cmd = p.replicaInput.Update(msg)
			return p, cmd
		}
	}

	return p, nil
}

// renderBox builds the styled overlay box.
func (p *ScalePicker) renderBox() string {
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	mutedStyle := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Background(theme.ColorBackground)

	focusLabelStyle := lipgloss.NewStyle().
		Foreground(theme.ColorHighlight).
		Background(theme.ColorBackground)

	overlayWidth := p.width * 2 / 5 // 40% of terminal width
	if overlayWidth < 40 {
		overlayWidth = 40
	}
	if overlayWidth > 70 {
		overlayWidth = 70
	}
	if p.width > 0 && overlayWidth > p.width-4 {
		overlayWidth = p.width - 4
	}
	innerWidth := overlayWidth - 2 // subtract border columns

	borderStyle := lipgloss.NewStyle().Foreground(theme.ColorPrimary).Background(theme.ColorBackground)
	borderChar := borderStyle.Render

	// Build top border with centered title
	kindLabel := "Deployment"
	if p.kind == "statefulsets" || p.kind == "statefulset" {
		kindLabel = "StatefulSet"
	}
	titleText := "Scale " + kindLabel
	title := lipgloss.NewStyle().
		Foreground(theme.ColorHighlight).
		Background(theme.ColorBackground).
		Bold(true).
		Render(titleText)
	titleWidth := lipgloss.Width(titleText)
	dashSpace := innerWidth - titleWidth - 2 // 2 for spaces around title
	if dashSpace < 2 {
		dashSpace = 2
	}
	leftDashes := dashSpace / 2
	rightDashes := dashSpace - leftDashes
	topBorder := borderChar("╭") +
		borderChar(strings.Repeat("─", leftDashes)) +
		borderChar(" ") + title + borderChar(" ") +
		borderChar(strings.Repeat("─", rightDashes)) +
		borderChar("╮")

	// Build content lines
	padContent := func(line string) string {
		w := lipgloss.Width(line)
		pad := innerWidth - 2 // inner padding (1 each side)
		if w < pad {
			line += bgStyle.Render(strings.Repeat(" ", pad-w))
		} else if w > pad {
			line = ansiTruncateClean(line, pad)
		}
		return borderChar("│") + bgStyle.Render(" ") + line + bgStyle.Render(" ") + borderChar("│")
	}
	emptyLine := borderChar("│") + bgStyle.Render(strings.Repeat(" ", innerWidth)) + borderChar("│")

	var lines []string
	lines = append(lines, topBorder)

	// Namespace / resource name
	maxContentWidth := innerWidth - 2
	nsResource := p.namespace + "/" + p.name
	if len(nsResource) > maxContentWidth {
		nsResource = nsResource[:maxContentWidth-1] + "…"
	}
	lines = append(lines, padContent(mutedStyle.Render(nsResource)))

	// Current replicas
	lines = append(lines, padContent(mutedStyle.Render(fmt.Sprintf("  current: %d replicas", p.currentReplicas))))

	// Blank separator
	lines = append(lines, emptyLine)

	// Input field
	line := focusLabelStyle.Render("Replicas:") + " " + p.replicaInput.View()
	lines = append(lines, padContent(line))

	// Blank separator + footer
	lines = append(lines, emptyLine)
	lines = append(lines, padContent(mutedStyle.Render("enter:confirm  esc:cancel")))

	// Bottom border
	bottomBorder := borderChar("╰") + borderChar(strings.Repeat("─", innerWidth)) + borderChar("╯")
	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

// ViewOverlay composites the picker box on top of the background content.
func (p *ScalePicker) ViewOverlay(background string) string {
	if !p.visible {
		return background
	}
	return OverlayCenter(p.renderBox(), background, p.width, p.height)
}
