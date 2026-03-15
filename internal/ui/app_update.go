package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/commands"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
	"github.com/bijaya/kview/internal/ui/views"
)

// Update handles messages
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.invalidateHeader()
		a.updateSizes()

	case tea.KeyPressMsg:
		// Handle dialog first
		if a.dialog.IsVisible() {
			var cmd tea.Cmd
			a.dialog, cmd = a.dialog.Update(msg)
			return a, cmd
		}

		// Handle namespace picker
		if a.namespacePicker.IsVisible() {
			var cmd tea.Cmd
			a.namespacePicker, cmd = a.namespacePicker.Update(msg)
			return a, cmd
		}

		// Handle API resource picker
		if a.apiResourcePicker.IsVisible() {
			var cmd tea.Cmd
			a.apiResourcePicker, cmd = a.apiResourcePicker.Update(msg)
			return a, cmd
		}

		// Handle port forward picker
		if a.pfPicker.IsVisible() {
			var cmd tea.Cmd
			a.pfPicker, cmd = a.pfPicker.Update(msg)
			return a, cmd
		}

		// Handle scale picker
		if a.scalePicker.IsVisible() {
			var cmd tea.Cmd
			a.scalePicker, cmd = a.scalePicker.Update(msg)
			return a, cmd
		}

		// Handle palette
		if a.palette.IsVisible() {
			var cmd tea.Cmd
			a.palette, cmd = a.palette.Update(msg)
			return a, cmd
		}

		// Handle command input mode
		if a.inputMode == ModeCommand {
			var cmd tea.Cmd
			a.commandInput, cmd = a.commandInput.Update(msg)
			return a, cmd
		}

		// Handle filter input mode
		if a.inputMode == ModeFilter {
			var cmd tea.Cmd
			a.searchInput, cmd = a.searchInput.Update(msg)
			return a, cmd
		}

		// Global keybindings (normal mode)
		switch {
		case key.Matches(msg, DefaultKeyMap().Quit):
			a.quitting = true
			a.pfManager.StopAll()
			a.stopAllInformers()
			if a.store != nil {
				a.store.Close()
			}
			return a, tea.Quit

		case key.Matches(msg, DefaultKeyMap().Command):
			// Enter command mode
			a.inputMode = ModeCommand
			a.commandInput.Show()
			a.invalidateHeader()
			a.updateSizes()
			return a, a.loadCommandCompletions()

		case key.Matches(msg, DefaultKeyMap().Filter):
			// Enter filter mode (resource list views and viewport-searchable views)
			canFilter := a.isResourceListView()
			if !canFilter {
				if _, ok := a.views[a.activeView].(views.ViewportSearcher); ok {
					canFilter = true
				}
			}
			if canFilter {
				a.inputMode = ModeFilter
				a.searchInput.SetWidth(a.width - 2)
				a.searchInput.Show()
				// Pre-populate with existing search pattern
				if vs, ok := a.views[a.activeView].(views.ViewportSearcher); ok {
					if pattern := vs.ActiveSearchPattern(); pattern != "" {
						a.searchInput.SetValue(pattern)
					}
				}
				a.invalidateHeader()
				a.updateSizes()
				return a, nil
			}

		case key.Matches(msg, DefaultKeyMap().Palette):
			a.palette.Show()
			return a, nil

		case key.Matches(msg, DefaultKeyMap().Help):
			if a.activeView == ViewHelp {
				return a, a.goBack()
			}
			return a, a.drillDown(ViewHelp)

		case key.Matches(msg, DefaultKeyMap().NextTab):
			// Move to next resource/view
			newIdx := a.categoryTabs.NextResource()
			a.tabs.SetActive(newIdx)
			a.tabBar.SetActive(newIdx)
			return a, a.switchView(ViewType(newIdx))

		case key.Matches(msg, DefaultKeyMap().PrevTab):
			// Move to previous resource/view
			newIdx := a.categoryTabs.PrevResource()
			a.tabs.SetActive(newIdx)
			a.tabBar.SetActive(newIdx)
			return a, a.switchView(ViewType(newIdx))

		case key.Matches(msg, DefaultKeyMap().Right):
			// Move to next category
			newIdx := a.categoryTabs.NextCategory()
			a.tabs.SetActive(newIdx)
			a.tabBar.SetActive(newIdx)
			return a, a.switchView(ViewType(newIdx))

		case key.Matches(msg, DefaultKeyMap().Left):
			// Move to previous category
			newIdx := a.categoryTabs.PrevCategory()
			a.tabs.SetActive(newIdx)
			a.tabBar.SetActive(newIdx)
			return a, a.switchView(ViewType(newIdx))

		case key.Matches(msg, DefaultKeyMap().DetailsPanel):
			a.showDetailsPanel = !a.showDetailsPanel
			if a.showDetailsPanel && a.selectedResource != nil {
				return a, a.detailsPanel.SetResource(
					a.selectedResource.Kind,
					a.selectedResource.Namespace,
					a.selectedResource.Name,
				)
			}
			return a, nil

		case key.Matches(msg, DefaultKeyMap().Escape):
			// When in a drill-down (view stack non-empty), Escape goes back.
			// Views that handle Escape themselves (health, pulse, logs, etc.)
			// return early before reaching the App's global keybindings.
			if len(a.viewStack) > 0 {
				return a, a.goBack()
			}
			// If a filter is active, clear it (like k9s).
			if view, ok := a.views[a.activeView]; ok {
				if ta, ok := view.(views.TableAccess); ok {
					if ta.GetTable().GetFilter() != "" {
						ta.GetTable().SetFilter("")
						a.searchInput.Clear()
						a.invalidateHeader()
					}
				}
			}

		case key.Matches(msg, DefaultKeyMap().DeltaFilter):
			if a.isResourceListView() {
				if v, ok := a.views[a.activeView].(views.TableAccess); ok {
					tbl := v.GetTable()
					tbl.ToggleDeltaFilter()
					a.header.SetDeltaFilter(tbl.IsDeltaFilterActive())
					a.invalidateHeader()
				}
			}
			return a, nil

		case key.Matches(msg, DefaultKeyMap().AutoRefresh):
			a.autoRefresh = !a.autoRefresh
			if a.autoRefresh {
				a.setStatus("Auto-refresh enabled", false)
				return a, a.autoRefreshCmd()
			}
			a.setStatus("Auto-refresh disabled", false)
			return a, nil

		case key.Matches(msg, DefaultKeyMap().Refresh):
			if a.isInformerView(a.activeView) {
				if inf, ok := a.informers[a.activeView]; ok {
					inf.Invalidate()
				}
				a.loading = true
				return a, a.dataPollImmediateCmd()
			}
			if view, ok := a.views[a.activeView]; ok {
				a.loading = true
				return a, view.Refresh()
			}

		default:
			// Handle number keys for category-relative resource selection
			keyStr := msg.String()
			if len(keyStr) == 1 && keyStr >= "1" && keyStr <= "9" {
				num := int(keyStr[0] - '0')
				if newIdx, ok := a.categoryTabs.SelectResourceByNumber(num); ok {
					return a, a.switchView(ViewType(newIdx))
				}
			}
		}

		// Copy keybinding: copies resource name in list views, content in detail views
		if key.Matches(msg, DefaultKeyMap().CopyName) {
			var copyText string
			var footerMsg string
			switch a.activeView {
			case ViewDescribe, ViewYAML, ViewLogs, ViewSecretDecode, ViewHelmValues, ViewHelmManifest:
				switch a.activeView {
				case ViewDescribe:
					copyText = a.describeView.Content()
				case ViewYAML:
					copyText = a.yamlView.Content()
				case ViewLogs:
					copyText = a.logsView.Content()
				case ViewSecretDecode:
					copyText = a.secretDecodeView.Content()
				case ViewHelmValues:
					copyText = a.helmValuesView.Content()
				case ViewHelmManifest:
					copyText = a.helmManifestView.Content()
				}
				footerMsg = "Content copied to clipboard"
			default:
				if view, ok := a.views[a.activeView]; ok {
					name := view.SelectedName()
					if name != "" {
						copyText = name
						footerMsg = name + " copied to clipboard"
					}
				}
			}
			if copyText != "" {
				err := clipboard.WriteAll(copyText)
				if err != nil {
					a.setFooterMessage("Copy failed: " + err.Error())
					return a, a.toasts.PushError("Copy Failed", err.Error())
				}
				a.setFooterMessage(footerMsg)
				return a, a.toasts.PushSuccess("Copied", footerMsg)
			} else {
				a.setFooterMessage("Nothing to copy")
			}
			return a, nil
		}

		// Xray keybinding: opens xray view for selected resource
		if key.Matches(msg, DefaultKeyMap().Xray) && a.isResourceListView() {
			res := a.selectedResource
			// Resolve from current view's table if selectedResource is not set
			if res == nil {
				if view, ok := a.views[a.activeView]; ok {
					if ta, ok := view.(views.TableAccess); ok {
						if row := ta.GetTable().SelectedRow(); row != nil {
							name := view.SelectedName()
							ns := a.namespace
							if ns == "" {
								ns = ta.GetTable().SelectedValue(0) // NAMESPACE column
							}
							res = &ResourceSelectedMsg{
								Kind:      a.getViewKind(),
								Resource:  a.getViewTypeString(),
								Namespace: ns,
								Name:      name,
								UID:       row.ID,
							}
						}
					}
				}
			}
			if res != nil {
				a.xrayView.SetModeForResource(
					res.Kind,
					res.Name,
					res.Namespace,
					res.UID,
				)
				return a, a.drillDownWithContext(ViewXray, res.Kind+"/"+res.Name)
			}
		}

		// Edit keybinding: opens resource in $EDITOR
		if key.Matches(msg, DefaultKeyMap().Edit) && a.isResourceListView() {
			if view, ok := a.views[a.activeView]; ok {
				name := view.SelectedName()
				if name == "" {
					a.setStatus("No resource selected", true)
					return a, nil
				}
				kind := a.getViewTypeString()
				ns := a.namespace
				if ns == "" {
					if ta, ok := view.(views.TableAccess); ok {
						ns = ta.GetTable().SelectedValue(0) // NAMESPACE column
					}
				}
				return a, a.editResource(kind, ns, name)
			}
			return a, nil
		}

	case TickMsg:
		// Check if status message should be cleared
		if a.statusMessage != "" && time.Now().After(a.statusExpiresAt) {
			a.statusMessage = ""
			a.statusIsError = false
		}
		// Clear expired footer message
		if a.footerMessage != "" && time.Now().After(a.footerExpiresAt) {
			a.footerMessage = ""
		}
		return a, a.tickCmd()

	case components.CommandExecuteMsg:
		a.inputMode = ModeNormal
		a.invalidateHeader()
		a.updateSizes()
		return a, a.handleCommand(msg.Command, msg.Args)

	case components.CommandCancelMsg:
		a.inputMode = ModeNormal
		a.invalidateHeader()
		a.updateSizes()
		return a, nil

	case CommandCompletionsMsg:
		a.cachedNamespaces = msg.Namespaces
		a.commandInput.SetNamespaces(msg.Namespaces)
		a.commandInput.SetContexts(msg.Contexts)
		a.commandInput.SetPortForwardIDs(msg.PortForwardIDs)
		a.commandInput.SetDiscoveredResources(msg.Resources)
		return a, nil

	case components.FilterChangedMsg:
		if vs, ok := a.views[a.activeView].(views.ViewportSearcher); ok {
			vs.ApplySearch(msg.Value)
		} else if view, ok := a.views[a.activeView]; ok {
			if ta, ok := view.(views.TableAccess); ok {
				ta.GetTable().SetFilter(msg.Value)
			}
		}

	case components.FilterClosedMsg:
		a.inputMode = ModeNormal
		if vs, ok := a.views[a.activeView].(views.ViewportSearcher); ok {
			if msg.Submitted {
				vs.ApplySearch(a.searchInput.Value())
			} else {
				vs.ClearSearch()
			}
		}
		a.invalidateHeader()
		a.updateSizes()

	// Handle command package message types
	case commands.SwitchViewMsg:
		return a, a.switchView(ViewType(msg.View))

	// Handle theme package message types (from components)
	case theme.SwitchViewMsg:
		return a, a.switchView(ViewType(msg.View))

	// Handle generic drill-down to a view type (pushes current view onto stack)
	case views.DrillDownViewMsg:
		return a, a.drillDown(ViewType(msg.View))

	// Handle drill-down from Deployment to filtered Pods
	case views.DrillDownNodeMsg:
		// Save current namespace and switch to all-namespaces for node pods
		a.drillDownSavedNS = a.namespace
		if a.namespace != "" {
			a.namespace = ""
			a.propagateNamespace("")
			a.header.SetNamespace("all")
			a.setInformersNamespace("")
		}
		if pv, ok := a.views[ViewPods].(*views.PodsView); ok {
			pv.SetNodeFilter(msg.NodeName)
		}
		a.header.SetViewName("Pods (" + msg.NodeName + ")")
		return a, a.drillDownWithContext(ViewPods, msg.NodeName)

	case views.DrillDownDeploymentMsg:
		if pv, ok := a.views[ViewPods].(*views.PodsView); ok {
			pv.SetOwnerFilter("Deployment", msg.DeploymentName)
		}
		a.header.SetViewName("Pods (" + msg.DeploymentName + ")")
		return a, a.drillDownWithContext(ViewPods, msg.DeploymentName)

	// Handle drill-down from Pod to Containers
	case views.DrillDownContainersMsg:
		a.containersView.SetPod(msg.Pod)
		containerCtx := msg.Pod.Name
		if msg.Pod.Namespace != "" {
			containerCtx = msg.Pod.Namespace + "/" + msg.Pod.Name
		}
		a.header.SetViewName("Containers (" + msg.Pod.Name + ")")
		return a, a.drillDownWithContext(ViewContainers, containerCtx)

	// Handle drill-down from Helm Release to History
	case views.DrillDownHelmHistoryMsg:
		a.helmHistoryView.SetRelease(msg.Namespace, msg.ReleaseName)
		a.header.SetViewName("History (" + msg.ReleaseName + ")")
		return a, a.drillDownWithContext(ViewHelmHistory, msg.ReleaseName)

	// Handle generic drill-down from workload to filtered Pods
	case views.DrillDownToPodsMsg:
		if pv, ok := a.views[ViewPods].(*views.PodsView); ok {
			pv.SetOwnerFilter(msg.OwnerKind, msg.OwnerName)
		}
		a.header.SetViewName("Pods (" + msg.OwnerName + ")")
		return a, a.drillDownWithContext(ViewPods, msg.OwnerName)

	// Handle drill-down from CronJob to filtered Jobs
	case views.DrillDownCronJobMsg:
		if jv, ok := a.views[ViewJobs].(*views.JobsView); ok {
			jv.SetOwnerFilter("CronJob", msg.CronJobName)
		}
		a.header.SetViewName("Jobs (" + msg.CronJobName + ")")
		return a, a.drillDownWithContext(ViewJobs, msg.CronJobName)

	// Handle drill-down from Service to Pods via label selector
	case views.DrillDownServiceMsg:
		if pv, ok := a.views[ViewPods].(*views.PodsView); ok {
			pv.SetLabelSelector(msg.Selector)
		}
		a.header.SetViewName("Pods (" + msg.ServiceName + ")")
		return a, a.drillDownWithContext(ViewPods, msg.ServiceName)

	// Handle navigation from Events to describe involved resource
	case views.NavigateToResourceMsg:
		resource := kindToAPIResource(msg.Kind)
		a.describeView.SetResource(resource, msg.Namespace, msg.Name)
		a.header.SetViewName("Describe (" + msg.Name + ")")
		return a, a.drillDownWithContext(ViewDescribe, msg.Kind+"/"+msg.Name)

	// Handle secret decode view
	case views.DecodeSecretMsg:
		a.secretDecodeView.SetResource(msg.Namespace, msg.Name)
		return a, a.drillDownWithContext(ViewSecretDecode, msg.Name)

	// Handle Helm values/manifest content view
	case views.OpenHelmContentMsg:
		switch msg.Mode {
		case views.HelmContentValues:
			a.helmValuesView.SetRelease(msg.Namespace, msg.ReleaseName, msg.Revision)
			return a, a.drillDownWithContext(ViewHelmValues, msg.ReleaseName)
		case views.HelmContentManifest:
			a.helmManifestView.SetRelease(msg.Namespace, msg.ReleaseName, msg.Revision)
			return a, a.drillDownWithContext(ViewHelmManifest, msg.ReleaseName)
		}

	// Handle combined resource+view message (atomic, no race)
	case views.OpenViewMsg:
		// Set the selected resource first
		a.selectedResource = &ResourceSelectedMsg{
			Kind:      msg.Kind,
			Resource:  msg.Resource,
			Namespace: msg.Namespace,
			Name:      msg.Name,
			UID:       msg.UID,
		}
		// Set up the target view
		targetView := ViewType(msg.TargetView)
		switch targetView {
		case ViewDescribe:
			a.describeView.SetResource(
				msg.Resource,
				msg.Namespace,
				msg.Name,
			)
		case ViewLogs:
			a.logsView.SetPod(msg.Namespace, msg.Name, msg.Container)
		case ViewYAML:
			a.yamlView.SetResource(
				msg.Resource,
				msg.Namespace,
				msg.Name,
			)
		}
		var ctx string
		switch targetView {
		case ViewLogs:
			if msg.Namespace != "" {
				ctx = msg.Namespace + "/" + msg.Name
			} else {
				ctx = msg.Name
			}
			if msg.Container != "" {
				ctx += "/" + msg.Container
			}
		case ViewDescribe, ViewYAML:
			ctx = msg.Kind + "/" + msg.Name
		}
		return a, a.drillDownWithContext(targetView, ctx)

	case commands.RefreshMsg:
		if a.isInformerView(a.activeView) {
			if inf, ok := a.informers[a.activeView]; ok {
				inf.Invalidate()
			}
			cmds = append(cmds, a.dataPollImmediateCmd())
		} else if view, ok := a.views[a.activeView]; ok {
			cmds = append(cmds, view.Refresh())
		}

	case commands.SwitchNamespaceMsg:
		a.namespace = msg.Namespace
		a.propagateNamespace(msg.Namespace)
		if msg.Namespace == "" {
			a.header.SetNamespace("all")
		} else {
			a.header.SetNamespace(msg.Namespace)
		}
		a.invalidateHeader()
		a.loading = true
		a.setInformersNamespace(msg.Namespace)
		cmds = append(cmds, a.dataPollImmediateCmd())

	case commands.ExecuteCommandMsg:
		return a, a.executeCommand(msg.CommandID)

	case StatusMsg:
		a.setStatus(msg.Message, msg.IsError)

	case ContextSwitchedMsg:
		a.invalidateHeader()
		a.setStatus("Switched to context: "+msg.Context, false)
		cmds = append(cmds, a.toasts.PushInfo("Context", "Switched to "+msg.Context))
		// Stop all port forwards on context switch
		a.pfManager.StopAll()
		a.pfManager.SetClient(a.client.GetRestConfig(), a.client.GetClientset())
		// Restart all informers with new client
		a.stopAllInformers()
		a.initInformers()
		a.ensureInformer(a.activeView)
		a.loading = true
		cmds = append(cmds, a.dataPollImmediateCmd())

	case ErrorMsg:
		a.setStatus(msg.Err.Error(), true)
		cmds = append(cmds, a.toasts.PushError("Error", msg.Err.Error()))

	case RefreshMsg:
		if a.isInformerView(a.activeView) {
			if inf, ok := a.informers[a.activeView]; ok {
				inf.Invalidate()
			}
			cmds = append(cmds, a.dataPollImmediateCmd())
		} else if view, ok := a.views[a.activeView]; ok {
			cmds = append(cmds, view.Refresh())
		}

	case ResourceSelectedMsg:
		a.selectedResource = &msg
		a.setStatus(fmt.Sprintf("Selected: %s/%s", msg.Kind, msg.Name), false)

	// Handle message types from views package
	case views.ResourceSelectedMsg:
		a.selectedResource = &ResourceSelectedMsg{
			Kind:      msg.Kind,
			Resource:  msg.Resource,
			Namespace: msg.Namespace,
			Name:      msg.Name,
			UID:       msg.UID,
		}
		a.setStatus(fmt.Sprintf("Selected: %s/%s", msg.Kind, msg.Name), false)
		// Update details panel if visible
		if a.showDetailsPanel {
			cmds = append(cmds, a.detailsPanel.SetResource(msg.Kind, msg.Namespace, msg.Name))
		}

	case views.ConfirmActionMsg:
		a.dialog.ShowConfirm(msg.Title, msg.Message, func() {
			if msg.Action != nil {
				verb, detail, _ := strings.Cut(msg.Title, " ")
				if err := msg.Action(); err != nil {
					a.setStatus("Error: "+err.Error(), true)
					a.pendingActionResult = &actionResult{title: verb + " Failed", errMsg: err.Error(), success: false}
				} else {
					a.setStatus("Action completed", false)
					a.pendingActionResult = &actionResult{title: pastTense(verb), errMsg: detail, success: true}
				}
			}
		}, nil)
		return a, nil

	case views.ShowToastMsg:
		if msg.IsError {
			cmds = append(cmds, a.toasts.PushError(msg.Title, msg.Message))
		} else {
			cmds = append(cmds, a.toasts.PushWarning(msg.Title, msg.Message))
		}

	case views.StatusMsg:
		a.setStatus(msg.Message, msg.IsError)

	case views.LogsSavedMsg:
		if msg.Err != nil {
			a.setStatus("Save failed: "+msg.Err.Error(), true)
			cmds = append(cmds, a.toasts.PushError("Save Failed", msg.Err.Error()))
		} else {
			a.setStatus("Saved: "+msg.Path, false)
			cmds = append(cmds, a.toasts.PushSuccess("Logs Saved", msg.Path))
		}

	case ConfirmActionMsg:
		a.dialog.ShowConfirm(msg.Title, msg.Message, func() {
			if msg.Action != nil {
				verb, detail, _ := strings.Cut(msg.Title, " ")
				if err := msg.Action(); err != nil {
					a.setStatus("Error: "+err.Error(), true)
					a.pendingActionResult = &actionResult{title: verb + " Failed", errMsg: err.Error(), success: false}
				} else {
					a.setStatus("Action completed", false)
					a.pendingActionResult = &actionResult{title: pastTense(verb), errMsg: detail, success: true}
				}
			}
		}, nil)
		return a, nil

	case ActionCompletedMsg:
		a.setStatus(msg.Message, !msg.Success)
		if msg.Success {
			cmds = append(cmds, a.toasts.PushSuccess(msg.Action, msg.Message))
		} else {
			cmds = append(cmds, a.toasts.PushError(msg.Action+" Failed", msg.Message))
		}
		// Refresh via informer invalidate
		if a.isInformerView(a.activeView) {
			if inf, ok := a.informers[a.activeView]; ok {
				inf.Invalidate()
			}
			cmds = append(cmds, a.dataPollImmediateCmd())
		} else if view, ok := a.views[a.activeView]; ok {
			cmds = append(cmds, view.Refresh())
		}

	case DialogClosedMsg:
		// Dispatch pending toast from dialog callback (safe: cmd is captured here)
		if a.pendingActionResult != nil {
			r := a.pendingActionResult
			a.pendingActionResult = nil
			if r.success {
				cmds = append(cmds, a.toasts.PushSuccess(r.title, r.errMsg))
			} else {
				cmds = append(cmds, a.toasts.PushError(r.title, r.errMsg))
			}
		}
		// Dialog was closed, refresh via informer
		if msg.Confirmed {
			if a.isInformerView(a.activeView) {
				if inf, ok := a.informers[a.activeView]; ok {
					inf.Invalidate()
				}
				cmds = append(cmds, a.dataPollImmediateCmd())
			} else if view, ok := a.views[a.activeView]; ok {
				cmds = append(cmds, view.Refresh())
			}
		}

	case OpenPaletteMsg:
		a.palette.Show()
		return a, nil

	case ClosePaletteMsg:
		// Palette was closed
		return a, nil

	case components.PickerSelectedMsg:
		switch msg.PickerID {
		case "namespace":
			a.namespace = msg.Item.ID
			a.propagateNamespace(msg.Item.ID)
			nsLabel := msg.Item.Label
			if msg.Item.ID == "" {
				a.header.SetNamespace("all")
				nsLabel = "all"
			} else {
				a.header.SetNamespace(msg.Item.ID)
			}
			a.invalidateHeader()
			a.setStatus("Switched to namespace: "+nsLabel, false)
			cmds = append(cmds, a.toasts.PushInfo("Namespace", "Switched to "+nsLabel))
			a.loading = true
			a.setInformersNamespace(msg.Item.ID)
			cmds = append(cmds, a.dataPollImmediateCmd())
		case "api-resource":
			if reg := a.client.APIResources(); reg != nil {
				if info, found := reg.Lookup(msg.Item.ID); found {
					return a, a.switchToGenericResource(info)
				}
			}
		}
		return a, tea.Batch(cmds...)

	case components.PickerCancelledMsg:
		// Picker was cancelled
		return a, nil

	case NamespacesLoadedMsg:
		if msg.Err != nil {
			a.setStatus("Failed to load namespaces: "+msg.Err.Error(), true)
		} else {
			items := []components.PickerItem{
				{ID: "", Label: "all", Desc: "All namespaces"},
			}
			for _, ns := range msg.Namespaces {
				items = append(items, components.PickerItem{ID: ns, Label: ns})
			}
			a.namespacePicker.SetItems(items)
		}
		return a, nil

	case views.PortForwardMsg:
		if msg.ResourceType == "services" {
			a.pfPicker.ShowForService(msg.Namespace, msg.ResourceName, msg.ServicePorts)
		} else {
			a.pfPicker.Show(msg.Namespace, msg.ResourceName, msg.Containers)
		}
		return a, nil

	case components.PortForwardPickerConfirmMsg:
		ns := msg.Namespace
		resourceType := msg.ResourceType
		resourceName := msg.ResourceName
		container := msg.Container
		localPort := msg.LocalPort
		remotePort := msg.RemotePort
		address := msg.Address
		return a, func() tea.Msg {
			session, err := a.pfManager.StartForward(ns, resourceType, resourceName, container, localPort, remotePort, address)
			if err != nil {
				return PortForwardErrorMsg{Namespace: ns, ResourceType: resourceType, ResourceName: resourceName, Err: err}
			}
			return PortForwardStartedMsg{
				ID:           session.ID,
				Namespace:    ns,
				ResourceType: resourceType,
				ResourceName: resourceName,
				LocalPort:    session.LocalPort,
				RemotePort:   remotePort,
				Address:      session.Address,
			}
		}

	case components.PortForwardPickerCancelMsg:
		// No-op
		return a, nil

	case views.ScalePickerMsg:
		a.scalePicker.Show(msg.Namespace, msg.Name, msg.Kind, msg.CurrentReplicas)
		return a, textinput.Blink

	case components.ScalePickerConfirmMsg:
		kind := msg.Kind
		ns := msg.Namespace
		name := msg.Name
		replicas := msg.Replicas
		return a, func() tea.Msg {
			err := a.client.Scale(context.Background(), kind, ns, name, replicas)
			if err != nil {
				return ActionCompletedMsg{Action: "Scaled", Success: false, Message: err.Error()}
			}
			return ActionCompletedMsg{Action: "Scaled", Success: true, Message: fmt.Sprintf("%s/%s scaled to %d replicas", ns, name, replicas)}
		}

	case components.ScalePickerCancelMsg:
		return a, nil

	case PortForwardStartedMsg:
		summary := fmt.Sprintf("%s:%d -> %s/%s:%d", msg.Address, msg.LocalPort, msg.Namespace, msg.ResourceName, msg.RemotePort)
		a.setStatus("Port forward: "+summary, false)
		cmds = append(cmds, a.toasts.PushSuccess("Port Forward", summary))
		// Refresh pods view to update PF column
		if a.isInformerView(ViewPods) {
			if inf, ok := a.informers[ViewPods]; ok {
				inf.Invalidate()
			}
			if a.activeView == ViewPods {
				cmds = append(cmds, a.dataPollImmediateCmd())
			}
		}
		// Refresh PF view if active
		if a.activeView == ViewPortForwards {
			a.pfView.Refresh()
		}

	case PortForwardStoppedMsg:
		summary := fmt.Sprintf("Stopped localhost:%d -> %s/%s:%d", msg.LocalPort, msg.Namespace, msg.ResourceName, msg.RemotePort)
		a.setStatus(summary, false)
		cmds = append(cmds, a.toasts.PushInfo("Port Forward", summary))
		// Refresh pods view to update PF column
		if a.isInformerView(ViewPods) {
			if inf, ok := a.informers[ViewPods]; ok {
				inf.Invalidate()
			}
			if a.activeView == ViewPods {
				cmds = append(cmds, a.dataPollImmediateCmd())
			}
		}
		// Refresh PF view if active
		if a.activeView == ViewPortForwards {
			a.pfView.Refresh()
		}

	case PortForwardErrorMsg:
		errMsg := fmt.Sprintf("Port forward failed for %s/%s: %s", msg.Namespace, msg.ResourceName, msg.Err.Error())
		a.setStatus(errMsg, true)
		cmds = append(cmds, a.toasts.PushError("Port Forward Failed", msg.Err.Error()))

	case views.ExecShellMsg:
		config := a.client.GetRestConfig()
		clientset := a.client.GetClientset()
		if config == nil || clientset == nil {
			a.setStatus("Shell not available: no cluster connection", true)
			cmds = append(cmds, a.toasts.PushError("Shell Failed", "No cluster connection"))
			return a, tea.Batch(cmds...)
		}
		shellCmd := k8s.NewShellExecCmd(clientset, config, msg.Namespace, msg.Pod, msg.Container)
		// Set execing so View() returns a blank screen. This prevents the
		// renderer's final flush (before ReleaseTerminal) from leaking TUI
		// content to the normal screen during the alt-screen transition.
		a.execing = true
		return a, tea.Exec(shellCmd, func(err error) tea.Msg {
			return ShellExitMsg{Err: err}
		})

	case ShellExitMsg:
		a.execing = false
		a.invalidateHeader()
		a.updateSizes()
		// Force fresh row rendering to prevent stale cached rows from pre-shell state
		if tbl := a.activeTable(); tbl != nil {
			tbl.InvalidateCache()
		}
		if msg.Err != nil && !errors.Is(msg.Err, io.EOF) && !errors.Is(msg.Err, context.Canceled) {
			a.setStatus("Shell error: "+msg.Err.Error(), true)
			cmds = append(cmds, a.toasts.PushError("Shell Failed", msg.Err.Error()))
		} else {
			a.setStatus("Shell session ended", false)
			cmds = append(cmds, a.toasts.PushInfo("Shell", "Session ended"))
		}
		return a, tea.Batch(cmds...)

	case EditExitMsg:
		a.execing = false
		a.invalidateHeader()
		a.updateSizes()
		if tbl := a.activeTable(); tbl != nil {
			tbl.InvalidateCache()
		}
		if msg.Err != nil {
			a.setStatus("Edit error: "+msg.Err.Error(), true)
			cmds = append(cmds, a.toasts.PushError("Edit Failed", msg.Err.Error()))
		} else if msg.Applied {
			a.setStatus("Resource updated successfully", false)
			cmds = append(cmds, a.toasts.PushSuccess("Edited", "Resource updated"))
			if a.isInformerView(a.activeView) {
				if inf, ok := a.informers[a.activeView]; ok {
					inf.Invalidate()
				}
				cmds = append(cmds, a.dataPollImmediateCmd())
			}
		} else {
			a.setStatus("No changes made", false)
		}
		return a, tea.Batch(cmds...)

	case views.GoBackMsg:
		return a, a.goBack()

	case views.NamespaceSelectedMsg:
		ns := msg.Namespace
		a.namespace = ns
		a.propagateNamespace(ns)
		a.invalidateHeader()
		nsLabel := ns
		if ns == "" {
			a.header.SetNamespace("all")
			a.setStatus("Switched to all namespaces", false)
			nsLabel = "all"
		} else {
			a.header.SetNamespace(ns)
			a.setStatus("Switched to namespace: "+ns, false)
		}
		// Go back to previous view; redirect cluster-scoped views to Pods
		dest := a.previousView
		if dest == ViewNodes || dest == ViewPVs {
			dest = ViewPods
		}
		a.activeView = dest
		a.loading = true
		var nsCmds []tea.Cmd
		nsCmds = append(nsCmds, a.toasts.PushInfo("Namespace", "Switched to "+nsLabel))
		a.setInformersNamespace(ns)
		nsCmds = append(nsCmds, a.dataPollImmediateCmd())
		return a, tea.Batch(nsCmds...)

	case views.ContextSelectedMsg:
		a.activeView = a.previousView
		return a, a.switchContext(msg.Context)

	case components.ToastExpiredMsg:
		a.toasts, _ = a.toasts.Update(msg)
		return a, nil

	case AutoRefreshTickMsg:
		if a.autoRefresh {
			if a.isInformerView(a.activeView) {
				if inf, ok := a.informers[a.activeView]; ok {
					inf.Invalidate()
				}
				cmds = append(cmds, a.dataPollImmediateCmd())
			} else if view, ok := a.views[a.activeView]; ok {
				cmds = append(cmds, view.Refresh())
			}
			cmds = append(cmds, a.autoRefreshCmd())
		}
		return a, tea.Batch(cmds...)

	case ExecuteCommandMsg:
		// Handle specific commands
		return a, a.executeCommand(msg.CommandID)

	case DataPollMsg:
		return a, a.handleDataPoll()

	case MetricsUpdatedMsg:
		if msg.Err == nil {
			a.header.SetCPUUsage(msg.CPU)
			a.header.SetMemUsage(msg.MEM)
			a.invalidateHeader()
			// Schedule next metrics fetch at normal interval
			cmds = append(cmds, a.metricsTickCmd())
		} else {
			// Backoff to 5 minutes on error
			cmds = append(cmds, tea.Tick(5*time.Minute, func(time.Time) tea.Msg {
				return MetricsTickMsg{}
			}))
		}

	case MetricsTickMsg:
		cmds = append(cmds, a.fetchMetrics())

	case PodMetricsTickMsg:
		cmds = append(cmds, a.podMetricsTickCmd()) // schedule next tick
		if a.podMetricsBackoff > 0 {
			a.podMetricsBackoff--
			break
		}
		if a.activeView == ViewPods {
			cmds = append(cmds, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				metrics, err := a.client.ListPodMetrics(ctx, a.namespace)
				return views.PodMetricsLoadedMsg{Metrics: metrics, Err: err}
			})
		}

	case views.PodMetricsLoadedMsg:
		if msg.Err != nil {
			a.podMetricsBackoff = 4 // skip 4 ticks (~60s before retry)
		} else {
			a.podMetricsBackoff = 0
		}
		// Let view handle the message via normal forwarding below

	case NodeMetricsTickMsg:
		cmds = append(cmds, a.nodeMetricsTickCmd()) // schedule next tick
		if a.nodeMetricsBackoff > 0 {
			a.nodeMetricsBackoff--
			break
		}
		if a.activeView == ViewNodes {
			cmds = append(cmds, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				metrics, err := a.client.ListNodeMetrics(ctx)
				return views.NodeMetricsLoadedMsg{Metrics: metrics, Err: err}
			})
		}

	case views.NodeMetricsLoadedMsg:
		if msg.Err != nil {
			a.nodeMetricsBackoff = 4 // skip 4 ticks (~60s before retry)
		} else {
			a.nodeMetricsBackoff = 0
		}
		// Let view handle the message via normal forwarding below
	}

	// Update active view
	if view, ok := a.views[a.activeView]; ok {
		var cmd tea.Cmd
		newView, cmd := view.Update(msg)
		a.views[a.activeView] = newView
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Clear loading flag when view has finished loading
		if a.loading {
			if lc, ok := newView.(LoadingChecker); ok {
				if !lc.IsLoading() {
					a.loading = false
				}
			}
		}

		// Consume scroll events from the table so they don't accumulate.
		// Scroll region API is not used (it marks lines as "ignored" which
		// prevents normal cursor-highlight updates within the viewport).
		// The performance optimizations (header cache, column width cache,
		// pre-computed padding, frame prePadded) handle flicker reduction.
		if tbl := a.activeTable(); tbl != nil {
			tbl.ConsumeBulkScroll()
			tbl.ConsumeScrollEvent()
		}
	}

	return a, tea.Batch(cmds...)
}

// pastTense converts an action verb to past tense for toast titles.
func pastTense(verb string) string {
	if strings.HasSuffix(verb, "e") {
		return verb + "d"
	}
	return verb + "ed"
}
