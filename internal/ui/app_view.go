package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
	"github.com/bijaya/kview/internal/ui/views"
)

// View renders the application layout:
//   - Header (borderless): cluster info + shortcuts + category/resource tabs (7 plain text rows)
//   - Command box (conditional): appears between header and body when : is pressed
//   - Body box (bordered): resource label + main table/view content
func (a *App) View() tea.View {
	if a.quitting {
		return tea.NewView("Goodbye!\n")
	}

	// During shell exec, return a blank screen so the renderer's final flush
	// before ReleaseTerminal writes invisible content (no TUI leak to normal screen).
	if a.execing {
		v := tea.NewView(strings.Repeat("\n", max(a.height-1, 0)))
		v.AltScreen = true
		return v
	}

	// Ensure minimum dimensions
	width := a.width
	height := a.height
	if width < 40 {
		width = 40
	}
	if height < 15 {
		height = 15
	}

	// Calculate layout heights
	// Header: 7 info lines (borderless, plain text rows)
	// Footer: 1 line (resource name + loading status)
	headerHeight := 7
	footerHeight := 1
	commandBoxHeight := 0
	if a.inputMode == ModeCommand || a.inputMode == ModeFilter {
		commandBoxHeight = 3 // top border + input + bottom border
	}
	bodyBoxHeight := height - headerHeight - commandBoxHeight - footerHeight
	if bodyBoxHeight < 5 {
		bodyBoxHeight = 5
	}
	bodyInnerHeight := bodyBoxHeight - 2 // minus body frame borders
	contentHeight := bodyInnerHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	innerWidth := width - 2 // body inner width (frame border offset)

	// Ensure inner dimensions are valid
	if innerWidth < 20 {
		innerWidth = 20
	}

	// === 1. Build header content (7 lines, borderless) ===
	a.header.SetWidth(width) // full terminal width, no border offset
	a.header.SetActiveTab(int(a.activeView))
	a.header.SetCategoryName(a.getCategoryName())
	a.header.SetCurrentViewType(a.getViewTypeString())

	// Set category tabs data in header for InfoPaneView
	categories := []string{}
	resourceTabs := [][]string{}
	for _, cat := range components.Categories {
		categories = append(categories, cat.Name)
		resourceTabs = append(resourceTabs, cat.Resources)
	}
	a.header.SetCategories(categories)
	a.header.SetResourceTabs(resourceTabs)
	a.header.SetActiveCategory(a.categoryTabs.ActiveCategory())
	a.header.SetActiveResourceIdx(a.categoryTabs.ActiveResourceIndex())
	a.header.SetHighlightTabs(a.categoryTabs.IsHighlightActive())

	// Use cached header if dirty flag hasn't changed
	headerBox := a.cachedHeader
	if a.lastRenderedHeader != a.headerDirtyFlag || headerBox == "" {
		var headerLines []string
		infoPaneLines := strings.Split(a.header.InfoPaneView(), "\n")
		for _, line := range infoPaneLines {
			headerLines = append(headerLines, padLineWithBackground(line, width))
		}
		headerBox = strings.Join(headerLines, "\n")
		a.cachedHeader = headerBox
		a.lastRenderedHeader = a.headerDirtyFlag
	}

	// === 2. Command/filter box (conditional, self-bordered) ===
	commandBox := ""
	if a.inputMode == ModeCommand {
		commandBox = a.renderCommandBox(width) + "\n"
	} else if a.inputMode == ModeFilter {
		commandBox = a.renderFilterBox(width) + "\n"
	}

	// === 3. Build body content (table only) ===
	var bodyContent string
	if view, ok := a.views[a.activeView]; ok {
		view.SetSize(innerWidth, contentHeight)
		bodyContent = view.View()
	}

	// Build resource label for the body border: Pods(ns)[25]
	resourceLabel := a.buildResourceLabel()

	// Wrap body in frame with centered resource label in top border
	a.frame.SetSize(width, bodyBoxHeight)
	bodyBox := a.frame.WrapWithCenteredLabel(bodyContent, resourceLabel)

	// === 4. Footer (1 line: loading status + resource name) ===
	footer := a.renderFooter(width)

	// === 5. Combine all parts ===
	content := headerBox + "\n" + commandBox + bodyBox + "\n" + footer

	// Overlay dialog if visible
	if a.dialog.IsVisible() {
		a.dialog.SetSize(a.width, a.height)
		content = a.dialog.ViewOverlay(content)
	}

	// Overlay palette if visible
	if a.palette.IsVisible() {
		overlay := a.palette.ViewCentered(a.width, a.height)
		content = overlay
	}

	// Overlay port forward picker if visible (inline, on top of current view)
	if a.pfPicker.IsVisible() {
		a.pfPicker.SetSize(a.width, a.height)
		content = a.pfPicker.ViewOverlay(content)
	}

	// Overlay scale picker if visible
	if a.scalePicker.IsVisible() {
		a.scalePicker.SetSize(a.width, a.height)
		content = a.scalePicker.ViewOverlay(content)
	}

	// Overlay toasts
	if a.toasts.Count() > 0 {
		a.toasts.SetSize(a.width, a.height)
		content = a.toasts.ViewOverlay(content)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.BackgroundColor = theme.ColorBackground
	v.ForegroundColor = theme.ColorText
	return v
}

// getCategoryName returns the current category name
func (a *App) getCategoryName() string {
	for _, cat := range components.Categories {
		for _, vt := range cat.ViewTypes {
			if a.activeView == ViewType(vt) {
				return cat.Name
			}
		}
	}
	return "Workloads"
}

// getViewTypeString returns the view type as a string for status bar hints
func (a *App) getViewTypeString() string {
	switch a.activeView {
	case ViewPods:
		return "pods"
	case ViewDeployments:
		return "deployments"
	case ViewServices:
		return "services"
	case ViewEndpoints:
		return "endpoints"
	case ViewEndpointSlices:
		return "endpointslices"
	case ViewConfigMaps:
		return "configmaps"
	case ViewSecrets:
		return "secrets"
	case ViewIngresses:
		return "ingresses"
	case ViewPVCs:
		return "pvcs"
	case ViewStatefulSets:
		return "statefulsets"
	case ViewNodes:
		return "nodes"
	case ViewEvents:
		return "events"
	case ViewReplicaSets:
		return "replicasets"
	case ViewDaemonSets:
		return "daemonsets"
	case ViewJobs:
		return "jobs"
	case ViewCronJobs:
		return "cronjobs"
	case ViewContainers:
		return "containers"
	case ViewGenericResource:
		if a.genericView != nil {
			return a.genericView.ResourceKind()
		}
		return "resources"
	case ViewHPAs:
		return "horizontalpodautoscalers"
	case ViewPVs:
		return "persistentvolumes"
	case ViewRoleBindings:
		return "rolebindings"
	case ViewHealth:
		return "health"
	case ViewPulse:
		return "pulse"
	case ViewHelmReleases:
		return "helmreleases"
	case ViewHelmHistory:
		return "helmhistory"
	case ViewHelmValues:
		return "helmvalues"
	case ViewHelmManifest:
		return "helmmanifest"
	case ViewSecretDecode:
		return "secretdecode"
	case ViewHelp:
		return "help"
	case ViewPortForwards:
		return "portforwards"
	case ViewXray:
		return "xray"
	case ViewAPIResources:
		return "api-resources"
	case ViewTimeline:
		return "timeline"
	case ViewDiagnosis:
		return "diagnosis"
	default:
		return "default"
	}
}

// viewTypeFromString converts a string identifier to a ViewType.
// Returns the ViewType and true if recognized, or ViewPods and false if not.
func viewTypeFromString(s string) (ViewType, bool) {
	switch s {
	case "pods":
		return ViewPods, true
	case "deployments":
		return ViewDeployments, true
	case "services":
		return ViewServices, true
	case "endpoints":
		return ViewEndpoints, true
	case "endpointslices":
		return ViewEndpointSlices, true
	case "configmaps":
		return ViewConfigMaps, true
	case "secrets":
		return ViewSecrets, true
	case "ingresses":
		return ViewIngresses, true
	case "pvcs":
		return ViewPVCs, true
	case "statefulsets":
		return ViewStatefulSets, true
	case "nodes":
		return ViewNodes, true
	case "events":
		return ViewEvents, true
	case "replicasets":
		return ViewReplicaSets, true
	case "daemonsets":
		return ViewDaemonSets, true
	case "jobs":
		return ViewJobs, true
	case "cronjobs":
		return ViewCronJobs, true
	case "horizontalpodautoscalers":
		return ViewHPAs, true
	case "persistentvolumes":
		return ViewPVs, true
	case "rolebindings":
		return ViewRoleBindings, true
	case "health":
		return ViewHealth, true
	case "pulse":
		return ViewPulse, true
	case "helmreleases":
		return ViewHelmReleases, true
	case "portforwards":
		return ViewPortForwards, true
	case "xray":
		return ViewXray, true
	case "api-resources":
		return ViewAPIResources, true
	case "timeline":
		return ViewTimeline, true
	case "diagnosis":
		return ViewDiagnosis, true
	default:
		return ViewPods, false
	}
}

// isDrillDownView returns true for views that require a selected resource
// and should not be restored on session restore.
func isDrillDownView(v ViewType) bool {
	switch v {
	case ViewDescribe, ViewLogs, ViewYAML, ViewContainers,
		ViewHelmHistory, ViewHelmValues, ViewHelmManifest,
		ViewSecretDecode, ViewHelp, ViewNamespaceSelect, ViewContextSelect:
		return true
	}
	return false
}

// getViewKind returns the singular Kind name for the current active view.
func (a *App) getViewKind() string {
	switch a.activeView {
	case ViewPods:
		return "Pod"
	case ViewDeployments:
		return "Deployment"
	case ViewServices:
		return "Service"
	case ViewEndpoints:
		return "Endpoints"
	case ViewEndpointSlices:
		return "EndpointSlice"
	case ViewConfigMaps:
		return "ConfigMap"
	case ViewSecrets:
		return "Secret"
	case ViewIngresses:
		return "Ingress"
	case ViewPVCs:
		return "PersistentVolumeClaim"
	case ViewStatefulSets:
		return "StatefulSet"
	case ViewNodes:
		return "Node"
	case ViewEvents:
		return "Event"
	case ViewReplicaSets:
		return "ReplicaSet"
	case ViewDaemonSets:
		return "DaemonSet"
	case ViewJobs:
		return "Job"
	case ViewCronJobs:
		return "CronJob"
	case ViewHPAs:
		return "HorizontalPodAutoscaler"
	case ViewPVs:
		return "PersistentVolume"
	case ViewRoleBindings:
		return "RoleBinding"
	case ViewHelmReleases:
		return "HelmRelease"
	case ViewGenericResource:
		if a.genericView != nil {
			return a.genericView.KindName()
		}
		return "Resource"
	default:
		return "Resource"
	}
}


// renderCommandBox renders the command input inside a bordered box (k9s style)
func (a *App) renderCommandBox(width int) string {
	borderStyle := lipgloss.NewStyle().
		Foreground(theme.ColorFrameBorder).
		Background(theme.ColorBackground)
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	boxInner := width - 2 // space inside the left/right borders

	// Top border
	top := borderStyle.Render("╭") + borderStyle.Render(strings.Repeat("─", boxInner)) + borderStyle.Render("╮")

	// Content line: border + command input + border
	cmdContent := a.commandInput.View()
	cmdWidth := lipgloss.Width(cmdContent)
	if cmdWidth < boxInner {
		cmdContent = cmdContent + bgStyle.Render(strings.Repeat(" ", boxInner-cmdWidth))
	}
	middle := borderStyle.Render("│") + cmdContent + borderStyle.Render("│")

	// Bottom border
	bottom := borderStyle.Render("╰") + borderStyle.Render(strings.Repeat("─", boxInner)) + borderStyle.Render("╯")

	return top + "\n" + middle + "\n" + bottom
}

// renderFilterBox renders the filter input inside a bordered box (same layout as command box, teal border)
func (a *App) renderFilterBox(width int) string {
	borderStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Background(theme.ColorBackground)
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	boxInner := width - 2 // space inside the left/right borders

	// Top border
	top := borderStyle.Render("╭") + borderStyle.Render(strings.Repeat("─", boxInner)) + borderStyle.Render("╮")

	// Content line: border + filter input + border
	filterContent := a.searchInput.View()
	filterWidth := lipgloss.Width(filterContent)
	if filterWidth < boxInner {
		filterContent = filterContent + bgStyle.Render(strings.Repeat(" ", boxInner-filterWidth))
	}
	middle := borderStyle.Render("│") + filterContent + borderStyle.Render("│")

	// Bottom border
	bottom := borderStyle.Render("╰") + borderStyle.Render(strings.Repeat("─", boxInner)) + borderStyle.Render("╯")

	return top + "\n" + middle + "\n" + bottom
}

// buildResourceLabel returns a styled resource label like "Pods(ns)[25]<filter>"
// for embedding in the body frame's top border
func (a *App) buildResourceLabel() string {
	labelStyle := lipgloss.NewStyle().
		Foreground(theme.ColorHighlight).
		Background(theme.ColorBackground).
		Bold(true)
	mutedStyle := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Background(theme.ColorBackground)

	viewName := ViewName(a.activeView)
	if a.activeView == ViewGenericResource && a.genericView != nil {
		viewName = a.genericView.Name()
	}
	ns := a.namespace
	if ns == "" || a.isClusterScopedView() {
		ns = "all"
	}
	// Override with drill-down context when active
	if a.drillContext != "" {
		ns = a.drillContext
	}

	count := ""
	if view, ok := a.views[a.activeView]; ok {
		if rc, ok := view.(RowCounter); ok {
			count = fmt.Sprintf("[%d]", rc.RowCount())
		}
	}

	// Check for active filter (k9s-style <filter> indicator)
	filterText := ""
	if view, ok := a.views[a.activeView]; ok {
		if ta, ok := view.(views.TableAccess); ok {
			filterText = ta.GetTable().GetFilter()
		}
	}

	result := labelStyle.Render(viewName) +
		mutedStyle.Render("("+ns+")") +
		mutedStyle.Render(count)

	if filterText != "" {
		display := filterText
		if len(display) > 20 {
			display = display[:17] + "..."
		}
		filterStyle := lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Background(theme.ColorBackground)
		result += filterStyle.Render("<" + display + ">")
	}

	return result
}

// renderFooter renders a 1-line footer below the body box
// Left: resource name (e.g. "Pods")
// Center: loading status (vanishes when loaded)
func (a *App) renderFooter(width int) string {
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	labelStyle := lipgloss.NewStyle().
		Background(theme.ColorBackground).
		Foreground(theme.ColorHighlight).
		Bold(true)

	// Left: breadcrumb trail or simple resource name
	var left string
	if len(a.viewStack) > 0 {
		sepStyle := lipgloss.NewStyle().
			Background(theme.ColorBackground).
			Foreground(theme.ColorMuted)
		parts := make([]string, 0, len(a.viewStack)+1)
		for _, entry := range a.viewStack {
			parts = append(parts, labelStyle.Render(ViewName(entry.View)))
		}
		parts = append(parts, labelStyle.Render(ViewName(a.activeView)))
		left = bgStyle.Render(" ") + strings.Join(parts, sepStyle.Render(" > "))
	} else {
		left = bgStyle.Render(" ") + labelStyle.Render(ViewName(a.activeView))
	}
	leftWidth := lipgloss.Width(left)

	// Center: footer message (copy confirmation, loading, etc.)
	center := ""
	centerWidth := 0
	if a.footerMessage != "" && time.Now().Before(a.footerExpiresAt) {
		msgStyle := lipgloss.NewStyle().
			Background(theme.ColorBackground).
			Foreground(theme.ColorSuccess)
		center = msgStyle.Render(a.footerMessage)
		centerWidth = lipgloss.Width(center)
	} else if a.loading {
		loadingStyle := lipgloss.NewStyle().
			Background(theme.ColorBackground).
			Foreground(theme.ColorWarning)
		center = loadingStyle.Render("Loading...")
		centerWidth = lipgloss.Width(center)
	}

	// Build line: [left] [pad] [center] [pad]
	// Center the loading text in the space right of the left label
	availableForCenter := width - leftWidth
	leftPad := 0
	rightPad := 0
	if centerWidth > 0 {
		leftPad = (availableForCenter - centerWidth) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		rightPad = availableForCenter - leftPad - centerWidth
		if rightPad < 0 {
			rightPad = 0
		}
	} else {
		rightPad = availableForCenter
	}

	line := left +
		bgStyle.Render(strings.Repeat(" ", leftPad)) +
		center +
		bgStyle.Render(strings.Repeat(" ", rightPad))
	return padLineWithBackground(line, width)
}

// padLineWithBackground pads a single line to full width with background
func padLineWithBackground(line string, width int) string {
	lineWidth := lipgloss.Width(line)
	if lineWidth >= width {
		return line
	}
	padding := lipgloss.NewStyle().
		Background(theme.ColorBackground).
		Render(strings.Repeat(" ", width-lineWidth))
	return line + padding
}

