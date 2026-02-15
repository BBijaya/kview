package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/ui/theme"
)

// bigDigits contains 3x3 box-drawing character grids for digits 0-9.
// Each digit is 3 characters wide and 3 lines tall.
var bigDigits = [10][3]string{
	{"┏━┓", "┃ ┃", "┗━┛"}, // 0
	{" ╻ ", " ┃ ", " ╹ "}, // 1
	{"╺━┓", "┏━┛", "┗━╸"}, // 2
	{"╺━┓", " ━┫", "╺━┛"}, // 3
	{"╻ ╻", "┗━┫", "  ╹"}, // 4
	{"┏━╸", "┗━┓", "╺━┛"}, // 5
	{"┏━╸", "┣━┓", "┗━┛"}, // 6
	{"╺━┓", "  ┃", "  ╹"}, // 7
	{"┏━┓", "┣━┫", "┗━┛"}, // 8
	{"┏━┓", "┗━┫", "╺━┛"}, // 9
}

// renderBigNumber renders an integer as big 3x3 box-drawing digits.
// maxDigits controls zero-padding (leading zeros rendered in muted color).
// Returns 3 styled lines.
func renderBigNumber(n int, color lipgloss.TerminalColor, maxDigits int) [3]string {
	if n < 0 {
		n = 0
	}

	numStr := fmt.Sprintf("%d", n)
	if len(numStr) > maxDigits {
		maxDigits = len(numStr)
	}

	// Pad with leading zeros
	padded := fmt.Sprintf("%0*d", maxDigits, n)

	digitStyle := lipgloss.NewStyle().
		Foreground(color).
		Background(theme.ColorBackground)
	dimStyle := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Background(theme.ColorBackground)
	gapStyle := lipgloss.NewStyle().
		Background(theme.ColorBackground)
	gap := gapStyle.Render(" ")

	var lines [3]string
	leadingZero := true
	for i, ch := range padded {
		d := int(ch - '0')
		if d != 0 || i == len(padded)-1 {
			leadingZero = false
		}
		style := digitStyle
		if leadingZero && i < len(padded)-1 {
			style = dimStyle
		}
		for row := 0; row < 3; row++ {
			if i > 0 {
				lines[row] += gap
			}
			lines[row] += style.Render(bigDigits[d][row])
		}
	}

	return lines
}

// deltaArrow returns a styled arrow indicating change direction.
func deltaArrow(current, prev int) string {
	if current > prev {
		return lipgloss.NewStyle().
			Foreground(theme.ColorSuccess).
			Background(theme.ColorBackground).
			Render("↑")
	}
	if current < prev {
		return lipgloss.NewStyle().
			Foreground(theme.ColorError).
			Background(theme.ColorBackground).
			Render("↓")
	}
	return ""
}

// separatorDot returns a styled vertical separator between OK and Fault digits.
func separatorDot() [3]string {
	style := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Background(theme.ColorBackground)
	return [3]string{
		style.Render(" "),
		style.Render("⠔"),
		style.Render(" "),
	}
}

// maxDigitsForCount returns the number of digits needed to display a count.
func maxDigitsForCount(n int) int {
	if n < 10 {
		return 1
	}
	s := fmt.Sprintf("%d", n)
	return len(s)
}

// renderBigNumberPair renders OK (green) and Fault (red) counts side by side
// with a separator dot between them. Returns 3 lines.
func renderBigNumberPair(ok, fault int) [3]string {
	// Determine max digits: at least 2, or whatever fits the larger number
	maxDig := 2
	total := ok + fault
	if d := maxDigitsForCount(total); d > maxDig {
		maxDig = d
	}
	if d := maxDigitsForCount(ok); d > maxDig {
		maxDig = d
	}
	if d := maxDigitsForCount(fault); d > maxDig {
		maxDig = d
	}

	okLines := renderBigNumber(ok, theme.ColorSuccess, maxDig)
	sep := separatorDot()
	faultLines := renderBigNumber(fault, theme.ColorError, maxDig)

	var lines [3]string
	for row := 0; row < 3; row++ {
		lines[row] = okLines[row] + sep[row] + faultLines[row]
	}
	return lines
}

// centerLine centers content within the given width using background padding.
// Uses lipgloss.Width to measure actual visual width of styled content.
func centerLine(content string, _ int, totalWidth int) string {
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	actualWidth := lipgloss.Width(content)
	leftPad := (totalWidth - actualWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	rightPad := totalWidth - actualWidth - leftPad
	if rightPad < 0 {
		rightPad = 0
	}
	return bgStyle.Render(strings.Repeat(" ", leftPad)) + content + bgStyle.Render(strings.Repeat(" ", rightPad))
}
