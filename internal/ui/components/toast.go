package components

import (
	"image/color"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/ui/theme"
)

// ToastType represents the type of toast notification
type ToastType int

const (
	ToastInfo ToastType = iota
	ToastSuccess
	ToastWarning
	ToastError
)

// Toast represents a single toast notification
type Toast struct {
	ID        int
	Type      ToastType
	Title     string
	Message   string
	CreatedAt time.Time
	Duration  time.Duration
}

// ToastExpiredMsg is sent when a toast expires
type ToastExpiredMsg struct {
	ID int
}

// ToastStack manages a stack of toast notifications
type ToastStack struct {
	toasts    []Toast
	nextID    int
	maxToasts int
	width     int
	height    int
}

// NewToastStack creates a new toast stack
func NewToastStack() *ToastStack {
	return &ToastStack{
		toasts:    []Toast{},
		nextID:    1,
		maxToasts: 5,
	}
}

// Push adds a new toast to the stack
func (t *ToastStack) Push(toastType ToastType, title, message string, duration time.Duration) tea.Cmd {
	toast := Toast{
		ID:        t.nextID,
		Type:      toastType,
		Title:     title,
		Message:   message,
		CreatedAt: time.Now(),
		Duration:  duration,
	}
	t.nextID++

	// Add to front
	t.toasts = append([]Toast{toast}, t.toasts...)

	// Limit stack size
	if len(t.toasts) > t.maxToasts {
		t.toasts = t.toasts[:t.maxToasts]
	}

	// Return expiration command
	id := toast.ID
	return tea.Tick(duration, func(time.Time) tea.Msg {
		return ToastExpiredMsg{ID: id}
	})
}

// PushInfo adds an info toast
func (t *ToastStack) PushInfo(title, message string) tea.Cmd {
	return t.Push(ToastInfo, title, message, 3*time.Second)
}

// PushSuccess adds a success toast
func (t *ToastStack) PushSuccess(title, message string) tea.Cmd {
	return t.Push(ToastSuccess, title, message, 3*time.Second)
}

// PushWarning adds a warning toast
func (t *ToastStack) PushWarning(title, message string) tea.Cmd {
	return t.Push(ToastWarning, title, message, 5*time.Second)
}

// PushError adds an error toast
func (t *ToastStack) PushError(title, message string) tea.Cmd {
	return t.Push(ToastError, title, message, 7*time.Second)
}

// Remove removes a toast by ID
func (t *ToastStack) Remove(id int) {
	for i, toast := range t.toasts {
		if toast.ID == id {
			t.toasts = append(t.toasts[:i], t.toasts[i+1:]...)
			return
		}
	}
}

// Clear removes all toasts
func (t *ToastStack) Clear() {
	t.toasts = []Toast{}
}

// SetSize sets the available width and height
func (t *ToastStack) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// Count returns the number of active toasts
func (t *ToastStack) Count() int {
	return len(t.toasts)
}

// Update handles toast messages
func (t *ToastStack) Update(msg tea.Msg) (*ToastStack, tea.Cmd) {
	switch msg := msg.(type) {
	case ToastExpiredMsg:
		t.Remove(msg.ID)
	}
	return t, nil
}

// View renders the toast stack as a standalone box (no positioning).
func (t *ToastStack) View() string {
	if len(t.toasts) == 0 {
		return ""
	}

	toastWidth := 40
	for _, toast := range t.toasts {
		if strings.Contains(toast.Message, "\n") {
			toastWidth = 50
			break
		}
	}
	if t.width > 0 && toastWidth > t.width-10 {
		toastWidth = t.width - 10
	}

	var toastViews []string

	for _, toast := range t.toasts {
		toastViews = append(toastViews, t.renderToast(toast, toastWidth))
	}

	box := strings.Join(toastViews, "\n")

	// Normalize all lines to the same width so the Overlay function
	// can position right-side background content at the correct column.
	// Lipgloss should produce consistent widths, but Unicode ambiguous-width
	// characters (icons ✓/⚠/✗/ℹ) can cause per-line discrepancies between
	// what lipgloss.Width reports and what the terminal renders.
	lines := strings.Split(box, "\n")
	maxWidth := 0
	for _, line := range lines {
		if w := lipgloss.Width(line); w > maxWidth {
			maxWidth = w
		}
	}
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	for i, line := range lines {
		if w := lipgloss.Width(line); w < maxWidth {
			lines[i] = line + bgStyle.Render(strings.Repeat(" ", maxWidth-w))
		}
	}

	return strings.Join(lines, "\n")
}

// ViewOverlay composites the toast stack onto background in the bottom-right.
func (t *ToastStack) ViewOverlay(background string) string {
	box := t.View()
	if box == "" {
		return background
	}

	boxWidth := lipgloss.Width(box)
	boxHeight := lipgloss.Height(box)

	padLeft := t.width - boxWidth - 2
	padTop := t.height - boxHeight - 2

	return Overlay(box, background, padLeft, padTop, t.width, t.height)
}

// renderToast renders a single toast
func (t *ToastStack) renderToast(toast Toast, width int) string {
	// Get style based on type
	var borderColor color.Color
	var icon string

	switch toast.Type {
	case ToastSuccess:
		borderColor = theme.ColorSuccess
		icon = "✓"
	case ToastWarning:
		borderColor = theme.ColorWarning
		icon = "⚠"
	case ToastError:
		borderColor = theme.ColorError
		icon = "✗"
	default:
		borderColor = theme.ColorInfo
		icon = "ℹ"
	}

	toastStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		BorderBackground(theme.ColorBackground).
		Background(theme.ColorBackground).
		Padding(0, 1).
		Width(width)

	titleStyle := lipgloss.NewStyle().
		Foreground(borderColor).
		Background(theme.ColorBackground).
		Bold(true)

	messageStyle := lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Background(theme.ColorBackground)

	// Build content
	var content strings.Builder
	content.WriteString(titleStyle.Render(icon + " " + toast.Title))
	if toast.Message != "" {
		content.WriteString("\n")
		var renderedMsg string
		if strings.Contains(toast.Message, "\n") {
			renderedMsg = toast.Message // Pre-formatted, preserve layout
		} else {
			renderedMsg = wrapText(toast.Message, width-4)
		}
		content.WriteString(messageStyle.Render(renderedMsg))
	}

	return toastStyle.Render(content.String())
}

// wrapText wraps text to a maximum width
func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if currentLine.Len()+1+len(word) <= maxWidth {
			currentLine.WriteString(" ")
			currentLine.WriteString(word)
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
}
