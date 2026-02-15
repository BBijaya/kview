package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

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
}

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
	userVal, userStyle := getStyledValue(theme.TruncateString(h.user, 30), valueStyle)
	k8sVerVal, k8sVerStyle := getStyledValue(h.serverVersion, valueStyle)
	cpuVal, cpuStyle := getStyledValue(h.cpuUsage, mutedStyle)
	memVal, memStyle := getStyledValue(h.memUsage, mutedStyle)

	// Append access mode to context value
	ctxDisplay := ctxStyle.Render(ctxVal)
	if h.accessMode != "" {
		ctxDisplay += bgPadStyle.Render(" ") + accessModeStyle.Render("["+h.accessMode+"]")
	}

	// Calculate column widths (split into 7 equal parts)
	// Col1 gets 2 parts, Col2 gets 2 parts, Col3 gets 1 part, Col4 gets 2 parts
	seventh := h.width / 7
	col1Width := seventh * 2
	col2Width := seventh * 2
	col3Width := seventh
	if col1Width < 25 {
		col1Width = 25
	}
	if col2Width < 20 {
		col2Width = 20
	}
	if col3Width < 14 {
		col3Width = 14
	}

	// Get navigation shortcuts (column 2), filter hints (column 3), contextual shortcuts (column 4)
	shortcuts := h.getShortcutRows()
	filterHints := h.getFilterHintRows()
	contextual := h.getContextualRows()

	// Build 6 info rows with 4 columns each
	// Labels padded to 8 chars (max of "Context:", "Cluster:", "K8s Rev:")
	labelWidth := 8
	infoRows := []struct {
		label        string
		renderedVal  string
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

		// Column 2: Navigation shortcuts
		col2 := ""
		if i < len(shortcuts) {
			col2 = shortcuts[i]
		}

		// Column 3: Filter/sort hints
		col3 := ""
		if i < len(filterHints) {
			col3 = filterHints[i]
		}

		// Column 4: Contextual action shortcuts
		col4 := ""
		if i < len(contextual) {
			col4 = contextual[i]
		}

		line := h.formatFourColumnLine(col1, col2, col3, col4, col1Width, col2Width, col3Width)
		lines = append(lines, theme.PadToWidth(bgStyle.Render(line), h.width, theme.ColorBackground))
	}

	// Row 7: Category tabs + resource tabs (full width)
	tabsLine := h.renderCategoryWithResources()
	lines = append(lines, theme.PadToWidth(bgStyle.Render(tabsLine), h.width, theme.ColorBackground))

	return strings.Join(lines, "\n")
}

// getShortcutRows returns navigation shortcuts (column 2, same for all views)
func (h *Header) getShortcutRows() []string {
	keyStyle := theme.Styles.ShortcutKey
	descStyle := theme.Styles.ShortcutDesc
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	sep := bgStyle.Render("  ")

	fmtKey := func(k, d string) string {
		return keyStyle.Render(k) + descStyle.Render(":"+d)
	}

	return []string{
		fmtKey("↑↓", "nav") + sep + fmtKey("enter", "select"),
		fmtKey("n", "ns") + sep + fmtKey("ctrl+k", "ctx"),
		fmtKey("ctrl+p", "palette") + sep + fmtKey("ctrl+r", "refresh"),
		fmtKey("c", "copy") + sep + fmtKey("q", "quit"),
		fmtKey("esc", "back") + sep + fmtKey("?", "help"),
	}
}

// getFilterHintRows returns filter/sort hints (column 3)
func (h *Header) getFilterHintRows() []string {
	keyStyle := theme.Styles.ShortcutKey
	descStyle := theme.Styles.ShortcutDesc
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	sep := bgStyle.Render("  ")

	fmtKey := func(k, d string) string {
		return keyStyle.Render(k) + descStyle.Render(":"+d)
	}

	return []string{
		fmtKey("/", "filter"),
		fmtKey("!", "invert") + sep + fmtKey("-f", "fuzzy"),
		fmtKey("-l", "label"),
		fmtKey("S", "sort") + sep + fmtKey("[", "prev") + sep + fmtKey("]", "next"),
	}
}

// getContextualRows returns view-specific action shortcuts (column 4)
func (h *Header) getContextualRows() []string {
	keyStyle := theme.Styles.ShortcutKey
	descStyle := theme.Styles.ShortcutDesc
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	sep := bgStyle.Render("  ")

	fmtKey := func(k, d string) string {
		return keyStyle.Render(k) + descStyle.Render(":"+d)
	}

	switch h.currentViewType {
	case "pods":
		return []string{
			fmtKey("d", "describe") + sep + fmtKey("y", "yaml"),
			fmtKey("l", "logs") + sep + fmtKey("s", "shell"),
			fmtKey("F", "pf") + sep + fmtKey("ctrl+d", "delete"),
		}
	case "deployments":
		return []string{
			fmtKey("d", "describe") + sep + fmtKey("y", "yaml"),
			fmtKey("r", "restart") + sep + fmtKey("s", "scale"),
			fmtKey("ctrl+d", "delete"),
		}
	case "services":
		return []string{
			fmtKey("d", "describe") + sep + fmtKey("y", "yaml"),
			fmtKey("F", "pf") + sep + fmtKey("ctrl+d", "delete"),
		}
	case "configmaps":
		return []string{
			fmtKey("d", "describe") + sep + fmtKey("y", "yaml"),
			fmtKey("ctrl+d", "delete"),
		}
	case "portforwards":
		return []string{
			fmtKey("ctrl+d", "stop"),
		}
	case "secrets":
		return []string{
			fmtKey("d", "describe") + sep + fmtKey("y", "yaml"),
			fmtKey("x", "decode") + sep + fmtKey("ctrl+d", "delete"),
		}
	case "containers":
		return []string{
			fmtKey("d", "describe") + sep + fmtKey("l", "logs"),
			fmtKey("s", "shell"),
		}
	case "helmreleases":
		return []string{
			fmtKey("enter", "history") + sep + fmtKey("d", "describe"),
			fmtKey("v", "values") + sep + fmtKey("m", "manifest"),
			fmtKey("y", "yaml") + sep + fmtKey("ctrl+d", "delete"),
		}
	case "helmhistory":
		return []string{
			fmtKey("enter", "detail") + sep + fmtKey("d", "describe"),
			fmtKey("v", "values") + sep + fmtKey("m", "manifest"),
			fmtKey("y", "yaml"),
		}
	case "xray":
		return []string{
			fmtKey("enter", "expand") + sep + fmtKey("d", "describe"),
			fmtKey("y", "yaml") + sep + fmtKey("l", "logs"),
			fmtKey("ctrl+d", "delete"),
		}
	default:
		return []string{
			fmtKey("d", "describe") + sep + fmtKey("y", "yaml"),
			fmtKey("X", "xray") + sep + fmtKey("ctrl+d", "delete"),
		}
	}
}

// formatFourColumnLine formats a line with four columns
func (h *Header) formatFourColumnLine(col1, col2, col3, col4 string, col1Width, col2Width, col3Width int) string {
	bgPadStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	// Pad col1 to its width
	col1Len := lipgloss.Width(col1)
	pad1 := col1Width - col1Len
	if pad1 < 1 {
		pad1 = 1
	}

	// Pad col2 to its width
	col2Len := lipgloss.Width(col2)
	pad2 := col2Width - col2Len
	if pad2 < 1 {
		pad2 = 1
	}

	// Pad col3 to its width
	col3Len := lipgloss.Width(col3)
	pad3 := col3Width - col3Len
	if pad3 < 1 {
		pad3 = 1
	}

	// Use styled padding to maintain background
	padding1 := bgPadStyle.Render(strings.Repeat(" ", pad1))
	padding2 := bgPadStyle.Render(strings.Repeat(" ", pad2))
	padding3 := bgPadStyle.Render(strings.Repeat(" ", pad3))

	return col1 + padding1 + col2 + padding2 + col3 + padding3 + col4
}

// formatTwoColumnLine formats a line with left and right content
func (h *Header) formatTwoColumnLine(left, right string, leftWidth int) string {
	bgPadStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	leftLen := lipgloss.Width(left)
	rightLen := lipgloss.Width(right)

	// Pad left to column width
	padding := leftWidth - leftLen
	if padding < 2 {
		padding = 2
	}

	// Ensure right fits
	totalLen := leftLen + padding + rightLen
	if totalLen > h.width {
		// Truncate padding
		padding = h.width - leftLen - rightLen
		if padding < 1 {
			padding = 1
		}
	}

	return left + bgPadStyle.Render(strings.Repeat(" ", padding)) + right
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
		if i == h.activeCategory {
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
		if i == h.activeResourceIdx {
			style = theme.Styles.ResourceItemActive
		}

		numStr := numStyle.Render("[" + intToStr(i+1) + "]")
		result.WriteString(numStr)
		result.WriteString(style.Render(tab))
		result.WriteString(bgStyle.Render(" "))
	}

	return result.String()
}

// formatInfoLine formats info items in columns
func (h *Header) formatInfoLine(parts []string) string {
	if len(parts) == 0 {
		return ""
	}

	bgPadStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	// Calculate column width - divide width into equal columns
	colWidth := (h.width - 4) / len(parts) // -4 for some padding
	if colWidth < 20 {
		colWidth = 20
	}

	var result strings.Builder
	result.WriteString(bgPadStyle.Render("  ")) // Left padding

	for i, part := range parts {
		partWidth := lipgloss.Width(part)
		result.WriteString(part)
		if i < len(parts)-1 {
			// Pad to column width
			padding := colWidth - partWidth
			if padding > 0 {
				result.WriteString(bgPadStyle.Render(strings.Repeat(" ", padding)))
			}
		}
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
