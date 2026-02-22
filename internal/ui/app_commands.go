package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bijaya/kview/internal/ui/theme"
	"github.com/bijaya/kview/internal/ui/views"
)

// handleCommand handles vim-style commands from the command input
func (a *App) handleCommand(cmd string, args []string) tea.Cmd {
	switch cmd {
	case "q", "quit":
		a.quitting = true
		a.pfManager.StopAll()
		a.stopAllInformers()
		if a.store != nil {
			a.store.Close()
		}
		return tea.Quit

	case "pods", "pod", "po":
		return a.switchView(ViewPods)

	case "deployments", "deployment", "deploy":
		return a.switchView(ViewDeployments)

	case "services", "service", "svc":
		return a.switchView(ViewServices)

	case "configmaps", "configmap", "cm":
		return a.switchView(ViewConfigMaps)

	case "secrets", "secret", "sec":
		return a.switchView(ViewSecrets)

	case "ingresses", "ingress", "ing":
		return a.switchView(ViewIngresses)

	case "pvcs", "pvc", "persistentvolumeclaim", "persistentvolumeclaims":
		return a.switchView(ViewPVCs)

	case "statefulsets", "statefulset", "sts":
		return a.switchView(ViewStatefulSets)

	case "nodes", "node", "no":
		return a.switchView(ViewNodes)

	case "events", "event", "ev":
		return a.switchView(ViewEvents)

	case "replicasets", "replicaset", "rs":
		return a.switchView(ViewReplicaSets)

	case "daemonsets", "daemonset", "ds":
		return a.switchView(ViewDaemonSets)

	case "jobs", "job":
		return a.switchView(ViewJobs)

	case "cronjobs", "cronjob", "cj":
		return a.switchView(ViewCronJobs)

	case "hpa", "hpas", "horizontalpodautoscaler", "horizontalpodautoscalers":
		return a.switchView(ViewHPAs)

	case "pv", "pvs", "persistentvolume", "persistentvolumes":
		return a.switchView(ViewPVs)

	case "rolebindings", "rolebinding", "rb":
		return a.switchView(ViewRoleBindings)

	case "helm", "helmreleases", "helmrelease", "releases", "release", "rel", "hr":
		return a.switchView(ViewHelmReleases)

	case "ns", "namespace":
		if len(args) > 0 {
			namespace := args[0]
			if namespace == "all" || namespace == "-" {
				a.namespace = ""
				a.propagateNamespace("")
				a.header.SetNamespace("all")
			} else {
				a.namespace = namespace
				a.propagateNamespace(namespace)
				a.header.SetNamespace(namespace)
			}
			a.loading = true
			a.setInformersNamespace(a.namespace)
			return a.dataPollImmediateCmd()
		}
		return a.loadNamespaces()

	case "ctx", "context":
		if len(args) > 0 {
			return a.switchContext(args[0])
		}
		a.setStatus("Usage: :ctx <context>", false)
		return nil

	case "refresh", "r":
		if a.isInformerView(a.activeView) {
			if inf, ok := a.informers[a.activeView]; ok {
				inf.Invalidate()
			}
			return a.dataPollImmediateCmd()
		}
		if view, ok := a.views[a.activeView]; ok {
			return view.Refresh()
		}
		return nil

	case "delete", "del":
		if a.selectedResource != nil {
			return func() tea.Msg {
				return ConfirmActionMsg{
					Title:   "Delete " + a.selectedResource.Kind,
					Message: fmt.Sprintf("Delete %s/%s?", a.selectedResource.Namespace, a.selectedResource.Name),
					Action: func() error {
						return a.client.Delete(context.Background(),
							a.selectedResource.Resource,
							a.selectedResource.Namespace,
							a.selectedResource.Name)
					},
				}
			}
		}
		a.setStatus("No resource selected", true)
		return nil

	case "describe", "desc":
		if a.selectedResource != nil {
			a.describeView.SetResource(
				a.selectedResource.Resource,
				a.selectedResource.Namespace,
				a.selectedResource.Name,
			)
			return a.switchView(ViewDescribe)
		}
		a.setStatus("No resource selected", true)
		return nil

	case "logs", "log":
		if a.selectedResource != nil && a.selectedResource.Kind == "Pod" {
			a.logsView.SetPod(a.selectedResource.Namespace, a.selectedResource.Name, "")
			return a.switchView(ViewLogs)
		}
		a.setStatus("Select a pod to view logs", true)
		return nil

	case "shell", "sh", "exec":
		switch a.activeView {
		case ViewPods:
			if pv, ok := a.views[ViewPods].(*views.PodsView); ok {
				if pod := pv.SelectedPod(); pod != nil {
					container := ""
					if len(args) > 0 {
						container = args[0]
					} else if len(pod.Containers) > 0 {
						container = pod.Containers[0].Name
					}
					return func() tea.Msg {
						return views.ExecShellMsg{
							Namespace: pod.Namespace,
							Pod:       pod.Name,
							Container: container,
						}
					}
				}
				a.setStatus("No pod selected", true)
			}
			return nil
		case ViewContainers:
			pod := a.containersView.Pod()
			containerName := a.containersView.SelectedName()
			if containerName != "" {
				return func() tea.Msg {
					return views.ExecShellMsg{
						Namespace: pod.Namespace,
						Pod:       pod.Name,
						Container: containerName,
					}
				}
			}
			a.setStatus("No container selected", true)
			return nil
		default:
			a.setStatus("Shell only available in Pods/Containers view", true)
			return nil
		}

	case "yaml":
		if a.selectedResource != nil {
			a.yamlView.SetResource(
				a.selectedResource.Resource,
				a.selectedResource.Namespace,
				a.selectedResource.Name,
			)
			return a.switchView(ViewYAML)
		}
		a.setStatus("No resource selected", true)
		return nil

	case "edit":
		if a.selectedResource != nil {
			return a.editResource(a.selectedResource.Resource, a.selectedResource.Namespace, a.selectedResource.Name)
		}
		a.setStatus("No resource selected", true)
		return nil

	case "scale":
		if len(args) >= 2 {
			name := args[0]
			replicas, err := strconv.Atoi(args[1])
			if err != nil {
				a.setStatus("Invalid replica count: "+args[1], true)
				return nil
			}
			ns := a.namespace
			replicaStr := args[1]
			return func() tea.Msg {
				err := a.client.Scale(context.Background(), "deployments", ns, name, replicas)
				if err != nil {
					return ActionCompletedMsg{Action: "Scale", Success: false, Message: "Scale failed: " + err.Error()}
				}
				return ActionCompletedMsg{Action: "Scale", Success: true, Message: fmt.Sprintf("Scaled %s to %s replicas", name, replicaStr)}
			}
		}
		a.setStatus("Usage: :scale <name> <replicas>", false)
		return nil

	case "xray":
		if len(args) == 0 {
			return a.toasts.PushInfo("Xray", "Usage: :xray <kind|name|kind/name|ns/kind/name>  (e.g. :xray deploy, :xray svc/nginx, :xray default/deploy/nginx)")
		}
		if err := a.xrayView.SetMode(strings.Join(args, " ")); err != nil {
			return a.toasts.PushError("Xray", err.Error())
		}
		return a.switchView(ViewXray)

	case "graph": // backward compat alias
		return a.toasts.PushInfo("Xray", "Usage: :xray <kind|name|kind/name|ns/kind/name>  (e.g. :xray deploy, :xray svc/nginx, :xray default/deploy/nginx)")

	case "timeline", "tl":
		return a.switchView(ViewTimeline)

	case "diagnosis", "diag":
		return a.switchView(ViewDiagnosis)

	case "health":
		return a.switchView(ViewHealth)

	case "pulse":
		return a.switchView(ViewPulse)

	case "pf", "portforwards", "portforward":
		a.pfView.Refresh()
		return a.drillDown(ViewPortForwards)

	case "pf-stop":
		if len(args) == 0 {
			a.setStatus("Usage: :pf-stop <id|all>", false)
			return nil
		}
		if args[0] == "all" {
			a.pfManager.StopAll()
			a.setStatus("All port forwards stopped", false)
			cmds := []tea.Cmd{a.toasts.PushInfo("Port Forward", "All port forwards stopped")}
			if a.isInformerView(ViewPods) {
				if inf, ok := a.informers[ViewPods]; ok {
					inf.Invalidate()
				}
				if a.activeView == ViewPods {
					cmds = append(cmds, a.dataPollImmediateCmd())
				}
			}
			if a.activeView == ViewPortForwards {
				a.pfView.Refresh()
			}
			return tea.Batch(cmds...)
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			a.setStatus("Invalid port forward ID: "+args[0], true)
			return nil
		}
		if err := a.pfManager.StopForward(id); err != nil {
			a.setStatus("Error: "+err.Error(), true)
			return nil
		}
		a.setStatus(fmt.Sprintf("Port forward %d stopped", id), false)
		pfCmds := []tea.Cmd{a.toasts.PushInfo("Port Forward", fmt.Sprintf("Session %d stopped", id))}
		if a.isInformerView(ViewPods) {
			if inf, ok := a.informers[ViewPods]; ok {
				inf.Invalidate()
			}
			if a.activeView == ViewPods {
				pfCmds = append(pfCmds, a.dataPollImmediateCmd())
			}
		}
		if a.activeView == ViewPortForwards {
			a.pfView.Refresh()
		}
		return tea.Batch(pfCmds...)

	case "help", "h":
		if a.activeView == ViewHelp {
			return a.goBack()
		}
		return a.drillDown(ViewHelp)

	case "themes":
		names := theme.ThemeNames
		lines := make([]string, len(names))
		for i, name := range names {
			lines[i] = "  " + name
		}
		return a.toasts.PushInfo("Themes ("+strconv.Itoa(len(names))+")", strings.Join(lines, "\n"))

	case "api-resources", "ar":
		return a.showAPIResourcePicker()

	default:
		// Try discovery before reporting unknown command
		if reg := a.client.APIResources(); reg != nil {
			if info, found := reg.Lookup(cmd); found {
				return a.switchToGenericResource(info)
			}
		}
		a.setStatus("Unknown command: "+cmd, true)
		return nil
	}
}

// executeCommand handles commands from the command palette/registry
func (a *App) executeCommand(commandID string) tea.Cmd {
	switch commandID {
	case "action.delete":
		if a.selectedResource != nil {
			ns := a.selectedResource.Namespace
			name := a.selectedResource.Name
			kind := a.selectedResource.Kind
			resource := a.selectedResource.Resource
			a.dialog.ShowConfirm("Delete "+kind,
				fmt.Sprintf("Delete %s %s/%s?", kind, ns, name),
				func() {
					err := a.client.Delete(context.Background(),
						resource, ns, name)
					if err != nil {
						a.setStatus("Delete failed: "+err.Error(), true)
						a.pendingActionResult = &actionResult{title: "Delete Failed", errMsg: err.Error(), success: false}
					} else {
						a.setStatus("Deleted: "+name, false)
						a.pendingActionResult = &actionResult{title: "Deleted", errMsg: kind + "/" + name, success: true}
					}
				}, nil)
		}
		return nil
	case "action.restart":
		if a.selectedResource != nil && a.selectedResource.Kind == "Deployment" {
			ns := a.selectedResource.Namespace
			name := a.selectedResource.Name
			a.dialog.ShowConfirm("Restart Deployment",
				fmt.Sprintf("Restart deployment %s/%s?", ns, name),
				func() {
					err := a.client.Restart(context.Background(), "deployments", ns, name)
					if err != nil {
						a.setStatus("Restart failed: "+err.Error(), true)
						a.pendingActionResult = &actionResult{title: "Restart Failed", errMsg: err.Error(), success: false}
					} else {
						a.setStatus("Restarted: "+name, false)
						a.pendingActionResult = &actionResult{title: "Restarted", errMsg: "Deployment/" + name, success: true}
					}
				}, nil)
		}
		return nil
	case "action.scale":
		a.setStatus("Usage: :scale <name> <replicas>", false)
		return nil
	case "switch.namespace":
		return a.loadNamespaces()
	case "switch.context":
		return a.loadContexts()
	}
	return nil
}
