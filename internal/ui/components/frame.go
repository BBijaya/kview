package components

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/ui/theme"
)

// Frame wraps content in a bordered frame with optional title
type Frame struct {
	width      int
	height     int
	title      string
	version    string
	showBorder bool
	borderStyle lipgloss.Style
}

// NewFrame creates a new frame component
func NewFrame() *Frame {
	return &Frame{
		width:      80,
		height:     24,
		title:      "kview",
		version:    "v0.1.0",
		showBorder: true,
		borderStyle: lipgloss.NewStyle().Foreground(theme.ColorFrameBorder).Background(theme.ColorBackground),
	}
}

// SetSize sets the frame dimensions
func (f *Frame) SetSize(width, height int) {
	f.width = width
	f.height = height
}

// SetTitle sets the frame title
func (f *Frame) SetTitle(title string) {
	f.title = title
}

// SetVersion sets the version string
func (f *Frame) SetVersion(version string) {
	f.version = version
}

// BorderRender returns the border style's Render function for wrapping border characters.
func (f *Frame) BorderRender() func(strs ...string) string {
	return f.borderStyle.Render
}

// InnerWidth returns the usable width inside the frame
func (f *Frame) InnerWidth() int {
	if f.showBorder {
		return f.width - 2 // Account for left and right borders
	}
	return f.width
}

// InnerHeight returns the usable height inside the frame
func (f *Frame) InnerHeight() int {
	if f.showBorder {
		return f.height - 2 // Account for top and bottom borders
	}
	return f.height
}

// Wrap wraps content in the frame border with title and version
func (f *Frame) Wrap(content string) string {
	if !f.showBorder {
		return content
	}

	innerWidth := f.InnerWidth()
	topBorder := f.buildTopBorder(innerWidth)
	return f.wrapWithTopBorder(content, topBorder, false)
}

// WrapPlain wraps content in a plain frame border (no title/version)
func (f *Frame) WrapPlain(content string) string {
	if !f.showBorder {
		return content
	}

	innerWidth := f.InnerWidth()
	borderChar := f.borderStyle.Render
	topBorder := borderChar("╭") + borderChar(strings.Repeat("─", innerWidth)) + borderChar("╮")
	return f.wrapWithTopBorder(content, topBorder, false)
}

// WrapWithCenteredLabel wraps content with a centered label in the top border
// e.g. ╭──── Pods(all)[25] ────╮
func (f *Frame) WrapWithCenteredLabel(content, label string) string {
	if !f.showBorder {
		return content
	}

	innerWidth := f.InnerWidth()
	borderChar := f.borderStyle.Render
	labelWidth := lipgloss.Width(label)

	// Calculate dash widths for centering
	dashSpace := innerWidth - labelWidth - 2 // 2 for spaces around label
	if dashSpace < 2 {
		// Not enough room, fall back to plain border
		topBorder := borderChar("╭") + borderChar(strings.Repeat("─", innerWidth)) + borderChar("╮")
		return f.wrapWithTopBorder(content, topBorder, false)
	}
	leftDashes := dashSpace / 2
	rightDashes := dashSpace - leftDashes

	topBorder := borderChar("╭") +
		borderChar(strings.Repeat("─", leftDashes)) +
		borderChar(" ") +
		label +
		borderChar(" ") +
		borderChar(strings.Repeat("─", rightDashes)) +
		borderChar("╮")

	return f.wrapWithTopBorder(content, topBorder, true)
}

// wrapWithTopBorder is the shared logic for Wrap and WrapPlain.
// If prePadded is true, skip per-line width measurement (content already padded to inner width).
func (f *Frame) wrapWithTopBorder(content string, topBorder string, prePadded bool) string {
	innerWidth := f.InnerWidth()
	borderChar := f.borderStyle.Render

	// Create a style for filling content with background
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	// Split content into lines and ensure they fit
	lines := strings.Split(content, "\n")

	// Pad or truncate lines to fit inner width with background
	formattedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		if !prePadded {
			lineWidth := lipgloss.Width(line)
			if lineWidth < innerWidth {
				// Pad to full width with background color
				padding := bgStyle.Render(strings.Repeat(" ", innerWidth-lineWidth))
				line = line + padding
			} else if lineWidth > innerWidth {
				// Truncate - this is a simplistic approach
				line = theme.TruncateString(line, innerWidth)
			}
		}
		formattedLines = append(formattedLines, borderChar("│")+line+borderChar("│"))
	}

	// Fill remaining height with empty lines with background
	contentHeight := len(formattedLines)
	targetHeight := f.InnerHeight()
	emptyContent := bgStyle.Render(strings.Repeat(" ", innerWidth))
	emptyLine := borderChar("│") + emptyContent + borderChar("│")
	for contentHeight < targetHeight {
		formattedLines = append(formattedLines, emptyLine)
		contentHeight++
	}

	// Truncate if too many lines
	if len(formattedLines) > targetHeight {
		formattedLines = formattedLines[:targetHeight]
	}

	// Build the bottom border
	bottomBorder := borderChar("╰") + borderChar(strings.Repeat("─", innerWidth)) + borderChar("╯")

	// Combine all parts
	var result strings.Builder
	result.WriteString(topBorder)
	result.WriteString("\n")
	result.WriteString(strings.Join(formattedLines, "\n"))
	result.WriteString("\n")
	result.WriteString(bottomBorder)

	return result.String()
}

// buildTopBorder creates the top border with embedded title and version
func (f *Frame) buildTopBorder(innerWidth int) string {
	borderChar := f.borderStyle.Render
	title := theme.Styles.FrameTitle.Render(f.title)
	version := theme.Styles.FrameVersion.Render(f.version)

	titleLen := lipgloss.Width(f.title)
	versionLen := lipgloss.Width(f.version)

	// Calculate available space for dashes
	// Format: ╭─ title ───────────────────────── version ─╮
	leftDashCount := 1
	rightDashCount := 1
	middleDashCount := innerWidth - titleLen - versionLen - leftDashCount - rightDashCount - 4 // 4 for spaces around title and version

	if middleDashCount < 1 {
		middleDashCount = 1
	}

	var border strings.Builder
	border.WriteString(borderChar("╭"))
	border.WriteString(borderChar(strings.Repeat("─", leftDashCount)))
	border.WriteString(borderChar(" "))
	border.WriteString(title)
	border.WriteString(borderChar(" "))
	border.WriteString(borderChar(strings.Repeat("─", middleDashCount)))
	border.WriteString(borderChar(" "))
	border.WriteString(version)
	border.WriteString(borderChar(" "))
	border.WriteString(borderChar(strings.Repeat("─", rightDashCount)))
	border.WriteString(borderChar("╮"))

	return border.String()
}

// HorizontalDivider returns a horizontal divider line for inside the frame
func (f *Frame) HorizontalDivider() string {
	borderChar := f.borderStyle.Render
	innerWidth := f.InnerWidth()
	return borderChar("├") + borderChar(strings.Repeat("─", innerWidth)) + borderChar("┤")
}

// HorizontalDividerPlain returns a horizontal divider without frame connectors
func (f *Frame) HorizontalDividerPlain() string {
	borderChar := f.borderStyle.Render
	innerWidth := f.InnerWidth()
	return borderChar(strings.Repeat("─", innerWidth+2))
}

// HorizontalDividerStyled returns a styled horizontal divider with background
func (f *Frame) HorizontalDividerStyled() string {
	dividerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorBorder).
		Background(theme.ColorBackground)
	innerWidth := f.InnerWidth()
	divider := dividerStyle.Width(innerWidth).Render(strings.Repeat("─", innerWidth))
	return f.borderStyle.Render("├") + divider + f.borderStyle.Render("┤")
}

// EmptyLineWithBackground returns an empty line with solid background
func (f *Frame) EmptyLineWithBackground() string {
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	innerWidth := f.InnerWidth()
	return bgStyle.Render(strings.Repeat(" ", innerWidth))
}

// ThinDivider returns a thin divider line with background color (no box connectors)
// This is used to separate info pane from content pane with minimal visual noise
func (f *Frame) ThinDivider() string {
	innerWidth := f.InnerWidth()
	return theme.Styles.ThinDivider.Width(innerWidth).Render(strings.Repeat("─", innerWidth))
}
