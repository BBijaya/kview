package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// PodsLoadedMsg is sent when pods are loaded
type PodsLoadedMsg struct {
	Pods []k8s.PodInfo
	Err  error
}

// PodMetricsLoadedMsg is sent when pod metrics are loaded
type PodMetricsLoadedMsg struct {
	Metrics []k8s.PodMetrics
	Err     error
}

// ResourceSelectedMsg is sent when a resource is selected
type ResourceSelectedMsg struct {
	Kind      string
	Resource  string // plural API resource name (e.g., "pods", "deployments")
	Namespace string
	Name      string
	UID       string
}

// OpenViewMsg requests opening a view for a specific resource (atomic, no race)
type OpenViewMsg struct {
	TargetView theme.ViewType
	Kind       string
	Resource   string // plural API resource name (e.g., "pods", "deployments")
	Namespace  string
	Name       string
	UID        string
	Container  string // for container-specific log viewing
}

// ConfirmActionMsg requests confirmation for an action
type ConfirmActionMsg struct {
	Title   string
	Message string
	Action  func() error
}

// ExecShellMsg requests opening a shell session in a container
type ExecShellMsg struct {
	Namespace string
	Pod       string
	Container string
}

// PortForwardMsg requests opening the port forward picker for a resource
type PortForwardMsg struct {
	Namespace    string
	ResourceType string // "pods" or "services"
	ResourceName string
	Containers   []k8s.ContainerInfo
	ServicePorts []k8s.ServicePort
}

// ShowToastMsg requests showing a toast notification from a view
type ShowToastMsg struct {
	Title   string
	Message string
	IsError bool
}

// podColumns builds the column list for the pods table.
// When showNS is true, the NAMESPACE column is prepended.
func podColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 40, MinWidth: 20, Flexible: true},
		components.Column{Title: "PF", Width: 4, Align: lipgloss.Center},
		components.Column{Title: "READY", Width: 7, Align: lipgloss.Center},
		components.Column{Title: "STATUS", Width: 18},
		components.Column{Title: "RESTARTS", Width: 10, Align: lipgloss.Right, IsNumeric: true},
		components.Column{Title: "CPU", Width: 7, Align: lipgloss.Right},
		components.Column{Title: "%CPU/R", Width: 7, Align: lipgloss.Right},
		components.Column{Title: "%CPU/L", Width: 7, Align: lipgloss.Right},
		components.Column{Title: "MEM", Width: 7, Align: lipgloss.Right},
		components.Column{Title: "%MEM/R", Width: 7, Align: lipgloss.Right},
		components.Column{Title: "%MEM/L", Width: 7, Align: lipgloss.Right},
		components.Column{Title: "IP", Width: 16},
		components.Column{Title: "NODE", Width: 20, MinWidth: 10, Flexible: true},
		components.Column{Title: "AGE", Width: 6, Align: lipgloss.Right},
	)
	return cols
}

// PodsView displays a list of pods
type PodsView struct {
	BaseView
	table     *components.Table
	filter    *components.SearchInput
	client    k8s.Client
	pods      []k8s.PodInfo
	metrics   map[string]*k8s.PodMetrics // key: "namespace/name"
	pfManager *k8s.PortForwardManager    // port forward manager (optional)
	showNS    bool                       // whether NAMESPACE column is shown
	loading   bool
	err       error
	spinner   *components.Spinner

	// Drill-down owner filter (set when navigating from Deployments)
	ownerKind string // e.g. "Deployment"
	ownerName string // e.g. "nginx"

	// Drill-down node filter (set when navigating from Nodes)
	nodeFilter string // e.g. "minikube"

	// Drill-down label selector filter (set when navigating from Services)
	labelSelector map[string]string
}

// NewPodsView creates a new pods view
func NewPodsView(client k8s.Client) *PodsView {
	// Default: all namespaces (showNS = true)
	v := &PodsView{
		table:   components.NewTable(podColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		metrics: make(map[string]*k8s.PodMetrics),
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading pods...")

	// Set contextual empty state
	v.table.SetEmptyState("📦", "No pods found",
		"No pods match your current namespace and filter",
		"Try changing namespace (n) or clearing filter (/)")

	return v
}

// Init initializes the view
func (v *PodsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *PodsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case PodsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.pods = msg.Pods
			v.updateTable()
		}

	case PodMetricsLoadedMsg:
		if msg.Err == nil {
			v.metrics = make(map[string]*k8s.PodMetrics)
			for i := range msg.Metrics {
				m := &msg.Metrics[i]
				v.metrics[m.Namespace+"/"+m.Name] = m
			}
			v.updateTable()
		}

	case components.FilterChangedMsg:
		v.table.SetFilter(msg.Value)

	case components.FilterClosedMsg:
		v.filter.Hide()

	case tea.KeyPressMsg:
		// Handle filter input first if visible
		if v.filter.IsVisible() {
			var cmd tea.Cmd
			v.filter, cmd = v.filter.Update(msg)
			return v, cmd
		}

		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Filter):
			v.filter.Show()
			return v, nil

		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			if v.HasOwnerFilter() || v.HasNodeFilter() || v.HasLabelSelector() {
				return v, func() tea.Msg { return GoBackMsg{} }
			}

		case key.Matches(msg, theme.DefaultKeyMap().Enter):
			if pod := v.SelectedPod(); pod != nil {
				p := *pod
				return v, func() tea.Msg {
					return DrillDownContainersMsg{Pod: p}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Logs):
			if row := v.table.SelectedRow(); row != nil {
				for _, pod := range v.pods {
					if pod.UID == row.ID {
						pod := pod
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewLogs,
								Kind: "Pod", Resource: "pods", Namespace: pod.Namespace,
								Name: pod.Name, UID: pod.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, pod := range v.pods {
					if pod.UID == row.ID {
						pod := pod
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind: "Pod", Resource: "pods", Namespace: pod.Namespace,
								Name: pod.Name, UID: pod.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, pod := range v.pods {
					if pod.UID == row.ID {
						pod := pod
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind: "Pod", Resource: "pods", Namespace: pod.Namespace,
								Name: pod.Name, UID: pod.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, pod := range v.pods {
					if pod.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete Pod",
								Message: fmt.Sprintf("Delete pod %s/%s?", pod.Namespace, pod.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "pods", pod.Namespace, pod.Name)
								},
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Shell):
			if pod := v.SelectedPod(); pod != nil {
				container := ""
				if len(pod.Containers) > 0 {
					container = pod.Containers[0].Name
				}
				p := *pod
				return v, func() tea.Msg {
					return ExecShellMsg{
						Namespace: p.Namespace,
						Pod:       p.Name,
						Container: container,
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().PortForward):
			if pod := v.SelectedPod(); pod != nil {
				p := *pod
				return v, func() tea.Msg {
					return PortForwardMsg{
						Namespace:    p.Namespace,
						ResourceType: "pods",
						ResourceName: p.Name,
						Containers:   p.Containers,
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
func (v *PodsView) View() string {
	if v.loading {
		return v.spinner.ViewCentered(v.width, v.height)
	}

	if v.err != nil {
		return theme.Styles.StatusError.Render("Error: " + v.err.Error())
	}

	content := v.table.View()

	// Add filter input if visible
	if v.filter.IsVisible() {
		content = v.filter.View() + "\n" + content
	}

	return content
}

// Name returns the view name
func (v *PodsView) Name() string {
	return "Pods"
}

// ShortHelp returns keybindings for help
func (v *PodsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Logs,
		theme.DefaultKeyMap().Describe,
	}
}

// SetSize sets the view dimensions
func (v *PodsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	// Account for filter if visible
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *PodsView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(podColumns(newShowNS))
	}
}

// ResetSelection resets the table cursor to the top
func (v *PodsView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *PodsView) IsLoading() bool {
	return v.loading
}

// SelectedName returns the name of the currently selected resource
func (v *PodsView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1) // NS=0, NAME=1
	}
	return v.table.SelectedValue(0) // NAME=0
}

// Refresh refreshes the pod list
func (v *PodsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			pods, err := v.client.ListPods(context.Background(), v.namespace)
			return PodsLoadedMsg{Pods: pods, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *PodsView) SetClient(client k8s.Client) {
	v.client = client
}

// SetPortForwardManager sets the port forward manager for PF column display
func (v *PodsView) SetPortForwardManager(pfm *k8s.PortForwardManager) {
	v.pfManager = pfm
}

// SetOwnerFilter sets a drill-down owner filter (e.g. from Deployments view)
func (v *PodsView) SetOwnerFilter(kind, name string) {
	v.ownerKind = kind
	v.ownerName = name
	v.updateTable()
}

// ClearOwnerFilter removes the drill-down owner filter
func (v *PodsView) ClearOwnerFilter() {
	v.ownerKind = ""
	v.ownerName = ""
	v.updateTable()
}

// HasOwnerFilter returns whether an owner filter is active
func (v *PodsView) HasOwnerFilter() bool {
	return v.ownerName != ""
}

// SetNodeFilter sets a drill-down node filter (from Nodes view)
func (v *PodsView) SetNodeFilter(nodeName string) {
	v.nodeFilter = nodeName
	v.updateTable()
}

// ClearNodeFilter removes the drill-down node filter
func (v *PodsView) ClearNodeFilter() {
	v.nodeFilter = ""
	v.updateTable()
}

// HasNodeFilter returns whether a node filter is active
func (v *PodsView) HasNodeFilter() bool {
	return v.nodeFilter != ""
}

// SetLabelSelector sets a drill-down label selector filter (from Services view)
func (v *PodsView) SetLabelSelector(selector map[string]string) {
	v.labelSelector = selector
	v.updateTable()
}

// ClearLabelSelector removes the drill-down label selector filter
func (v *PodsView) ClearLabelSelector() {
	v.labelSelector = nil
	v.updateTable()
}

// HasLabelSelector returns whether a label selector filter is active
func (v *PodsView) HasLabelSelector() bool {
	return len(v.labelSelector) > 0
}

// SelectedPod returns the currently selected pod
func (v *PodsView) SelectedPod() *k8s.PodInfo {
	if row := v.table.SelectedRow(); row != nil {
		for _, pod := range v.pods {
			if pod.UID == row.ID {
				return &pod
			}
		}
	}
	return nil
}

// RowCount returns the number of visible rows
func (v *PodsView) RowCount() int {
	return v.table.RowCount()
}

func (v *PodsView) filteredPods() []k8s.PodInfo {
	if v.ownerName == "" && v.nodeFilter == "" && len(v.labelSelector) == 0 {
		return v.pods
	}
	var filtered []k8s.PodInfo
	for _, pod := range v.pods {
		if v.nodeFilter != "" && pod.NodeName != v.nodeFilter {
			continue
		}
		if v.ownerName != "" && !v.matchesOwnerFilter(pod) {
			continue
		}
		if len(v.labelSelector) > 0 && !v.matchesLabelSelector(pod) {
			continue
		}
		filtered = append(filtered, pod)
	}
	return filtered
}

// matchesLabelSelector checks whether a pod's labels match the selector.
func (v *PodsView) matchesLabelSelector(pod k8s.PodInfo) bool {
	for k, val := range v.labelSelector {
		if pod.Labels[k] != val {
			return false
		}
	}
	return true
}

// matchesOwnerFilter checks whether a pod belongs to the filtered owner.
// For Deployments: Deployment "foo" -> ReplicaSet "foo-<hash>" -> Pod.
// So we match pods whose OwnerRef Kind=="ReplicaSet" and Name starts with "ownerName-".
func (v *PodsView) matchesOwnerFilter(pod k8s.PodInfo) bool {
	prefix := v.ownerName + "-"
	switch v.ownerKind {
	case "Deployment":
		for _, ref := range pod.OwnerRefs {
			if ref.Kind == "ReplicaSet" && strings.HasPrefix(ref.Name, prefix) {
				return true
			}
		}
	default:
		// Direct owner match for other kinds
		for _, ref := range pod.OwnerRefs {
			if ref.Kind == v.ownerKind && ref.Name == v.ownerName {
				return true
			}
		}
	}
	return false
}

func (v *PodsView) updateTable() {
	pods := v.filteredPods()
	rows := make([]components.Row, len(pods))
	for i, pod := range pods {
		status := getPodStatus(pod)

		// Look up metrics for this pod
		key := pod.Namespace + "/" + pod.Name
		m := v.metrics[key]

		cpuStr, cpuReqPct, cpuLimPct := "n/a", "n/a", "n/a"
		memStr, memReqPct, memLimPct := "n/a", "n/a", "n/a"
		if m != nil {
			cpuStr = k8s.FormatCPU(m.CPUUsage)
			memStr = k8s.FormatMemory(m.MemUsage)
			if pod.CPURequest > 0 {
				cpuReqPct = fmt.Sprintf("%d%%", m.CPUUsage*100/pod.CPURequest)
			}
			if pod.CPULimit > 0 {
				cpuLimPct = fmt.Sprintf("%d%%", m.CPUUsage*100/pod.CPULimit)
			}
			if pod.MemRequest > 0 {
				memReqPct = fmt.Sprintf("%d%%", m.MemUsage*100/pod.MemRequest)
			}
			if pod.MemLimit > 0 {
				memLimPct = fmt.Sprintf("%d%%", m.MemUsage*100/pod.MemLimit)
			}
		}

		// Port forward indicator
		pfIndicator := "-"
		if v.pfManager != nil {
			if pfs := v.pfManager.ActiveForPod(pod.Namespace, pod.Name); len(pfs) > 0 {
				if len(pfs) == 1 {
					pfIndicator = fmt.Sprintf("%d", pfs[0].LocalPort)
				} else {
					pfIndicator = fmt.Sprintf("%d(%d)", pfs[0].LocalPort, len(pfs))
				}
			}
		}

		values := []string{}
		if v.showNS {
			values = append(values, pod.Namespace)
		}
		values = append(values,
			pod.Name,
			pfIndicator,
			pod.Ready,
			status,
			fmt.Sprintf("%d", pod.Restarts),
			cpuStr, cpuReqPct, cpuLimPct,
			memStr, memReqPct, memLimPct,
			pod.IP, pod.NodeName,
			formatAge(pod.Age),
		)

		rows[i] = components.Row{
			ID:     pod.UID,
			Values: values,
			Status: status,
			Labels: pod.Labels,
		}
	}
	v.table.SetRows(rows)
}

func getPodStatus(pod k8s.PodInfo) string {
	// Check container states for more specific status
	for _, c := range pod.Containers {
		if c.StateReason != "" {
			return c.StateReason
		}
	}
	for _, c := range pod.InitContainers {
		if c.State == "Waiting" || c.State == "Running" {
			return "Init:" + c.StateReason
		}
	}
	return pod.Phase
}

func formatAge(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// GetTable returns the underlying table component.
func (v *PodsView) GetTable() *components.Table {
	return v.table
}

func (v *PodsView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
