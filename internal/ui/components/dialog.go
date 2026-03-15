package components

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/ui/theme"
)

// DialogType represents the type of dialog
type DialogType int

const (
	DialogConfirm DialogType = iota
	DialogInfo
	DialogError
	DialogInput
)

// Dialog is a modal dialog component
type Dialog struct {
	dialogType   DialogType
	title        string
	message      string
	width        int
	screenWidth  int
	screenHeight int
	visible      bool
	onConfirm    func()
	onCancel     func()
}

// NewDialog creates a new dialog
func NewDialog() *Dialog {
	return &Dialog{
		width:   50,
		visible: false,
	}
}

// Show shows a confirmation dialog
func (d *Dialog) Show(dialogType DialogType, title, message string) {
	d.dialogType = dialogType
	d.title = title
	d.message = message
	d.visible = true
}

// ShowConfirm shows a confirmation dialog with callbacks
func (d *Dialog) ShowConfirm(title, message string, onConfirm, onCancel func()) {
	d.dialogType = DialogConfirm
	d.title = title
	d.message = message
	d.visible = true
	d.onConfirm = onConfirm
	d.onCancel = onCancel
}

// Hide hides the dialog
func (d *Dialog) Hide() {
	d.visible = false
	d.onConfirm = nil
	d.onCancel = nil
}

// IsVisible returns whether the dialog is visible
func (d *Dialog) IsVisible() bool {
	return d.visible
}

// SetWidth sets the dialog width
func (d *Dialog) SetWidth(width int) {
	d.width = width
}

// Update handles key events
func (d *Dialog) Update(msg tea.Msg) (*Dialog, tea.Cmd) {
	if !d.visible {
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Confirm):
			if d.onConfirm != nil {
				d.onConfirm()
			}
			d.Hide()
			return d, func() tea.Msg { return theme.DialogClosedMsg{Confirmed: true} }

		case key.Matches(msg, theme.DefaultKeyMap().Cancel),
			key.Matches(msg, theme.DefaultKeyMap().Escape):
			if d.onCancel != nil {
				d.onCancel()
			}
			d.Hide()
			return d, func() tea.Msg { return theme.DialogClosedMsg{Confirmed: false} }
		}
	}

	return d, nil
}

// View renders the dialog with hand-drawn borders matching the port forward picker style.
func (d *Dialog) View() string {
	if !d.visible {
		return ""
	}

	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	textStyle := lipgloss.NewStyle().Foreground(theme.ColorText).Background(theme.ColorBackground)
	mutedStyle := lipgloss.NewStyle().Foreground(theme.ColorMuted).Background(theme.ColorBackground)
	borderStyle := lipgloss.NewStyle().Foreground(theme.ColorPrimary).Background(theme.ColorBackground)
	borderChar := borderStyle.Render

	// Title color varies by dialog type
	titleColor := theme.ColorHighlight
	if d.dialogType == DialogError {
		titleColor = theme.ColorError
	}

	// Compute overlay width (similar to port forward picker)
	overlayWidth := d.screenWidth * 2 / 5
	if overlayWidth < 30 {
		overlayWidth = 30
	}
	if overlayWidth > 50 {
		overlayWidth = 50
	}
	if d.screenWidth > 0 && overlayWidth > d.screenWidth-4 {
		overlayWidth = d.screenWidth - 4
	}
	innerWidth := overlayWidth - 2 // subtract border columns

	// Build top border with centered title: ╭──── Title ────╮
	title := lipgloss.NewStyle().
		Foreground(titleColor).
		Background(theme.ColorBackground).
		Bold(true).
		Render(d.title)
	titleWidth := lipgloss.Width(d.title)
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

	// padContent normalizes each content line to exact inner width
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
	lines = append(lines, emptyLine)

	// Word-wrap message to fit content area
	maxMsgWidth := innerWidth - 2
	wrapped := lipgloss.NewStyle().Width(maxMsgWidth).Render(d.message)
	for _, msgLine := range strings.Split(wrapped, "\n") {
		lines = append(lines, padContent(textStyle.Render(msgLine)))
	}

	// Blank separator + footer hints
	lines = append(lines, emptyLine)
	switch d.dialogType {
	case DialogConfirm:
		lines = append(lines, padContent(mutedStyle.Render("enter:confirm  esc:cancel")))
	case DialogInfo, DialogError:
		lines = append(lines, padContent(mutedStyle.Render("enter/esc:dismiss")))
	}

	// Bottom border
	bottomBorder := borderChar("╰") + borderChar(strings.Repeat("─", innerWidth)) + borderChar("╯")
	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

// ViewCentered renders the dialog centered on the screen
func (d *Dialog) ViewCentered(screenWidth, screenHeight int) string {
	if !d.visible {
		return ""
	}

	content := d.View()
	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	// Calculate centering
	padLeft := (screenWidth - contentWidth) / 2
	padTop := (screenHeight - contentHeight) / 2

	if padLeft < 0 {
		padLeft = 0
	}
	if padTop < 0 {
		padTop = 0
	}

	// Build centered content with background fill
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	bgLine := bgStyle.Render(strings.Repeat(" ", screenWidth))
	bgPad := bgStyle.Render(strings.Repeat(" ", padLeft))

	var lines []string
	for i := 0; i < padTop; i++ {
		lines = append(lines, bgLine)
	}

	for _, line := range strings.Split(content, "\n") {
		lines = append(lines, bgPad+line)
	}

	return strings.Join(lines, "\n")
}

// SetSize sets the screen dimensions for overlay rendering
func (d *Dialog) SetSize(width, height int) {
	d.screenWidth = width
	d.screenHeight = height
}

// ViewOverlay composites the dialog box on top of the background content.
// Only the exact dialog area is replaced; background content remains visible
// on both sides and above/below the dialog.
func (d *Dialog) ViewOverlay(background string) string {
	if !d.visible {
		return background
	}
	return OverlayCenter(d.View(), background, d.screenWidth, d.screenHeight)
}
