package views

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/analyzer"
	"github.com/bijaya/kview/internal/ui/theme"
)

// clampCursor constrains a cursor value to valid bounds [0, count-1]
func clampCursor(val, count int) int {
	if count <= 0 {
		return 0
	}
	if val < 0 {
		return 0
	}
	if val >= count {
		return count - 1
	}
	return val
}

// plural returns "s" if n != 1, empty string otherwise.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// qualifiedName returns "namespace/name" when in all-namespaces mode,
// or just "name" when viewing a specific namespace (since ns is redundant).
func (v *HealthView) qualifiedName(namespace, name string) string {
	if v.namespace == "" && namespace != "" {
		return namespace + "/" + name
	}
	return name
}

// severityRank returns a sort rank for diagnosis severity (lower = more severe)
func severityRank(s analyzer.Severity) int {
	switch s {
	case analyzer.SeverityCritical:
		return 0
	case analyzer.SeverityWarning:
		return 1
	case analyzer.SeverityInfo:
		return 2
	default:
		return 3
	}
}

// problemPhaseRank returns a sort rank for pod phase (lower = worse)
func problemPhaseRank(phase string) int {
	switch phase {
	case "Failed":
		return 0
	case "Pending":
		return 1
	default:
		return 2
	}
}

// --- Row rendering helpers (match table component visual style) ---

// healthIndicator returns the row indicator prefix (2 visual chars wide).
// When the row is selected, shows ► with selection background (matching table's RowIndicatorFocused).
// Otherwise shows 2 spaces with normal background.
func healthIndicator(isSelected bool) string {
	if isSelected {
		return theme.Styles.RowIndicatorFocused.Render("►") +
			lipgloss.NewStyle().Background(theme.ColorSelectionBg).Render(" ")
	}
	return lipgloss.NewStyle().Background(theme.ColorBackground).Render("  ")
}

// healthPadRow pads a rendered row line to the target width using the appropriate
// background color (selection background for selected rows, normal background otherwise).
func healthPadRow(line string, width int, isSelected bool) string {
	lineWidth := lipgloss.Width(line)
	if lineWidth >= width {
		return line
	}
	var bg lipgloss.TerminalColor = theme.ColorBackground
	if isSelected {
		bg = theme.ColorSelectionBg
	}
	return line + lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", width-lineWidth))
}

// healthCellStyle returns a lipgloss style for rendering a cell value in health view rows.
// Selected rows use selection background with the original foreground color preserved,
// matching how the table component renders status-colored cells on selection.
func healthCellStyle(fg lipgloss.TerminalColor, isSelected bool) lipgloss.Style {
	bg := theme.ColorBackground
	if isSelected {
		bg = theme.ColorSelectionBg
	}
	return lipgloss.NewStyle().
		Foreground(fg).
		Background(bg)
}

// healthGap returns a single-space column gap with appropriate background,
// matching the table component's columnGap = 1.
func healthGap(isSelected bool) string {
	if isSelected {
		return lipgloss.NewStyle().Background(theme.ColorSelectionBg).Render(" ")
	}
	return lipgloss.NewStyle().Background(theme.ColorBackground).Render(" ")
}

// --- Rendering helpers ---

// renderSectionHeader renders "── Title ──..." or "━━ Title ━━..." when focused.
// itemCount controls header color: -1 = neutral (no count semantics, e.g., Overview/Nodes),
// 0 = healthy (green checkmark appended), >0 = warning/error coloring.
func renderSectionHeader(title string, width int, focused bool, itemCount int) string {
	var titleStyle, dividerStyle lipgloss.Style
	divChar := "─"
	if focused {
		titleStyle = lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Background(theme.ColorBackground).
			Bold(true)
		dividerStyle = lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Background(theme.ColorBackground)
		divChar = "━"
	} else if itemCount == 0 {
		titleStyle = lipgloss.NewStyle().
			Foreground(theme.ColorSuccess).
			Background(theme.ColorBackground).
			Bold(true)
		dividerStyle = lipgloss.NewStyle().
			Foreground(theme.ColorMuted).
			Background(theme.ColorBackground)
	} else if itemCount > 0 {
		titleStyle = lipgloss.NewStyle().
			Foreground(theme.ColorWarning).
			Background(theme.ColorBackground).
			Bold(true)
		dividerStyle = lipgloss.NewStyle().
			Foreground(theme.ColorMuted).
			Background(theme.ColorBackground)
	} else {
		titleStyle = lipgloss.NewStyle().
			Foreground(theme.ColorHighlight).
			Background(theme.ColorBackground).
			Bold(true)
		dividerStyle = lipgloss.NewStyle().
			Foreground(theme.ColorMuted).
			Background(theme.ColorBackground)
	}

	prefix := dividerStyle.Render(divChar + divChar + " ")
	titleRendered := titleStyle.Render(title)

	// Append checkmark for healthy sections
	checkmark := ""
	if itemCount == 0 && !focused {
		checkStyle := lipgloss.NewStyle().
			Foreground(theme.ColorSuccess).
			Background(theme.ColorBackground)
		checkmark = " " + checkStyle.Render(theme.IconSuccess)
	}

	suffix := " "
	usedWidth := 3 + lipgloss.Width(titleRendered) + lipgloss.Width(checkmark) + 1
	remaining := width - usedWidth
	if remaining < 0 {
		remaining = 0
	}
	line := prefix + titleRendered + checkmark + dividerStyle.Render(suffix+strings.Repeat(divChar, remaining))
	return theme.PadToWidth(line, width, theme.ColorBackground)
}

// joinColumns renders left and right content on one line with left padded to leftWidth
func joinColumns(left, right string, leftWidth, totalWidth int) string {
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	lw := lipgloss.Width(left)
	gap := leftWidth - lw
	if gap < 0 {
		gap = 0
	}
	line := left + bgStyle.Render(strings.Repeat(" ", gap)) + right
	return theme.PadToWidth(line, totalWidth, theme.ColorBackground)
}

// --- Bar chart helpers ---

func renderBar(pct int, barWidth int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := barWidth * pct / 100
	empty := barWidth - filled

	filledStyle := lipgloss.NewStyle().
		Foreground(barColor(pct)).
		Background(theme.ColorBackground)
	emptyStyle := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Background(theme.ColorBackground)

	return filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty))
}

func barColor(pct int) lipgloss.TerminalColor {
	if pct >= 85 {
		return theme.ColorError
	}
	if pct >= 60 {
		return theme.ColorWarning
	}
	return theme.ColorAccent
}

func parsePercentage(s string) int {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}

// --- Node column layout ---

type nodeCols struct {
	name, status, roles, cpu, mem, pods, ip, version int
}

func nodeColWidths(width int) nodeCols {
	c := nodeCols{
		name:    16,
		status:  10,
		roles:   10,
		cpu:     7,
		mem:     7,
		pods:    6,
		ip:      16,
		version: 12,
	}
	// 2=indicator, 7=gaps
	widths := []int{c.name, c.status, c.roles, c.cpu, c.mem, c.pods, c.ip, c.version}
	distributeEqual(widths, width, 2+7)
	c.name, c.status, c.roles, c.cpu, c.mem, c.pods, c.ip, c.version = widths[0], widths[1], widths[2], widths[3], widths[4], widths[5], widths[6], widths[7]
	return c
}

// distributeEqual distributes remaining width equally across all columns,
// matching the table component's Phase 2 algorithm. Uses n+1 slots
// (n columns + 1 trailing) so the gap after the last column matches
// the inter-column spacing. The trailing slot becomes padding via
// healthPadRow automatically.
func distributeEqual(widths []int, totalWidth int, overhead int) {
	total := overhead
	for _, w := range widths {
		total += w
	}
	remaining := totalWidth - total
	if remaining <= 0 {
		return
	}
	slots := len(widths) + 1
	perSlot := remaining / slots
	extra := remaining % slots
	for i := range widths {
		widths[i] += perSlot
		if i < extra {
			widths[i]++
		}
	}
}

// --- Unhealthy Workloads column layout ---

type unhealthyCols struct {
	kind, name, ready, age int
}

func unhealthyColWidths(width int) unhealthyCols {
	c := unhealthyCols{
		kind:  12,
		name:  16,
		ready: 7,
		age:   6,
	}
	// 2=indicator, 3=gaps
	widths := []int{c.kind, c.name, c.ready, c.age}
	distributeEqual(widths, width, 2+3)
	c.kind, c.name, c.ready, c.age = widths[0], widths[1], widths[2], widths[3]
	return c
}

// --- Failed Jobs column layout ---

type failedJobCols struct {
	name, status, completions, age int
}

func failedJobColWidths(width int) failedJobCols {
	c := failedJobCols{
		name:        16,
		status:      10,
		completions: 12,
		age:         6,
	}
	// 2=indicator, 3=gaps
	widths := []int{c.name, c.status, c.completions, c.age}
	distributeEqual(widths, width, 2+3)
	c.name, c.status, c.completions, c.age = widths[0], widths[1], widths[2], widths[3]
	return c
}

// --- Problem Pods column layout ---

type problemCols struct {
	name, ready, status, age int
}

func problemColWidths(width int) problemCols {
	c := problemCols{
		name:   16,
		ready:  7,
		status: 12,
		age:    6,
	}
	// 2=indicator, 3=gaps, 3=status icon prefix
	widths := []int{c.name, c.ready, c.status, c.age}
	distributeEqual(widths, width, 2+3+3)
	c.name, c.ready, c.status, c.age = widths[0], widths[1], widths[2], widths[3]
	return c
}

// --- Pending PVCs column layout ---

type pendingPVCCols struct {
	name, storageClass, age int
}

func pendingPVCColWidths(width int) pendingPVCCols {
	c := pendingPVCCols{
		name:         16,
		storageClass: 16,
		age:          6,
	}
	// 2=indicator, 2=gaps
	widths := []int{c.name, c.storageClass, c.age}
	distributeEqual(widths, width, 2+2)
	c.name, c.storageClass, c.age = widths[0], widths[1], widths[2]
	return c
}

// --- Restarts column layout ---

type restartCols struct {
	restarts, name, reason, status, lastRestart int
}

func restartColWidths(width int) restartCols {
	c := restartCols{
		restarts:    8,
		name:        16,
		reason:      18,
		status:      12,
		lastRestart: 12,
	}
	// 2=indicator, 4=gaps, 3=status icon prefix
	widths := []int{c.name, c.restarts, c.reason, c.status, c.lastRestart}
	distributeEqual(widths, width, 2+4+3)
	c.name, c.restarts, c.reason, c.status, c.lastRestart = widths[0], widths[1], widths[2], widths[3], widths[4]
	return c
}

// --- Issues column layout ---

type issueCols struct {
	severity, resource, problem int
}

func issueColWidths(width int) issueCols {
	c := issueCols{
		severity: 4,
		resource: 22,
		problem:  10,
	}
	// 2=indicator, 2=gaps, 3=severity icon prefix (icon + 2 spaces)
	widths := []int{c.resource, c.severity, c.problem}
	distributeEqual(widths, width, 2+2+3)
	c.resource, c.severity, c.problem = widths[0], widths[1], widths[2]
	return c
}

// --- Events column layout ---

type eventCols struct {
	ago, typeLabel, resource, message int
}

func eventColWidths(width int) eventCols {
	c := eventCols{
		ago:       8,
		typeLabel: 7,
		resource:  22,
		message:   10,
	}
	// 2=indicator, 3=gaps, 3=type icon prefix (icon + 2 spaces)
	widths := []int{c.resource, c.ago, c.typeLabel, c.message}
	distributeEqual(widths, width, 2+3+3)
	c.resource, c.ago, c.typeLabel, c.message = widths[0], widths[1], widths[2], widths[3]
	return c
}
