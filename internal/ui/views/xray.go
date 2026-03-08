package views

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/bijaya/kview/internal/graph"
	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// XrayLoadedMsg is sent when the xray graph is loaded
type XrayLoadedMsg struct {
	Graph *graph.Graph
	Err   error
}

// xrayMode distinguishes the two xray modes
type xrayMode int

const (
	xrayModeType     xrayMode = iota // Mode 1: show all resources of a kind
	xrayModeResource                 // Mode 2: show relationships for a specific resource
)

// xrayNode maps tree nodes to table rows
type xrayNode struct {
	uid         string
	kind        string
	name        string
	ns          string
	status      graph.NodeStatus
	depth       int
	prefix      string // tree drawing prefix
	hasChildren bool
	isExpanded  bool
	isFocused   bool   // Mode 2: the resource being inspected
	relation    string // relationship label for Mode 2 lateral items
	isNsHeader  bool   // true for namespace grouping headers
	childCount  int    // number of direct children
	info            string // extra info (e.g., "1/1" ready count)
	isSectionHeader bool   // Mode 2: relationship group header
	isOwnerHint     bool   // Mode 2: ↳ owner display-only line
}

// xrayColumns returns a single column for the k9s-style tree view
func xrayColumns() []components.Column {
	return []components.Column{
		{Title: "NAME", Width: 50, MinWidth: 30, Flexible: true},
	}
}

// XrayView displays resource relationships as an interactive tree
type XrayView struct {
	BaseView
	table     *components.Table
	client    k8s.Client
	graph     *graph.Graph
	mode      xrayMode
	rootKind  string          // Mode 1: resource kind to show
	focusName string          // Mode 2: resource name to focus on
	focusUID  string          // Mode 2: focused resource UID
	focusKind string          // Mode 2: kind filter for name resolution (from kind/name syntax)
	focusNS   string          // Mode 2: explicit namespace from ns/kind/name syntax
	expanded  map[string]bool // Expand/collapse state by UID
	flatNodes []*xrayNode     // Current flattened tree
	loading   bool
	err       error
	spinner   *components.Spinner
}

// NewXrayView creates a new xray view
func NewXrayView(client k8s.Client) *XrayView {
	v := &XrayView{
		table:    components.NewTable(xrayColumns()),
		client:   client,
		expanded: make(map[string]bool),
		spinner:  components.NewSpinner(),
	}
	v.spinner.SetMessage("Building xray graph...")
	v.table.SetEmptyState("", "No resources found",
		"No resources match the xray query",
		"Try a different resource kind or name")
	return v
}

// SetMode configures the xray view mode from a command argument.
func (v *XrayView) SetMode(arg string) error {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		v.mode = xrayModeType
		v.rootKind = "Deployment"
		v.focusName, v.focusUID, v.focusKind, v.focusNS = "", "", "", ""
		return nil
	}

	parts := strings.SplitN(arg, "/", 3)

	if len(parts) == 3 {
		// ns/kind/name syntax: "default/deploy/nginx"
		nsPart := parts[0]
		kindPart := parts[1]
		namePart := parts[2]
		kind := resolveKindAlias(kindPart)
		if kind == "" {
			kind = kindPart
		}
		v.mode = xrayModeResource
		v.focusNS = nsPart
		v.focusKind = kind
		v.focusName = namePart
		v.focusUID = ""
		v.rootKind = ""
		return nil
	}

	if len(parts) == 2 {
		// kind/name syntax: "deploy/nginx" (existing behavior)
		kindPart := parts[0]
		namePart := parts[1]
		kind := resolveKindAlias(kindPart)
		if kind == "" {
			kind = kindPart
		}
		v.mode = xrayModeResource
		v.focusNS = ""
		v.focusKind = kind
		v.focusName = namePart
		v.focusUID = ""
		v.rootKind = ""
		return nil
	}

	// Single token: kind alias or resource name
	if kind := resolveKindAlias(arg); kind != "" {
		v.mode = xrayModeType
		v.rootKind = kind
		v.focusName, v.focusUID, v.focusKind, v.focusNS = "", "", "", ""
		return nil
	}

	v.mode = xrayModeResource
	v.focusName = arg
	v.focusKind, v.focusUID, v.focusNS = "", "", ""
	v.rootKind = ""
	return nil
}

// SetModeForResource sets Mode 2 directly for a specific resource
func (v *XrayView) SetModeForResource(kind, name, ns, uid string) {
	v.mode = xrayModeResource
	v.focusName = name
	v.focusUID = uid
	v.focusKind = ""  // UID is known; no kind filter needed
	v.focusNS = ""
	v.rootKind = ""
}

// resolveKindAlias resolves command aliases to canonical Kind names
func resolveKindAlias(arg string) string {
	switch strings.ToLower(arg) {
	case "pod", "pods", "po":
		return "Pod"
	case "deploy", "deployment", "deployments":
		return "Deployment"
	case "svc", "service", "services":
		return "Service"
	case "rs", "replicaset", "replicasets":
		return "ReplicaSet"
	case "ds", "daemonset", "daemonsets":
		return "DaemonSet"
	case "sts", "statefulset", "statefulsets":
		return "StatefulSet"
	case "job", "jobs":
		return "Job"
	case "cj", "cronjob", "cronjobs":
		return "CronJob"
	case "cm", "configmap", "configmaps":
		return "ConfigMap"
	case "sec", "secret", "secrets":
		return "Secret"
	case "ing", "ingress", "ingresses":
		return "Ingress"
	case "pvc", "pvcs", "persistentvolumeclaim":
		return "PersistentVolumeClaim"
	case "pv", "pvs", "persistentvolume":
		return "PersistentVolume"
	case "hpa", "horizontalpodautoscaler", "horizontalpodautoscalers":
		return "HorizontalPodAutoscaler"
	case "node", "nodes", "no":
		return "Node"
	default:
		return ""
	}
}

// Init initializes the view
func (v *XrayView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *XrayView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case XrayLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.graph = msg.Graph
			if cmd := v.initExpanded(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			v.rebuildTable()
		}

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			return v, func() tea.Msg { return GoBackMsg{} }

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Enter):
			if node := v.selectedNode(); node != nil && node.hasChildren {
				v.expanded[node.uid] = !v.expanded[node.uid]
				v.rebuildTable()
			}

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if node := v.selectedNode(); node != nil && !node.isNsHeader && !node.isSectionHeader && !node.isOwnerHint {
				if node.kind == "Container" {
					if podName, podNS := v.getContainerParentPod(node.uid); podName != "" {
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "Pod",
								Resource:   "pods",
								Namespace:  podNS,
								Name:       podName,
							}
						}
					}
				} else {
					return v, v.openViewForNode(node, theme.ViewDescribe)
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if node := v.selectedNode(); node != nil && !node.isNsHeader && !node.isSectionHeader && !node.isOwnerHint && node.kind != "Container" {
				return v, v.openViewForNode(node, theme.ViewYAML)
			}

		case key.Matches(msg, theme.DefaultKeyMap().Logs):
			if node := v.selectedNode(); node != nil && !node.isSectionHeader && !node.isOwnerHint {
				if node.kind == "Pod" {
					return v, func() tea.Msg {
						return OpenViewMsg{
							TargetView: theme.ViewLogs,
							Kind:       "Pod",
							Resource:   "pods",
							Namespace:  node.ns,
							Name:       node.name,
							UID:        node.uid,
						}
					}
				} else if node.kind == "Container" {
					if podName, podNS := v.getContainerParentPod(node.uid); podName != "" {
						containerName := node.name
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewLogs,
								Kind:       "Pod",
								Resource:   "pods",
								Namespace:  podNS,
								Name:       podName,
								Container:  containerName,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Shell):
			if node := v.selectedNode(); node != nil && !node.isSectionHeader && !node.isOwnerHint {
				if node.kind == "Pod" {
					return v, func() tea.Msg {
						return ExecShellMsg{
							Namespace: node.ns,
							Pod:       node.name,
						}
					}
				} else if node.kind == "Container" {
					if podName, podNS := v.getContainerParentPod(node.uid); podName != "" {
						containerName := node.name
						return v, func() tea.Msg {
							return ExecShellMsg{
								Namespace: podNS,
								Pod:       podName,
								Container: containerName,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if node := v.selectedNode(); node != nil && !node.isNsHeader && !node.isSectionHeader && !node.isOwnerHint && node.kind != "Container" {
				resource := xrayKindToResource(node.kind)
				nodeName := node.name
				nodeNs := node.ns
				return v, func() tea.Msg {
					return ConfirmActionMsg{
						Title:   "Delete " + node.kind,
						Message: fmt.Sprintf("Delete %s/%s?", nodeNs, nodeName),
						Action: func() error {
							return v.client.Delete(context.Background(), resource, nodeNs, nodeName)
						},
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().PortForward):
			if node := v.selectedNode(); node != nil && !node.isSectionHeader && !node.isOwnerHint {
				if node.kind == "Pod" || node.kind == "Service" {
					resource := xrayKindToResource(node.kind)
					return v, func() tea.Msg {
						return PortForwardMsg{
							Namespace:    node.ns,
							ResourceType: resource,
							ResourceName: node.name,
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Restart):
			if node := v.selectedNode(); node != nil && !node.isSectionHeader && !node.isOwnerHint && node.kind == "Deployment" {
				nodeName := node.name
				nodeNs := node.ns
				return v, func() tea.Msg {
					return ConfirmActionMsg{
						Title:   "Restart Deployment",
						Message: fmt.Sprintf("Restart deployment %s/%s?", nodeNs, nodeName),
						Action: func() error {
							return v.client.Restart(context.Background(), "deployments", nodeNs, nodeName)
						},
					}
				}
			}
		}
	}

	// Update spinner
	if v.loading {
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update table
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// View renders the view
func (v *XrayView) View() string {
	if v.loading {
		return v.spinner.ViewCentered(v.width, v.height)
	}

	if v.err != nil {
		return theme.Styles.StatusError.Render("Error: " + v.err.Error())
	}

	return v.table.View()
}

// Name returns the view name
func (v *XrayView) Name() string {
	return "Xray"
}

// ShortHelp returns keybindings for help
func (v *XrayView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Describe,
		theme.DefaultKeyMap().Escape,
	}
}

// SetSize sets the view dimensions
func (v *XrayView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	v.table.SetSize(width, height)
}

// SetClient sets a new k8s client
func (v *XrayView) SetClient(client k8s.Client) {
	v.client = client
}

// IsLoading returns whether the view is currently loading data
func (v *XrayView) IsLoading() bool {
	return v.loading
}

// SelectedName returns the name of the currently selected resource
func (v *XrayView) SelectedName() string {
	if node := v.selectedNode(); node != nil {
		return node.name
	}
	return ""
}

// ResetSelection resets the table cursor to the top
func (v *XrayView) ResetSelection() {
	v.table.GotoTop()
}

// GetTable returns the table for TableAccess interface
func (v *XrayView) GetTable() *components.Table {
	return v.table
}

// RowCount returns the number of visible rows
func (v *XrayView) RowCount() int {
	return v.table.RowCount()
}

// Refresh loads the graph data
func (v *XrayView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			rg := graph.NewResourceGraph(v.client)
			err := rg.Build(context.Background(), v.namespace)
			if err != nil {
				return XrayLoadedMsg{Err: err}
			}
			return XrayLoadedMsg{Graph: rg.GetGraph()}
		},
	)
}

// initExpanded sets the initial expand/collapse state
func (v *XrayView) initExpanded() tea.Cmd {
	if v.graph == nil {
		return nil
	}

	v.expanded = make(map[string]bool)

	if v.mode == xrayModeResource && v.focusUID == "" && v.focusName != "" {
		matches := v.graph.FindNodeByName(v.focusName)
		if v.focusKind != "" {
			var filtered []*graph.Node
			for _, m := range matches {
				if matchKind(m.Kind, v.focusKind) {
					filtered = append(filtered, m)
				}
			}
			matches = filtered
		}

		// Namespace filtering
		effectiveNS := v.focusNS // explicit ns/kind/name takes priority
		if effectiveNS == "" {
			effectiveNS = v.namespace // current namespace context
		}

		if effectiveNS != "" {
			var nsFiltered []*graph.Node
			for _, m := range matches {
				if m.Namespace == effectiveNS {
					nsFiltered = append(nsFiltered, m)
				}
			}
			matches = nsFiltered
		} else if len(matches) > 1 {
			// All namespaces + ambiguous: collect namespace list
			nsSet := make(map[string]bool)
			for _, m := range matches {
				nsSet[m.Namespace] = true
			}
			if len(nsSet) > 1 {
				var nsList []string
				for ns := range nsSet {
					nsList = append(nsList, ns)
				}
				sort.Strings(nsList)

				query := v.focusName
				if v.focusKind != "" {
					query = strings.ToLower(v.focusKind) + "/" + v.focusName
				}
				v.table.SetEmptyState("", "Ambiguous resource",
					fmt.Sprintf("'%s' found in namespaces: %s", query, strings.Join(nsList, ", ")),
					"Use ns/kind/name (e.g. default/deploy/nginx)")
				return func() tea.Msg {
					return ShowToastMsg{
						Title:   "Xray",
						Message: fmt.Sprintf("'%s' found in multiple namespaces: %s", query, strings.Join(nsList, ", ")),
						IsError: true,
					}
				}
			}
			// len(nsSet) == 1 means all matches are in same namespace — no ambiguity, fall through
		}

		if len(matches) > 0 {
			v.focusUID = matches[0].UID
		} else {
			query := v.focusName
			if v.focusKind != "" {
				query = strings.ToLower(v.focusKind) + "/" + v.focusName
			}
			v.table.SetEmptyState("", "Resource not found",
				fmt.Sprintf("No resource matching '%s' was found", query),
				"Check the name and try again")
		}
	}

	if v.mode == xrayModeResource && v.focusUID != "" {
		focusNode := v.graph.GetNode(v.focusUID)
		if focusNode != nil {
			sections := resourceSections(focusNode.Kind)
			for _, sec := range sections {
				v.expanded[fmt.Sprintf("section/%s/%s", v.focusUID, sec.label)] = true
			}
		}
	}

	if v.mode == xrayModeType {
		q := graph.NewQuery(v.graph)
		kindNodes := q.FindByKind(v.rootKind)
		for _, root := range kindNodes {
			v.expanded[root.UID] = true
			for _, desc := range q.GetDescendants(root.UID) {
				v.expanded[desc.UID] = true
			}
		}
		namespaces := make(map[string]bool)
		for _, n := range kindNodes {
			namespaces[n.Namespace] = true
		}
		for ns := range namespaces {
			v.expanded["ns/"+ns] = true
		}
	}

	return nil
}

// rebuildTable flattens the tree and sets table rows
func (v *XrayView) rebuildTable() {
	if v.graph == nil {
		return
	}
	v.flatNodes = v.flattenTree()
	v.table.SetRows(v.nodesToRows(v.flatNodes))
}

// selectedNode returns the xrayNode for the currently selected table row
func (v *XrayView) selectedNode() *xrayNode {
	row := v.table.SelectedRow()
	if row == nil {
		return nil
	}
	for _, node := range v.flatNodes {
		if node.uid == row.ID {
			return node
		}
	}
	return nil
}

// getContainerParentPod finds the parent pod for a container node
func (v *XrayView) getContainerParentPod(containerUID string) (podName, podNS string) {
	if v.graph == nil {
		return "", ""
	}
	parents := v.graph.GetParents(containerUID)
	for _, p := range parents {
		if p.Kind == "Pod" {
			return p.Name, p.Namespace
		}
	}
	return "", ""
}

// openViewForNode creates a command to open describe/yaml for a specific node
func (v *XrayView) openViewForNode(node *xrayNode, targetView theme.ViewType) tea.Cmd {
	resource := xrayKindToResource(node.kind)
	return func() tea.Msg {
		return OpenViewMsg{
			TargetView: targetView,
			Kind:       node.kind,
			Resource:   resource,
			Namespace:  node.ns,
			Name:       node.name,
			UID:        node.uid,
		}
	}
}

// Ensure XrayView implements the required interfaces
var (
	_ View        = (*XrayView)(nil)
	_ TableAccess = (*XrayView)(nil)
)
