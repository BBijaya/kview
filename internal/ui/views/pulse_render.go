package views

import (
	"image/color"
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/ui/theme"
)

const (
	sparklineHistoryMax = 30
)

// Sparkline block characters from low to high.
var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// gaugeHeight is the number of lines in each rendered gauge box.
const gaugeHeight = 7

// updateContent assembles the full pulse view content and sets it on the viewport.
func (v *PulseView) updateContent() {
	w := v.viewport.Width()
	if w < 40 {
		w = 40
	}

	// Calculate grid columns: 2-4 based on width
	v.gridCols = w / 21
	if v.gridCols < 2 {
		v.gridCols = 2
	}
	if v.gridCols > 4 {
		v.gridCols = 4
	}

	// Gauge box width: distribute evenly
	boxWidth := (w - (v.gridCols - 1)) / v.gridCols

	var b strings.Builder
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	// Build a multi-line gap column (1-char wide, gaugeHeight lines tall)
	gapLines := make([]string, gaugeHeight)
	for i := range gapLines {
		gapLines[i] = bgStyle.Render(" ")
	}
	gapCol := strings.Join(gapLines, "\n")

	// Render gauge grid
	rows := (gaugeCount + v.gridCols - 1) / v.gridCols
	for row := 0; row < rows; row++ {
		var rowBoxes []string
		for col := 0; col < v.gridCols; col++ {
			idx := row*v.gridCols + col
			if idx >= gaugeCount {
				// Empty cell padding — must be gaugeHeight lines to match
				emptyLines := make([]string, gaugeHeight)
				for i := range emptyLines {
					emptyLines[i] = bgStyle.Render(strings.Repeat(" ", boxWidth))
				}
				rowBoxes = append(rowBoxes, strings.Join(emptyLines, "\n"))
				continue
			}

			isSelected := idx == v.selectedGauge
			var prevGauge GaugeData
			hasPrev := v.hasPrevGauges
			if hasPrev {
				prevGauge = v.prevGauges[idx]
			}
			rowBoxes = append(rowBoxes, v.renderGauge(v.gauges[idx], prevGauge, hasPrev, isSelected, boxWidth))
		}

		// Join gauge boxes side-by-side with 1-char gap columns
		var parts []string
		for i, box := range rowBoxes {
			if i > 0 {
				parts = append(parts, gapCol)
			}
			parts = append(parts, box)
		}
		line := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Blank line before sparklines
	b.WriteString(theme.PadToWidth("", w, theme.ColorBackground))
	b.WriteString("\n")

	// CPU sparkline box
	sparkWidth := w - 4 // leave margin
	if sparkWidth < 20 {
		sparkWidth = 20
	}
	b.WriteString(v.renderSparklineBox("CPU", v.cpuHistory, sparkWidth, theme.ColorAccent, v.lastCPUPct))
	b.WriteString("\n")

	// MEM sparkline box
	b.WriteString(v.renderSparklineBox("MEM", v.memHistory, sparkWidth, theme.ColorSuccess, v.lastMemPct))

	v.viewport.SetContent(b.String())
}

// renderGauge renders a single gauge box (7 lines tall).
func (v *PulseView) renderGauge(g, prev GaugeData, hasPrev, selected bool, boxWidth int) string {
	borderColor := theme.ColorBorder
	if selected {
		borderColor = theme.ColorAccent
	}

	borderStyle := lipgloss.NewStyle().
		Foreground(borderColor).
		Background(theme.ColorBackground)
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorHighlight).
		Background(theme.ColorBackground).
		Bold(true)

	innerWidth := boxWidth - 2 // inside left/right borders
	if innerWidth < 10 {
		innerWidth = 10
	}

	var lines []string

	// Line 1: top border with title
	titleText := " " + g.Name + " "
	titleRendered := titleStyle.Render(titleText)
	titleWidth := lipgloss.Width(titleRendered)
	dashesAfter := innerWidth - titleWidth
	if dashesAfter < 0 {
		dashesAfter = 0
	}
	topLine := borderStyle.Render("┌─") + titleRendered + borderStyle.Render(strings.Repeat("─", dashesAfter)) + borderStyle.Render("┐")
	lines = append(lines, topLine)

	// Lines 2-4: big digit pair (OK / Fault)
	digitLines := renderBigNumberPair(g.OK, g.Fault)

	for row := 0; row < 3; row++ {
		content := centerLine(digitLines[row], 0, innerWidth)
		lines = append(lines, borderStyle.Render("│")+content+borderStyle.Render("│"))
	}

	// Line 5: spacer
	spacer := bgStyle.Render(strings.Repeat(" ", innerWidth))
	lines = append(lines, borderStyle.Render("│")+spacer+borderStyle.Render("│"))

	// Line 6: summary with delta arrows
	okStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSuccess).
		Background(theme.ColorBackground)
	faultStyle := lipgloss.NewStyle().
		Foreground(theme.ColorError).
		Background(theme.ColorBackground)

	summary := okStyle.Render(fmt.Sprintf("OK:%d", g.OK)) +
		bgStyle.Render("  ") +
		faultStyle.Render(fmt.Sprintf("Fault:%d", g.Fault))

	// Delta arrows
	if hasPrev {
		totalCurr := g.OK + g.Fault
		totalPrev := prev.OK + prev.Fault
		arrow := deltaArrow(totalCurr, totalPrev)
		if arrow != "" {
			summary += bgStyle.Render(" ") + arrow
		}
	}

	summaryWidth := lipgloss.Width(summary)
	leftPad := 1
	rightPad := innerWidth - summaryWidth - leftPad
	if rightPad < 0 {
		rightPad = 0
	}
	summaryLine := borderStyle.Render("│") +
		bgStyle.Render(strings.Repeat(" ", leftPad)) +
		summary +
		bgStyle.Render(strings.Repeat(" ", rightPad)) +
		borderStyle.Render("│")
	lines = append(lines, summaryLine)

	// Line 7: bottom border
	bottomLine := borderStyle.Render("└") + borderStyle.Render(strings.Repeat("─", innerWidth)) + borderStyle.Render("┘")
	lines = append(lines, bottomLine)

	return strings.Join(lines, "\n")
}

// renderSparkline renders a sparkline from values 0-100 using block characters.
func renderSparkline(values []float64, width int, color color.Color) string {
	style := lipgloss.NewStyle().
		Foreground(color).
		Background(theme.ColorBackground)
	dimStyle := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Background(theme.ColorBackground)

	if len(values) == 0 {
		return dimStyle.Render(strings.Repeat("▁", width))
	}

	// Take the last `width` values
	start := 0
	if len(values) > width {
		start = len(values) - width
	}
	visible := values[start:]

	var b strings.Builder
	// Pad with empty blocks if not enough history
	emptyCount := width - len(visible)
	if emptyCount > 0 {
		b.WriteString(dimStyle.Render(strings.Repeat("▁", emptyCount)))
	}

	for _, v := range visible {
		idx := int(math.Round(v / 100.0 * float64(len(sparkBlocks)-1)))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkBlocks) {
			idx = len(sparkBlocks) - 1
		}
		b.WriteString(style.Render(string(sparkBlocks[idx])))
	}

	return b.String()
}

// renderSparklineBox renders a bordered sparkline box with title and current percentage.
func (v *PulseView) renderSparklineBox(title string, values []float64, sparkWidth int, color color.Color, currentPct int) string {
	borderStyle := lipgloss.NewStyle().
		Foreground(theme.ColorBorder).
		Background(theme.ColorBackground)
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorHighlight).
		Background(theme.ColorBackground).
		Bold(true)
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	pctStyle := lipgloss.NewStyle().
		Foreground(color).
		Background(theme.ColorBackground).
		Bold(true)

	// Inner width
	innerWidth := sparkWidth
	if innerWidth < 20 {
		innerWidth = 20
	}

	// Top border with title
	titleText := " " + title + " "
	titleRendered := titleStyle.Render(titleText)
	titleVisWidth := lipgloss.Width(titleRendered)
	dashesAfter := innerWidth - titleVisWidth
	if dashesAfter < 0 {
		dashesAfter = 0
	}
	topLine := borderStyle.Render("┌─") + titleRendered + borderStyle.Render(strings.Repeat("─", dashesAfter)) + borderStyle.Render("┐")

	// Content line: sparkline + percentage
	pctStr := fmt.Sprintf(" %d%%", currentPct)
	pctRendered := pctStyle.Render(pctStr)
	pctWidth := lipgloss.Width(pctRendered)
	sparklineWidth := innerWidth - pctWidth - 1 // 1 for left margin
	if sparklineWidth < 5 {
		sparklineWidth = 5
	}
	sparklineStr := renderSparkline(values, sparklineWidth, color)
	sparkRenderedWidth := lipgloss.Width(sparklineStr) + pctWidth + 1
	rightPad := innerWidth - sparkRenderedWidth
	if rightPad < 0 {
		rightPad = 0
	}
	contentLine := borderStyle.Render("│") +
		bgStyle.Render(" ") +
		sparklineStr +
		pctRendered +
		bgStyle.Render(strings.Repeat(" ", rightPad)) +
		borderStyle.Render("│")

	// Bottom border
	bottomLine := borderStyle.Render("└") + borderStyle.Render(strings.Repeat("─", innerWidth)) + borderStyle.Render("┘")

	return topLine + "\n" + contentLine + "\n" + bottomLine
}
