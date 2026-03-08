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

// View renders the dialog
func (d *Dialog) View() string {
	if !d.visible {
		return ""
	}

	var b strings.Builder

	// Title
	titleStyle := theme.Styles.DialogTitle
	switch d.dialogType {
	case DialogError:
		titleStyle = titleStyle.Foreground(theme.ColorError)
	case DialogInfo:
		titleStyle = titleStyle.Foreground(theme.ColorInfo)
	}
	b.WriteString(titleStyle.Render(d.title))
	b.WriteString("\n\n")

	// Message
	messageStyle := theme.Styles.Base.Width(d.width - 4)
	b.WriteString(messageStyle.Render(d.message))
	b.WriteString("\n\n")

	// Buttons
	switch d.dialogType {
	case DialogConfirm:
		yesBtn := theme.Styles.TabActive.Render(" Yes (y) ")
		noBtn := theme.Styles.Tab.Render(" No (n/esc) ")
		b.WriteString(yesBtn + "  " + noBtn)
	case DialogInfo, DialogError:
		okBtn := theme.Styles.TabActive.Render(" OK (enter/esc) ")
		b.WriteString(okBtn)
	}

	// Wrap in dialog style
	return theme.Styles.Dialog.Width(d.width).Render(b.String())
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
