package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/bijaya/kview/internal/ui/theme"
)

// Overlay composites box on top of background at the given position within the
// screen dimensions. Background content remains visible around the box.
func Overlay(box, background string, padLeft, padTop, screenWidth, screenHeight int) string {
	boxLines := strings.Split(box, "\n")
	bgLines := strings.Split(background, "\n")

	boxHeight := len(boxLines)
	boxWidth := lipgloss.Width(box)

	if padLeft < 0 {
		padLeft = 0
	}
	if padTop < 0 {
		padTop = 0
	}

	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	// Ensure bgLines covers full height
	for len(bgLines) < screenHeight {
		bgLines = append(bgLines, bgStyle.Render(strings.Repeat(" ", screenWidth)))
	}

	result := make([]string, len(bgLines))
	for i, bgLine := range bgLines {
		if i >= padTop && i < padTop+boxHeight {
			overlayIdx := i - padTop
			// Left: keep background content up to padLeft, with clean reset
			left := ansiTruncateClean(bgLine, padLeft)
			leftWidth := lipgloss.Width(left)
			if leftWidth < padLeft {
				left += bgStyle.Render(strings.Repeat(" ", padLeft-leftWidth))
			}
			// Right: keep background content after padLeft+boxWidth
			right := ansiSkipLeft(bgLine, padLeft+boxWidth)
			line := left + boxLines[overlayIdx] + right
			// Normalize width to prevent Bubble Tea re-render artifacts
			lineWidth := lipgloss.Width(line)
			if lineWidth < screenWidth {
				line += bgStyle.Render(strings.Repeat(" ", screenWidth-lineWidth))
			} else if lineWidth > screenWidth {
				line = ansiTruncateClean(line, screenWidth)
			}
			result[i] = line
		} else {
			result[i] = bgLine
		}
	}

	return strings.Join(result, "\n")
}

// OverlayCenter composites box centered on top of background within the given
// screen dimensions. Background content remains visible around the box.
func OverlayCenter(box, background string, screenWidth, screenHeight int) string {
	boxHeight := len(strings.Split(box, "\n"))
	boxWidth := lipgloss.Width(box)

	padLeft := (screenWidth - boxWidth) / 2
	padTop := (screenHeight - boxHeight) / 2

	return Overlay(box, background, padLeft, padTop, screenWidth, screenHeight)
}

// ansiTruncateClean truncates an ANSI string to the given visible width and
// appends an SGR reset so that no trailing escape sequences from beyond the
// cut point can bleed into subsequent content.
func ansiTruncateClean(s string, width int) string {
	// ansi.Truncate preserves ALL escape sequences (even past the cut),
	// which causes style bleed. We rebuild using DecodeSequence instead.
	var buf strings.Builder
	var state byte
	cur := 0
	for len(s) > 0 {
		seq, w, n, newState := ansi.DecodeSequence(s, state, nil)
		state = newState
		if w > 0 {
			if cur+w > width {
				break
			}
			cur += w
		}
		buf.WriteString(seq)
		s = s[n:]
	}
	buf.WriteString("\033[0m")
	return buf.String()
}

// ansiSkipLeft returns the portion of an ANSI-encoded string starting after
// 'skip' visible columns. A leading SGR reset is emitted so the returned
// text starts with a clean state; subsequent styled content carries its own
// sequences.
func ansiSkipLeft(s string, skip int) string {
	var buf strings.Builder
	buf.WriteString("\033[0m")
	var state byte
	cur := 0
	for len(s) > 0 {
		seq, width, n, newState := ansi.DecodeSequence(s, state, nil)
		state = newState
		if cur >= skip {
			buf.WriteString(seq)
		} else if width > 0 {
			cur += width
		}
		// ANSI sequences (width==0) in the skipped region are discarded
		s = s[n:]
	}
	return buf.String()
}
