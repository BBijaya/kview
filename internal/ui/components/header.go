package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/ui/theme"
)

// Header is the top bar component
type Header struct {
	width          int
	title          string
	context        string
	namespace      string
	serverVersion  string
	version        string
	tabs           []string
	activeTab      int
	viewName       string
	filterActive   bool
	filterText     string
	categoryName   string

	// New fields for k9s-style info pane
	user        string
	clusterName string
	cpuUsage    string
	memUsage    string

	// Category and resource tabs for two-tier navigation
	categories       []string
	activeCategory   int
	resourceTabs     [][]string // tabs per category

	// Current view type for contextual shortcuts
	currentViewType  string

	// Access mode indicator (RW, RO, or n/a)
	accessMode string

	// Index of active resource within current category (for tab highlighting)
	activeResourceIdx int

	// Delta filter indicator
	deltaFilterActive bool

	// Whether to highlight active category/resource tabs
	highlightTabs bool
}

// shortcutItem represents a single keyboard shortcut for the header menu
type shortcutItem struct {
	key  string // e.g. "d", "ctrl+d", "shift+f"
	desc string // e.g. "Describe", "Delete", "Port-Forward"
}

// shortcutMaxRows is the number of rows available for shortcuts (matches info pane rows)
const shortcutMaxRows = 6

// NewHeader creates a new header component
func NewHeader() *Header {
	return &Header{
		width:        80,
		title:        "kview",
		version:      "v0.1.0",
		tabs:         []string{"Pods", "Deployments", "Services"},
		viewName:     "Pods",
		categoryName: "Workloads",
	}
}

// SetWidth sets the header width
func (h *Header) SetWidth(width int) {
	h.width = width
}

// SetContext sets the current context
func (h *Header) SetContext(context string) {
	h.context = context
}

// SetNamespace sets the current namespace
func (h *Header) SetNamespace(namespace string) {
	h.namespace = namespace
}

// SetServerVersion sets the server version
func (h *Header) SetServerVersion(version string) {
	h.serverVersion = version
}

// SetVersion sets the application version
func (h *Header) SetVersion(version string) {
	h.version = version
}

// SetViewName sets the current view name
func (h *Header) SetViewName(name string) {
	h.viewName = name
}

// SetCategoryName sets the current category name
func (h *Header) SetCategoryName(name string) {
	h.categoryName = name
}

// SetUser sets the current user name
func (h *Header) SetUser(user string) {
	h.user = user
}

// SetClusterName sets the cluster name
func (h *Header) SetClusterName(name string) {
	h.clusterName = name
}

// SetCPUUsage sets the CPU usage string
func (h *Header) SetCPUUsage(cpu string) {
	h.cpuUsage = cpu
}

// SetMemUsage sets the memory usage string
func (h *Header) SetMemUsage(mem string) {
	h.memUsage = mem
}

// SetCategories sets the category tabs
func (h *Header) SetCategories(categories []string) {
	h.categories = categories
}

// SetActiveCategory sets the active category index
func (h *Header) SetActiveCategory(index int) {
	if index >= 0 && index < len(h.categories) {
		h.activeCategory = index
	}
}

// ActiveCategory returns the active category index
func (h *Header) ActiveCategory() int {
	return h.activeCategory
}

// SetResourceTabs sets the resource tabs for each category
func (h *Header) SetResourceTabs(tabs [][]string) {
	h.resourceTabs = tabs
}

// SetCurrentViewType sets the current view type for contextual shortcuts
func (h *Header) SetCurrentViewType(viewType string) {
	h.currentViewType = viewType
}

// SetActiveResourceIdx sets the active resource index within the current category
func (h *Header) SetActiveResourceIdx(idx int) {
	h.activeResourceIdx = idx
}

// SetAccessMode sets the access mode indicator (e.g. "RW", "RO", "n/a")
func (h *Header) SetAccessMode(mode string) {
	h.accessMode = mode
}

// SetFilter sets the active filter
func (h *Header) SetFilter(active bool, text string) {
	h.filterActive = active
	h.filterText = text
}

// SetDeltaFilter sets the delta filter indicator
func (h *Header) SetDeltaFilter(active bool) {
	h.deltaFilterActive = active
}

// SetHighlightTabs sets whether to highlight active category/resource tabs
func (h *Header) SetHighlightTabs(v bool) {
	h.highlightTabs = v
}

// SetTabs sets the available tabs
func (h *Header) SetTabs(tabs []string) {
	h.tabs = tabs
}

// SetActiveTab sets the active tab index
func (h *Header) SetActiveTab(index int) {
	if index >= 0 && index < len(h.tabs) {
		h.activeTab = index
	}
}

// ActiveTab returns the active tab index
func (h *Header) ActiveTab() int {
	return h.activeTab
}

// View renders the header in a rich format showing context info
// ctx: minikube | ns: default | cluster: v1.28.0
func (h *Header) View() string {
	sep := theme.Styles.HeaderSeparator.Render(" │ ")
	mutedSep := theme.Styles.HeaderSeparator.Render(":")

	var parts []string

	if h.context != "" {
		ctx := theme.Styles.HelpDesc.Render("ctx") + mutedSep + theme.Styles.HeaderContext.Render(h.context)
		parts = append(parts, ctx)
	}

	if h.namespace != "" {
		ns := theme.Styles.HelpDesc.Render("ns") + mutedSep + theme.Styles.HeaderNamespace.Render(h.namespace)
		parts = append(parts, ns)
	}

	if h.serverVersion != "" {
		cluster := theme.Styles.HelpDesc.Render("cluster") + mutedSep + theme.Styles.HeaderCluster.Render(h.serverVersion)
		parts = append(parts, cluster)
	}

	leftContent := strings.Join(parts, sep)

	// Right side: breadcrumb trail
	var breadcrumb string
	if h.categoryName != "" && h.viewName != "" {
		breadcrumb = theme.Styles.HelpDesc.Render(h.categoryName) +
			theme.Styles.HeaderSeparator.Render(" > ") +
			theme.Styles.FrameTitle.Render(h.viewName)
	}

	if h.filterActive && h.filterText != "" {
		breadcrumb += theme.Styles.HeaderSeparator.Render(" > ") +
			theme.Styles.HeaderNamespace.Render("["+h.filterText+"]")
	}

	if h.deltaFilterActive {
		breadcrumb += theme.Styles.HeaderSeparator.Render(" > ") +
			theme.Styles.StatusError.Render("[ERRORS]")
	}

	// Calculate padding
	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(breadcrumb)
	padding := h.width - leftWidth - rightWidth - 2
	if padding < 1 {
		padding = 1
	}

	content := leftContent + strings.Repeat(" ", padding) + breadcrumb
	return theme.Styles.StatusBar.Width(h.width).Render(content)
}

// ViewWithTabs renders the header with tabs below
func (h *Header) ViewWithTabs() string {
	headerLine := h.View()

	// Tab bar
	var tabsBuilder strings.Builder
	for i, tab := range h.tabs {
		style := theme.Styles.Tab
		if i == h.activeTab {
			style = theme.Styles.TabActive
		}
		tabsBuilder.WriteString(style.Render(tab))
	}

	// Pad tabs to full width
	tabsContent := tabsBuilder.String()
	tabsLine := lipgloss.NewStyle().
		Background(theme.ColorSurface).
		Width(h.width).
		Render(tabsContent)

	return lipgloss.JoinVertical(lipgloss.Left, headerLine, tabsLine)
}

// TabsView returns just the tabs portion
func (h *Header) TabsView() string {
	var tabsBuilder strings.Builder
	for i, tab := range h.tabs {
		style := theme.Styles.Tab
		if i == h.activeTab {
			style = theme.Styles.TabActive
		}
		tabsBuilder.WriteString(style.Render(tab))
	}
	return tabsBuilder.String()
}

// InfoPaneView renders the k9s-style multi-line info pane with shortcuts
// Layout (4 columns, 6 info rows + 1 tabs row):
//   Col 1: Info (stacked)    Col 2: Nav shortcuts   Col 3: Filter hints   Col 4: Actions
//   C: Context: <ctx>        ↑↓:nav  enter:sel      /:filter              d:describe y:yaml
//   K: Cluster: <cluster>    n:ns  ctrl+k:ctx        !:invert  -f:fuzzy    l:logs s:shell
//   U: User: <user>          ctrl+p:palette          -l:label              F:pf ctrl+d:del
//   V: K8s Rev: <ver>        ctrl+r:refresh          S:sort  [:prev ]:next
//   CPU: <cpu>               c:copy  q:quit
//   MEM: <mem>               esc:back  ?:help
//   ► Workloads  Network  Config  Cluster   [1]Pods [2]Deploy ...
func (h *Header) InfoPaneView() string {
	bgStyle := theme.Styles.InfoPane.Width(h.width)
	bgPadStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	labelStyle := theme.Styles.InfoLabel
	valueStyle := theme.Styles.InfoValue
	mutedStyle := theme.Styles.InfoValueMuted
	naStyle := theme.Styles.InfoValueNA
	accessModeStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Background(theme.ColorBackground)

	// Helper to get value with proper styling
	getStyledValue := func(val string, defaultStyle lipgloss.Style) (string, lipgloss.Style) {
		if val == "" || val == "n/a" {
			return "n/a", naStyle
		}
		return val, defaultStyle
	}

	// Get values with defaults and proper styles
	ctxVal, ctxStyle := getStyledValue(h.context, valueStyle)
	clusterVal, clusterStyle := getStyledValue(h.clusterName, valueStyle)
	userVal, userStyle := getStyledValue(h.user, valueStyle)
	k8sVerVal, k8sVerStyle := getStyledValue(h.serverVersion, valueStyle)
	cpuVal, cpuStyle := getStyledValue(h.cpuUsage, mutedStyle)
	memVal, memStyle := getStyledValue(h.memUsage, mutedStyle)

	// Calculate column 1 width (info column)
	seventh := h.width / 7
	col1Width := seventh * 2
	if col1Width < 25 {
		col1Width = 25
	}

	// Available width for shortcut columns
	remaining := h.width - col1Width

	// Truncate all column 1 values to fit within col1Width
	// Overhead: 2 (leading spaces) + 8 (label) + 1 (space after label) = 11
	maxValueWidth := col1Width - 11
	if maxValueWidth < 5 {
		maxValueWidth = 5
	}

	// Context line has access mode badge eating into its budget
	ctxMaxWidth := maxValueWidth
	if h.accessMode != "" {
		ctxMaxWidth -= len(h.accessMode) + 3 // " [XX]"
		if ctxMaxWidth < 5 {
			ctxMaxWidth = 5
		}
	}
	ctxVal = theme.TruncateString(ctxVal, ctxMaxWidth)
	clusterVal = theme.TruncateString(clusterVal, maxValueWidth)
	userVal = theme.TruncateString(userVal, maxValueWidth)
	k8sVerVal = theme.TruncateString(k8sVerVal, maxValueWidth)
	cpuVal = theme.TruncateString(cpuVal, maxValueWidth)
	memVal = theme.TruncateString(memVal, maxValueWidth)

	// Append access mode to context value (after truncation)
	ctxDisplay := ctxStyle.Render(ctxVal)
	if h.accessMode != "" {
		ctxDisplay += bgPadStyle.Render(" ") + accessModeStyle.Render("["+h.accessMode+"]")
	}

	// Layout shortcut columns using k9s-style vertical flow
	items := h.getShortcutItems()
	shortcutCols, shortcutWidths := h.layoutShortcutColumns(items, remaining)

	// Build 6 info rows
	// Labels padded to 8 chars (max of "Context:", "Cluster:", "K8s Rev:")
	labelWidth := 8
	infoRows := []struct {
		label       string
		renderedVal string
	}{
		{"Context:", ctxDisplay},
		{"Cluster:", clusterStyle.Render(clusterVal)},
		{"User:", userStyle.Render(userVal)},
		{"K8s Rev:", k8sVerStyle.Render(k8sVerVal)},
		{"CPU:", cpuStyle.Render(cpuVal)},
		{"MEM:", memStyle.Render(memVal)},
	}

	var lines []string
	for i, info := range infoRows {
		// Column 1: Label (padded to fixed width) + value
		paddedLabel := labelStyle.Render(fmt.Sprintf("%-*s", labelWidth, info.label))
		col1 := bgPadStyle.Render("  ") + paddedLabel + bgPadStyle.Render(" ") + info.renderedVal

		// Clamp col1 to exactly col1Width to guarantee shortcut column alignment.
		col1W := lipgloss.Width(col1)
		if col1W < col1Width {
			col1 += bgPadStyle.Render(strings.Repeat(" ", col1Width-col1W))
		}

		// Assemble all columns for this row
		columns := []string{col1}
		widths := []int{col1Width}
		for j, col := range shortcutCols {
			val := ""
			if i < len(col) {
				val = col[i]
			}
			columns = append(columns, val)
			widths = append(widths, shortcutWidths[j])
		}

		line := h.formatColumnsLine(columns, widths)
		lines = append(lines, theme.PadToWidth(bgStyle.Render(line), h.width, theme.ColorBackground))
	}

	// Row 7: Category tabs + resource tabs (full width)
	tabsLine := h.renderCategoryWithResources()
	lines = append(lines, theme.PadToWidth(bgStyle.Render(tabsLine), h.width, theme.ColorBackground))

	return strings.Join(lines, "\n")
}

// getShortcutItems returns a flat, priority-ordered list of all keyboard shortcuts.
// View-specific items come first (highest value), followed by universal actions,
// core navigation, and extended navigation. When the terminal is too narrow,
// columns are dropped from the end, so lowest-priority items disappear first.
func (h *Header) getShortcutItems() []shortcutItem {
	// View-specific shortcuts (highest priority)
	var items []shortcutItem
	switch h.currentViewType {
	case "pods":
		items = []shortcutItem{
			{"d", "Describe"}, {"y", "YAML"},
			{"l", "Logs"}, {"s", "Shell"},
			{"F", "Port-Forward"}, {"ctrl+d", "Delete"},
		}
	case "deployments":
		items = []shortcutItem{
			{"d", "Describe"}, {"y", "YAML"},
			{"r", "Restart"}, {"s", "Scale"},
			{"ctrl+d", "Delete"},
		}
	case "services":
		items = []shortcutItem{
			{"d", "Describe"}, {"y", "YAML"},
			{"F", "Port-Forward"}, {"ctrl+d", "Delete"},
		}
	case "configmaps":
		items = []shortcutItem{
			{"d", "Describe"}, {"y", "YAML"},
			{"ctrl+d", "Delete"},
		}
	case "portforwards":
		items = []shortcutItem{
			{"ctrl+d", "Stop"},
		}
	case "secrets":
		items = []shortcutItem{
			{"d", "Describe"}, {"y", "YAML"},
			{"x", "Decode"}, {"ctrl+d", "Delete"},
		}
	case "containers":
		items = []shortcutItem{
			{"d", "Describe"}, {"l", "Logs"},
			{"s", "Shell"},
		}
	case "helmreleases":
		items = []shortcutItem{
			{"enter", "History"}, {"d", "Describe"},
			{"v", "Values"}, {"m", "Manifest"},
			{"y", "YAML"}, {"ctrl+d", "Delete"},
		}
	case "helmhistory":
		items = []shortcutItem{
			{"enter", "Detail"}, {"d", "Describe"},
			{"v", "Values"}, {"m", "Manifest"},
			{"y", "YAML"},
		}
	case "xray":
		items = []shortcutItem{
			{"enter", "Expand"}, {"d", "Describe"},
			{"y", "YAML"}, {"l", "Logs"},
			{"ctrl+d", "Delete"},
		}
	default:
		items = []shortcutItem{
			{"d", "Describe"}, {"y", "YAML"},
			{"X", "Xray"}, {"ctrl+d", "Delete"},
		}
	}

	// Universal actions
	items = append(items, []shortcutItem{
		{"e", "Edit"}, {"c", "Copy"},
	}...)

	// Core navigation
	items = append(items, []shortcutItem{
		{"↑↓", "Navigate"}, {"enter", "Select"},
		{"/", "Filter"}, {"?", "Help"},
		{"n", "Namespace"}, {":ctx", "Context"},
		{"ctrl+r", "Refresh"}, {"q", "Quit"},
		{"esc", "Back"},
	}...)

	// Extended navigation (power user, lowest priority)
	items = append(items, []shortcutItem{
		{"tab", "Next/Prev"}, {"←→", "Scroll"},
		{"pgup/dn", "Page"}, {"g", "Top"}, {"G", "Bottom"},
		{"ctrl+p", "Palette"}, {"!", "Invert"},
		{"-f", "Fuzzy"}, {"-l", "Label"},
		{"S", "Sort"}, {"[/]", "Prev/Next"},
		{"ctrl+z", "Errors"},
	}...)

	return items
}

// layoutShortcutColumns arranges shortcut items in k9s-style vertical-flow columns.
// Items fill top-to-bottom (up to shortcutMaxRows per column), then wrap to the next column.
// Each column is auto-sized to fit its widest key and description.
// Columns are added left-to-right until the available width is exhausted.
func (h *Header) layoutShortcutColumns(items []shortcutItem, availableWidth int) ([][]string, []int) {
	keyStyle := theme.Styles.ShortcutKey
	descStyle := theme.Styles.ShortcutDesc
	bgPadStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	colGap := 4
	innerPad := 2

	// First pass: compute a single uniform column width across ALL items
	globalMaxKeyW := 0
	globalMaxDescW := 0
	for _, item := range items {
		kw := lipgloss.Width("<" + item.key + ">")
		if kw > globalMaxKeyW {
			globalMaxKeyW = kw
		}
		dw := lipgloss.Width(item.desc)
		if dw > globalMaxDescW {
			globalMaxDescW = dw
		}
	}
	uniformColWidth := globalMaxKeyW + innerPad + globalMaxDescW + colGap

	// How many shortcut columns fit?
	maxCols := availableWidth / uniformColWidth
	if maxCols < 0 {
		maxCols = 0
	}

	// Second pass: render columns using the uniform width
	var columns [][]string
	var widths []int
	for start := 0; start < len(items) && len(columns) < maxCols; start += shortcutMaxRows {
		end := start + shortcutMaxRows
		if end > len(items) {
			end = len(items)
		}
		chunk := items[start:end]

		var col []string
		for _, item := range chunk {
			renderedKey := keyStyle.Render("<" + item.key + ">")
			keyW := lipgloss.Width(renderedKey)
			keyPad := globalMaxKeyW - keyW + innerPad

			descW := lipgloss.Width(item.desc)
			descPad := globalMaxDescW - descW

			line := renderedKey +
				bgPadStyle.Render(strings.Repeat(" ", keyPad)) +
				descStyle.Render(item.desc)
			if descPad > 0 {
				line += bgPadStyle.Render(strings.Repeat(" ", descPad))
			}
			col = append(col, line)
		}
		for len(col) < shortcutMaxRows {
			col = append(col, "")
		}

		columns = append(columns, col)
		widths = append(widths, uniformColWidth)
	}

	return columns, widths
}

// formatColumnsLine formats a line with a variable number of columns
func (h *Header) formatColumnsLine(columns []string, widths []int) string {
	bgPadStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	var result strings.Builder
	for i, col := range columns {
		result.WriteString(col)
		if i < len(columns)-1 && i < len(widths) {
			pad := widths[i] - lipgloss.Width(col)
			if pad < 0 {
				pad = 0
			}
			if pad > 0 {
				result.WriteString(bgPadStyle.Render(strings.Repeat(" ", pad)))
			}
		}
	}
	return result.String()
}

// renderCategoryWithResources renders category tabs and resource tabs on same row
func (h *Header) renderCategoryWithResources() string {
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	indicator := theme.Styles.CategoryIndicator.Render("►")

	var result strings.Builder
	result.WriteString(bgStyle.Render("  "))
	result.WriteString(indicator)
	result.WriteString(bgStyle.Render(" "))

	categories := h.categories
	if len(categories) == 0 {
		categories = []string{"Workloads", "Network", "Config", "Cluster"}
	}

	// Render categories
	for i, cat := range categories {
		style := theme.Styles.CategoryItem
		if h.highlightTabs && i == h.activeCategory {
			style = theme.Styles.CategoryItemActive
		}
		result.WriteString(style.Render(cat))
	}

	result.WriteString(bgStyle.Render("   "))

	// Render resource tabs for active category
	var tabs []string
	if h.activeCategory < len(h.resourceTabs) {
		tabs = h.resourceTabs[h.activeCategory]
	}
	if len(tabs) == 0 {
		tabs = h.tabs
	}

	for i, tab := range tabs {
		numStyle := theme.Styles.TabBarNumber
		style := theme.Styles.ResourceItem
		if h.highlightTabs && i == h.activeResourceIdx {
			style = theme.Styles.ResourceItemActive
		}

		numStr := numStyle.Render("[" + intToStr(i+1) + "]")
		result.WriteString(numStr)
		result.WriteString(style.Render(tab))
		result.WriteString(bgStyle.Render(" "))
	}

	return result.String()
}


// renderCategoryTabs renders the category tab row
func (h *Header) renderCategoryTabs() string {
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	indicator := theme.Styles.CategoryIndicator.Render("►")
	var result strings.Builder
	result.WriteString(bgStyle.Render("  ")) // Left padding
	result.WriteString(indicator)
	result.WriteString(bgStyle.Render(" "))

	categories := h.categories
	if len(categories) == 0 {
		categories = []string{"Workloads", "Network", "Config", "Cluster"}
	}

	for i, cat := range categories {
		style := theme.Styles.CategoryItem
		if i == h.activeCategory {
			style = theme.Styles.CategoryItemActive
		}
		result.WriteString(style.Render(cat))
	}

	return result.String()
}

// renderResourceTabs renders the resource tabs for the active category
func (h *Header) renderResourceTabs() string {
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	var result strings.Builder
	result.WriteString(bgStyle.Render("    ")) // Left padding (indented under category indicator)

	// Get tabs for active category
	var tabs []string
	if h.activeCategory < len(h.resourceTabs) {
		tabs = h.resourceTabs[h.activeCategory]
	}
	if len(tabs) == 0 {
		tabs = h.tabs
	}

	for i, tab := range tabs {
		numStyle := theme.Styles.TabBarNumber
		style := theme.Styles.ResourceItem
		if i == h.activeTab {
			style = theme.Styles.ResourceItemActive
		}

		// Format: [1]Pods
		numStr := numStyle.Render("[" + intToStr(i+1) + "]")
		result.WriteString(numStr)
		result.WriteString(style.Render(tab))
		result.WriteString(bgStyle.Render(" "))
	}

	return result.String()
}
