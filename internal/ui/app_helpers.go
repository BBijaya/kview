package ui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/views"
)

// AutoRefreshTickMsg is sent for auto-refresh ticks
type AutoRefreshTickMsg struct{}

// propagateNamespace sets the namespace on all views and resets selection to top
func (a *App) propagateNamespace(ns string) {
	for _, view := range a.views {
		view.SetNamespace(ns)
		view.ResetSelection()
	}
}

// switchView performs direct/top-level navigation (tabs, commands, palette).
// Clears the drill-down stack and any drill-down filters.
func (a *App) switchView(viewType ViewType) tea.Cmd {
	a.clearDrillDownState()
	return a.doSwitchView(viewType)
}

// drillDown performs drill-down navigation (Enter key).
// Pushes the current view onto the stack so goBack() can return to it.
// Saves the current view's filter so it can be restored on goBack().
func (a *App) drillDown(viewType ViewType) tea.Cmd {
	// Save filter of the current view before drill-down
	if view, ok := a.views[a.activeView]; ok {
		if ta, ok := view.(views.TableAccess); ok {
			if f := ta.GetTable().GetFilter(); f != "" {
				a.savedFilters[a.activeView] = f
			}
		}
	}
	a.viewStack = append(a.viewStack, a.activeView)
	return a.doSwitchView(viewType)
}

// goBack performs backward navigation (Escape/GoBackMsg).
// Pops from the stack and cleans up the current view's drill-down state.
// Restores any saved filter on the destination view.
func (a *App) goBack() tea.Cmd {
	if len(a.viewStack) == 0 {
		return a.doSwitchView(a.previousView) // legacy fallback
	}
	// Clean up when leaving drill-down views
	if a.activeView == ViewPods {
		if pv, ok := a.views[ViewPods].(*views.PodsView); ok {
			pv.ClearOwnerFilter()
		}
	}
	prev := a.viewStack[len(a.viewStack)-1]
	a.viewStack = a.viewStack[:len(a.viewStack)-1]
	cmd := a.doSwitchView(prev)

	// Restore saved filter on the destination view
	if f, ok := a.savedFilters[prev]; ok {
		if view, ok := a.views[prev]; ok {
			if ta, ok := view.(views.TableAccess); ok {
				ta.GetTable().SetFilter(f)
			}
		}
		delete(a.savedFilters, prev)
	}

	return cmd
}

// clearDrillDownState resets all drill-down navigation state.
func (a *App) clearDrillDownState() {
	a.viewStack = nil
	a.savedFilters = make(map[ViewType]string)
	if pv, ok := a.views[ViewPods].(*views.PodsView); ok {
		pv.ClearOwnerFilter()
	}
}

// doSwitchView is the shared implementation that performs the actual view switch.
func (a *App) doSwitchView(viewType ViewType) tea.Cmd {
	if viewType == a.activeView {
		return nil
	}

	// Clear any active filter on the old view
	if a.searchInput != nil {
		a.searchInput.Clear()
		a.searchInput.Hide()
		if a.inputMode == ModeFilter {
			a.inputMode = ModeNormal
		}
	}
	if view, ok := a.views[a.activeView]; ok {
		if ta, ok := view.(views.TableAccess); ok {
			ta.GetTable().SetFilter("")
		}
	}

	a.previousView = a.activeView
	a.activeView = viewType
	a.loading = true
	a.invalidateHeader()
	a.tabs.SetActive(int(viewType))
	a.tabBar.SetActive(int(viewType))
	a.categoryTabs.SetActiveByGlobalIndex(int(viewType))
	a.header.SetActiveTab(int(viewType))
	a.header.SetActiveResourceIdx(a.categoryTabs.ActiveResourceIndex())
	viewName := ViewName(viewType)
	if viewType == ViewGenericResource && a.genericView != nil {
		viewName = a.genericView.Name()
	}
	a.header.SetViewName(viewName)
	a.header.SetCategoryName(a.getCategoryName())

	var cmds []tea.Cmd

	if a.isInformerView(viewType) {
		// Informer-backed view: start informer if needed, poll immediately
		a.ensureInformer(viewType)
		cmds = append(cmds, a.dataPollImmediateCmd())
	} else {
		// Non-informer views (describe, logs, yaml, graph, containers, etc.)
		if view, ok := a.views[viewType]; ok {
			cmds = append(cmds, view.Init())
		}
	}

	return tea.Batch(cmds...)
}

func (a *App) updateSizes() {
	// Body inner width (frame border offset)
	innerWidth := a.width - 2

	// Header gets full terminal width (no border)
	a.header.SetWidth(a.width)
	// Other components use body inner width
	a.statusBar.SetWidth(innerWidth)
	a.tabs.SetWidth(innerWidth)
	a.tabBar.SetWidth(innerWidth)
	a.categoryTabs.SetWidth(innerWidth)
	a.palette.SetSize(a.width, a.height)
	a.dialog.SetWidth(min(50, a.width-10))
	a.commandInput.SetWidth(innerWidth)
	a.searchInput.SetWidth(innerWidth)

	// Layout:
	// Header: 7 info lines (borderless)
	// Command box (conditional): 3 lines
	// Body box: remaining height, with 2 borders
	// Footer: 1 line (resource name + loading status)
	headerHeight := 7
	footerHeight := 1
	commandBoxHeight := 0
	if a.inputMode == ModeCommand || a.inputMode == ModeFilter {
		commandBoxHeight = 3
	}
	bodyBoxHeight := a.height - headerHeight - commandBoxHeight - footerHeight
	if bodyBoxHeight < 5 {
		bodyBoxHeight = 5
	}
	bodyInnerHeight := bodyBoxHeight - 2
	contentHeight := bodyInnerHeight
	if contentHeight < 1 {
		contentHeight = 1
	}
	for _, view := range a.views {
		view.SetSize(innerWidth, contentHeight)
	}
}

func (a *App) updateStatusBar() {
	if a.statusMessage != "" {
		a.statusBar.SetMessage(a.statusMessage, a.statusIsError)
	} else {
		a.statusBar.ClearMessage()
	}
}

// activeTable returns the table component from the active view, if it supports TableAccess.
func (a *App) activeTable() *components.Table {
	if view, ok := a.views[a.activeView]; ok {
		if ta, ok := view.(views.TableAccess); ok {
			return ta.GetTable()
		}
	}
	return nil
}

// invalidateHeader bumps the header dirty flag so the next View() re-renders it.
func (a *App) invalidateHeader() {
	a.headerDirtyFlag++
}

func (a *App) setStatus(message string, isError bool) {
	a.statusMessage = message
	a.statusIsError = isError
	a.statusExpiresAt = time.Now().Add(5 * time.Second)
}

func (a *App) setFooterMessage(message string) {
	a.footerMessage = message
	a.footerExpiresAt = time.Now().Add(3 * time.Second)
}

func (a *App) tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}

// loadCommandCompletions fetches namespaces, contexts, and port-forward IDs for command input autocompletion
func (a *App) loadCommandCompletions() tea.Cmd {
	return func() tea.Msg {
		namespaces, _ := a.client.GetNamespaces(context.Background())
		var contextNames []string
		if contexts, err := k8s.GetContexts(); err == nil {
			contextNames = make([]string, len(contexts))
			for i, c := range contexts {
				contextNames[i] = c.Name
			}
		}
		var pfIDs []string
		for _, s := range a.pfManager.ActiveSessions() {
			pfIDs = append(pfIDs, fmt.Sprintf("%d", s.ID))
		}
		return CommandCompletionsMsg{Namespaces: namespaces, Contexts: contextNames, PortForwardIDs: pfIDs}
	}
}

// loadNamespaces switches to the namespace select view
func (a *App) loadNamespaces() tea.Cmd {
	return a.switchView(ViewNamespaceSelect)
}

// loadContexts loads contexts for the picker
func (a *App) loadContexts() tea.Cmd {
	cmd := a.contextPicker.ShowLoading()
	return tea.Batch(cmd, func() tea.Msg {
		contexts, err := k8s.GetContexts()
		if err != nil {
			return ContextsLoadedMsg{Err: err}
		}
		return ContextsLoadedMsg{Contexts: contexts}
	})
}

// switchContext switches to a new Kubernetes context
func (a *App) switchContext(contextName string) tea.Cmd {
	return func() tea.Msg {
		newClient, err := k8s.NewClientForContext(contextName)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to switch context: %w", err)}
		}

		// Update client reference
		a.client = newClient
		a.context = contextName
		a.header.SetContext(contextName)
		a.header.SetServerVersion(newClient.ServerVersion())
		a.header.SetAccessMode(newClient.CheckWriteAccess(context.Background()))

		// Update client in all views that support it
		for _, view := range a.views {
			if setter, ok := view.(views.ClientSetter); ok {
				setter.SetClient(newClient)
			}
		}

		// Update details panel and special views clients
		a.detailsPanel.SetClient(newClient)
		a.describeView.SetClient(newClient)
		a.logsView.SetClient(newClient)
		a.yamlView.SetClient(newClient)
		a.nsSelectView.SetClient(newClient)

		// Update informer clients
		for _, inf := range a.informers {
			inf.SetClient(newClient)
		}

		return ContextSwitchedMsg{Context: contextName}
	}
}

// autoRefreshCmd returns a command for auto-refresh tick
func (a *App) autoRefreshCmd() tea.Cmd {
	return tea.Tick(a.refreshInterval, func(t time.Time) tea.Msg {
		return AutoRefreshTickMsg{}
	})
}

// fetchMetrics fetches cluster CPU/MEM metrics
func (a *App) fetchMetrics() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		metrics, err := a.client.GetClusterMetrics(ctx)
		if err != nil {
			return MetricsUpdatedMsg{Err: err}
		}
		return MetricsUpdatedMsg{CPU: metrics.CPUUsage, MEM: metrics.MemUsage}
	}
}

// metricsTickCmd schedules the next metrics fetch
func (a *App) metricsTickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return MetricsTickMsg{}
	})
}

// podMetricsTickCmd schedules the next pod metrics fetch (15s interval)
func (a *App) podMetricsTickCmd() tea.Cmd {
	return tea.Tick(15*time.Second, func(t time.Time) tea.Msg {
		return PodMetricsTickMsg{}
	})
}

// nodeMetricsTickCmd schedules the next node metrics fetch (15s interval)
func (a *App) nodeMetricsTickCmd() tea.Cmd {
	return tea.Tick(15*time.Second, func(t time.Time) tea.Msg {
		return NodeMetricsTickMsg{}
	})
}

// initInformers creates all 14 resource informers.
func (a *App) initInformers() {
	a.informers = map[ViewType]*k8s.ResourceInformer{
		ViewPods: k8s.NewResourceInformer(a.client, "pods", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListPods(ctx, ns)
		}),
		ViewDeployments: k8s.NewResourceInformer(a.client, "deployments", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListDeployments(ctx, ns)
		}),
		ViewServices: k8s.NewResourceInformer(a.client, "services", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListServices(ctx, ns)
		}),
		ViewConfigMaps: k8s.NewResourceInformer(a.client, "configmaps", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListConfigMaps(ctx, ns)
		}),
		ViewSecrets: k8s.NewResourceInformer(a.client, "secrets", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListSecrets(ctx, ns)
		}),
		ViewIngresses: k8s.NewResourceInformer(a.client, "ingresses", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListIngresses(ctx, ns)
		}),
		ViewPVCs: k8s.NewResourceInformer(a.client, "persistentvolumeclaims", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListPVCs(ctx, ns)
		}),
		ViewStatefulSets: k8s.NewResourceInformer(a.client, "statefulsets", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListStatefulSets(ctx, ns)
		}),
		ViewNodes: k8s.NewResourceInformer(a.client, "nodes", func(ctx context.Context, c k8s.Client, _ string) (any, error) {
			return c.ListNodes(ctx)
		}),
		ViewEvents: k8s.NewResourceInformer(a.client, "events", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListEvents(ctx, ns)
		}),
		ViewReplicaSets: k8s.NewResourceInformer(a.client, "replicasets", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListReplicaSets(ctx, ns)
		}),
		ViewDaemonSets: k8s.NewResourceInformer(a.client, "daemonsets", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListDaemonSets(ctx, ns)
		}),
		ViewJobs: k8s.NewResourceInformer(a.client, "jobs", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListJobs(ctx, ns)
		}),
		ViewCronJobs: k8s.NewResourceInformer(a.client, "cronjobs", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListCronJobs(ctx, ns)
		}),
		ViewHPAs: k8s.NewResourceInformer(a.client, "horizontalpodautoscalers", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListHPAs(ctx, ns)
		}),
		ViewPVs: k8s.NewResourceInformer(a.client, "persistentvolumes", func(ctx context.Context, c k8s.Client, _ string) (any, error) {
			return c.ListPVs(ctx)
		}),
		ViewRoleBindings: k8s.NewResourceInformer(a.client, "rolebindings", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListRoleBindings(ctx, ns)
		}),
		ViewHelmReleases: k8s.NewResourceInformer(a.client, "helmreleases", func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.ListHelmReleases(ctx, ns)
		}),
	}
	a.lastSeenGen = make(map[ViewType]uint64)
}

// ensureInformer starts the informer for the given view type if not already started.
func (a *App) ensureInformer(vt ViewType) {
	inf, ok := a.informers[vt]
	if !ok {
		return
	}
	if !inf.Started() {
		inf.Start(a.namespace)
	}
}

// stopAllInformers stops all running informers and resets generation tracking.
func (a *App) stopAllInformers() {
	for _, inf := range a.informers {
		inf.Stop()
	}
	a.lastSeenGen = make(map[ViewType]uint64)
}

// setInformersNamespace updates the namespace on all started informers.
// Skips ViewNodes since it's cluster-scoped.
func (a *App) setInformersNamespace(ns string) {
	for vt, inf := range a.informers {
		if vt == ViewNodes || vt == ViewPVs {
			continue // cluster-scoped
		}
		// Skip cluster-scoped generic resources
		if vt == ViewGenericResource && a.genericResourceKind != "" {
			if reg := a.client.APIResources(); reg != nil {
				if info, found := reg.Lookup(a.genericResourceKind); found && !info.Namespaced {
					continue
				}
			}
		}
		if inf.Started() {
			inf.SetNamespace(ns)
		}
	}
	// Reset gen tracking so new data is dispatched
	a.lastSeenGen = make(map[ViewType]uint64)
}

// isInformerView returns true if the view type is backed by an informer.
func (a *App) isInformerView(vt ViewType) bool {
	_, ok := a.informers[vt]
	return ok
}

// dataPollTickCmd schedules the next data poll after 2 seconds.
func (a *App) dataPollTickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return DataPollMsg{}
	})
}

// dataPollImmediateCmd returns an immediate data poll command.
func (a *App) dataPollImmediateCmd() tea.Cmd {
	return func() tea.Msg {
		return DataPollMsg{}
	}
}

// handleDataPoll reads informer snapshots and dispatches LoadedMsg if data changed.
func (a *App) handleDataPoll() tea.Cmd {
	var cmds []tea.Cmd

	// Only poll the active view's informer
	if a.isInformerView(a.activeView) {
		if cmd := a.dispatchInformerData(a.activeView); cmd != nil {
			cmds = append(cmds, cmd)
			a.loading = false
		} else {
			// Informer hasn't produced data yet (gen==0), poll faster
			inf, ok := a.informers[a.activeView]
			if ok && inf.Generation() == 0 {
				cmds = append(cmds, tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
					return DataPollMsg{}
				}))
				return tea.Batch(cmds...)
			}
		}
	}

	// Schedule next poll
	cmds = append(cmds, a.dataPollTickCmd())
	return tea.Batch(cmds...)
}

// dispatchInformerData reads a snapshot from the informer and returns a typed
// LoadedMsg if the generation has changed since last dispatch.
func (a *App) dispatchInformerData(vt ViewType) tea.Cmd {
	inf, ok := a.informers[vt]
	if !ok {
		return nil
	}

	data, gen, err := inf.Snapshot()
	if gen == 0 && err == nil {
		return nil // Not loaded yet
	}

	if gen == a.lastSeenGen[vt] {
		return nil // No change
	}
	a.lastSeenGen[vt] = gen

	// Dispatch the appropriate typed message
	switch vt {
	case ViewPods:
		pods, _ := data.([]k8s.PodInfo)
		return func() tea.Msg { return views.PodsLoadedMsg{Pods: pods, Err: err} }
	case ViewDeployments:
		deps, _ := data.([]k8s.DeploymentInfo)
		return func() tea.Msg { return views.DeploymentsLoadedMsg{Deployments: deps, Err: err} }
	case ViewServices:
		svcs, _ := data.([]k8s.ServiceInfo)
		return func() tea.Msg { return views.ServicesLoadedMsg{Services: svcs, Err: err} }
	case ViewConfigMaps:
		cms, _ := data.([]k8s.ConfigMapInfo)
		return func() tea.Msg { return views.ConfigMapsLoadedMsg{ConfigMaps: cms, Err: err} }
	case ViewSecrets:
		secs, _ := data.([]k8s.SecretInfo)
		return func() tea.Msg { return views.SecretsLoadedMsg{Secrets: secs, Err: err} }
	case ViewIngresses:
		ings, _ := data.([]k8s.IngressInfo)
		return func() tea.Msg { return views.IngressesLoadedMsg{Ingresses: ings, Err: err} }
	case ViewPVCs:
		pvcs, _ := data.([]k8s.PVCInfo)
		return func() tea.Msg { return views.PVCsLoadedMsg{PVCs: pvcs, Err: err} }
	case ViewStatefulSets:
		stss, _ := data.([]k8s.StatefulSetInfo)
		return func() tea.Msg { return views.StatefulSetsLoadedMsg{StatefulSets: stss, Err: err} }
	case ViewNodes:
		nodes, _ := data.([]k8s.NodeInfo)
		return func() tea.Msg { return views.NodesLoadedMsg{Nodes: nodes, Err: err} }
	case ViewEvents:
		events, _ := data.([]k8s.EventInfo)
		return func() tea.Msg { return views.EventsLoadedMsg{Events: events, Err: err} }
	case ViewReplicaSets:
		rss, _ := data.([]k8s.ReplicaSetInfo)
		return func() tea.Msg { return views.ReplicaSetsLoadedMsg{ReplicaSets: rss, Err: err} }
	case ViewDaemonSets:
		dss, _ := data.([]k8s.DaemonSetInfo)
		return func() tea.Msg { return views.DaemonSetsLoadedMsg{DaemonSets: dss, Err: err} }
	case ViewJobs:
		jobs, _ := data.([]k8s.JobInfo)
		return func() tea.Msg { return views.JobsLoadedMsg{Jobs: jobs, Err: err} }
	case ViewCronJobs:
		cjs, _ := data.([]k8s.CronJobInfo)
		return func() tea.Msg { return views.CronJobsLoadedMsg{CronJobs: cjs, Err: err} }
	case ViewGenericResource:
		resources, _ := data.([]k8s.Resource)
		return func() tea.Msg { return views.GenericResourcesLoadedMsg{Resources: resources, Err: err} }
	case ViewHPAs:
		hpas, _ := data.([]k8s.HPAInfo)
		return func() tea.Msg { return views.HPAsLoadedMsg{HPAs: hpas, Err: err} }
	case ViewPVs:
		pvs, _ := data.([]k8s.PVInfo)
		return func() tea.Msg { return views.PVsLoadedMsg{PVs: pvs, Err: err} }
	case ViewRoleBindings:
		rbs, _ := data.([]k8s.RoleBindingInfo)
		return func() tea.Msg { return views.RoleBindingsLoadedMsg{RoleBindings: rbs, Err: err} }
	case ViewHelmReleases:
		releases, _ := data.([]k8s.HelmReleaseInfo)
		return func() tea.Msg { return views.HelmReleasesLoadedMsg{Releases: releases, Err: err} }
	}
	return nil
}

// isResourceListView returns true if the active view is a resource list (not describe/logs/yaml/xray/timeline/containers)
func (a *App) isResourceListView() bool {
	switch a.activeView {
	case ViewDescribe, ViewLogs, ViewYAML, ViewNamespaceSelect, ViewXray, ViewTimeline, ViewContainers, ViewHelmValues, ViewHelmManifest, ViewSecretDecode, ViewHelp:
		return false
	default:
		return true
	}
}

// switchToGenericResource configures the generic view for a discovered resource and switches to it.
func (a *App) switchToGenericResource(info *k8s.APIResourceInfo) tea.Cmd {
	a.genericView.SetResource(info.Resource, info.Kind, info.Kind)
	a.genericResourceKind = info.Resource
	a.ensureGenericInformer(info)

	// If already on the generic view (switching between CRDs), force a refresh
	// since doSwitchView short-circuits when viewType == activeView
	if a.activeView == ViewGenericResource {
		a.loading = true
		a.invalidateHeader()
		viewName := a.genericView.Name()
		a.header.SetViewName(viewName)
		return a.dataPollImmediateCmd()
	}
	return a.switchView(ViewGenericResource)
}

// ensureGenericInformer stops any existing generic informer and starts a new one for the given resource.
func (a *App) ensureGenericInformer(info *k8s.APIResourceInfo) {
	if inf, ok := a.informers[ViewGenericResource]; ok {
		inf.Stop()
	}
	resourceName := info.Resource
	a.informers[ViewGenericResource] = k8s.NewResourceInformer(
		a.client, resourceName,
		func(ctx context.Context, c k8s.Client, ns string) (any, error) {
			return c.List(ctx, resourceName, ns)
		},
	)
	delete(a.lastSeenGen, ViewGenericResource)
	a.informers[ViewGenericResource].Start(a.namespace)
}

// showAPIResourcePicker populates and displays the API resource picker overlay.
func (a *App) showAPIResourcePicker() tea.Cmd {
	reg := a.client.APIResources()
	if reg == nil {
		a.setStatus("API resources not available", true)
		return nil
	}
	all := reg.All()
	if len(all) == 0 {
		a.setStatus("API resources not yet discovered", true)
		return nil
	}
	items := make([]components.PickerItem, 0, len(all))
	for _, info := range all {
		group := info.Group
		if group == "" {
			group = "core"
		}
		items = append(items, components.PickerItem{
			ID:    info.Resource,
			Label: info.Resource,
			Desc:  fmt.Sprintf("%s/%s (%s)", group, info.Version, info.Kind),
		})
	}
	a.apiResourcePicker.SetItems(items)
	return nil
}

// editResource opens the selected resource in the user's editor via tea.Exec.
func (a *App) editResource(kind, namespace, name string) tea.Cmd {
	k8sClient, ok := a.client.(*k8s.K8sClient)
	if !ok {
		a.setStatus("Edit not available: no cluster connection", true)
		return a.toasts.PushError("Edit Failed", "No cluster connection")
	}
	editCmd := k8s.NewEditExecCmd(k8sClient, kind, namespace, name)
	a.execing = true
	return tea.Exec(editCmd, func(err error) tea.Msg {
		if err != nil {
			return EditExitMsg{Err: err}
		}
		return EditExitMsg{Applied: editCmd.Applied()}
	})
}
