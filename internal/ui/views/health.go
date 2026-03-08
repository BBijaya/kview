package views

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/bijaya/kview/internal/analyzer"
	"github.com/bijaya/kview/internal/analyzer/rules"
	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/theme"
)

// HealthDataMsg is sent when health dashboard data is loaded
type HealthDataMsg struct {
	ClusterMetrics *k8s.ClusterMetrics
	Pods           []k8s.PodInfo
	Nodes          []k8s.NodeInfo
	NodeMetrics    []k8s.NodeMetrics
	Deployments    []k8s.DeploymentInfo
	StatefulSets   []k8s.StatefulSetInfo
	DaemonSets     []k8s.DaemonSetInfo
	Jobs           []k8s.JobInfo
	PVCs           []k8s.PVCInfo
	Diagnoses      []analyzer.Diagnosis
	Events         []k8s.EventInfo
	Err            error
}

// kindToResource maps Kubernetes Kind to plural API resource name for drill-down
var kindToResource = map[string]string{
	"Pod":                   "pods",
	"Deployment":            "deployments",
	"ReplicaSet":            "replicasets",
	"DaemonSet":             "daemonsets",
	"StatefulSet":           "statefulsets",
	"Job":                   "jobs",
	"CronJob":               "cronjobs",
	"Service":               "services",
	"Node":                  "nodes",
	"ConfigMap":             "configmaps",
	"Secret":                "secrets",
	"Event":                 "events",
	"Ingress":               "ingresses",
	"PersistentVolumeClaim": "persistentvolumeclaims",
}

// Section indices for the nine navigable sections
const (
	sectionOverview    = 0
	sectionNodes       = 1
	sectionUnhealthy   = 2
	sectionFailedJobs  = 3
	sectionProblems    = 4
	sectionPendingPVCs = 5
	sectionRestarts    = 6
	sectionIssues      = 7
	sectionEvents      = 8
	sectionCount       = 9
)

var sectionNames = [sectionCount]string{"Overview", "Nodes", "Workloads", "Jobs", "Pods", "PVCs", "Restarts", "Issues", "Events"}

// healthNodeEntry holds a displayed node with precomputed metrics percentages
type healthNodeEntry struct {
	node   k8s.NodeInfo
	cpuPct int
	memPct int
}

// unhealthyWorkload holds a workload where ready < desired
type unhealthyWorkload struct {
	Kind      string // "Deployment", "StatefulSet", "DaemonSet"
	Resource  string // plural API resource name
	Name      string
	Namespace string
	UID       string
	Ready     int32
	Desired   int32
	Age       time.Duration
}

// HealthView displays a cluster health dashboard
type HealthView struct {
	BaseView
	viewport viewport.Model
	client   k8s.Client
	ruleSet  *rules.RuleSet

	// Data
	clusterMetrics *k8s.ClusterMetrics
	pods           []k8s.PodInfo
	nodes          []k8s.NodeInfo
	nodeMetrics    []k8s.NodeMetrics
	deployments    []k8s.DeploymentInfo
	statefulsets   []k8s.StatefulSetInfo
	daemonsets     []k8s.DaemonSetInfo
	jobs           []k8s.JobInfo
	pvcs           []k8s.PVCInfo
	diagnoses      []analyzer.Diagnosis
	events         []k8s.EventInfo

	// Precomputed display lists
	displayedNodes     []healthNodeEntry
	unhealthyWorkloads []unhealthyWorkload
	failedJobs         []k8s.JobInfo
	problemPods        []k8s.PodInfo
	pendingPVCs        []k8s.PVCInfo
	restartingPods     []k8s.PodInfo

	// Navigation
	sectionFocus      int
	itemMode          bool // true = navigating items inside focused section
	nodeCursor        int
	unhealthyCursor   int
	failedJobCursor   int
	problemCursor     int
	pendingPVCCursor  int
	restartCursor     int
	issueCursor       int
	eventCursor       int
	sectionLineOffsets [sectionCount]int // line number where each section header starts

	needsRefresh bool // set when namespace changes; triggers Refresh on next Update
	loading      bool
	err          error
}

// NewHealthView creates a new health dashboard view
func NewHealthView(client k8s.Client) *HealthView {
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))
	vp.Style = theme.Styles.Base

	return &HealthView{
		viewport: vp,
		client:   client,
		ruleSet:  rules.NewRuleSet(),
	}
}

// SetNamespace overrides BaseView to trigger a refresh when namespace changes.
func (v *HealthView) SetNamespace(ns string) {
	if ns != v.namespace {
		v.BaseView.SetNamespace(ns)
		v.needsRefresh = true
	}
}

// Init initializes the view
func (v *HealthView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *HealthView) Update(msg tea.Msg) (View, tea.Cmd) {
	if v.needsRefresh && !v.loading {
		v.needsRefresh = false
		return v, v.Refresh()
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case HealthDataMsg:
		v.loading = false
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.clusterMetrics = msg.ClusterMetrics
			v.pods = msg.Pods
			v.nodes = msg.Nodes
			v.nodeMetrics = msg.NodeMetrics
			v.deployments = msg.Deployments
			v.statefulsets = msg.StatefulSets
			v.daemonsets = msg.DaemonSets
			v.jobs = msg.Jobs
			v.pvcs = msg.PVCs
			v.diagnoses = msg.Diagnoses
			v.events = msg.Events
			v.sortDiagnoses()
			v.sortEvents()
			v.buildDisplayedNodes()
			v.buildUnhealthyWorkloads()
			v.buildFailedJobs()
			v.buildProblemPods()
			v.buildPendingPVCs()
			v.buildRestartingPods()
			v.updateContent()
		}

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			if v.itemMode {
				// Exit item mode back to section navigation
				v.itemMode = false
				v.updateContent()
				return v, nil
			}
			return v, func() tea.Msg { return GoBackMsg{} }

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Enter):
			if !v.itemMode {
				if v.sectionFocus == sectionOverview {
					return v, nil // nothing to enter
				}
				v.itemMode = true
				v.updateContent()
				return v, nil
			}
			// In item mode — drill down to describe
			return v, v.drillDown()

		case key.Matches(msg, theme.DefaultKeyMap().Up):
			if v.itemMode {
				v.moveCursor(-1)
			} else {
				// Move focus to previous section
				v.sectionFocus--
				if v.sectionFocus < 0 {
					v.sectionFocus = sectionCount - 1
				}
				v.ensureSectionVisible()
			}
			v.updateContent()
			return v, nil

		case key.Matches(msg, theme.DefaultKeyMap().Down):
			if v.itemMode {
				v.moveCursor(1)
			} else {
				// Move focus to next section
				v.sectionFocus++
				if v.sectionFocus >= sectionCount {
					v.sectionFocus = 0
				}
				v.ensureSectionVisible()
			}
			v.updateContent()
			return v, nil

		case msg.String() == "G":
			v.viewport.GotoBottom()

		case msg.String() == "g":
			v.viewport.GotoTop()
		}
	}

	// Update viewport for scrolling
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

func (v *HealthView) moveCursor(delta int) {
	switch v.sectionFocus {
	case sectionNodes:
		v.nodeCursor = clampCursor(v.nodeCursor+delta, len(v.displayedNodes))
	case sectionUnhealthy:
		v.unhealthyCursor = clampCursor(v.unhealthyCursor+delta, len(v.unhealthyWorkloads))
	case sectionFailedJobs:
		v.failedJobCursor = clampCursor(v.failedJobCursor+delta, len(v.failedJobs))
	case sectionProblems:
		v.problemCursor = clampCursor(v.problemCursor+delta, len(v.problemPods))
	case sectionPendingPVCs:
		v.pendingPVCCursor = clampCursor(v.pendingPVCCursor+delta, len(v.pendingPVCs))
	case sectionRestarts:
		v.restartCursor = clampCursor(v.restartCursor+delta, len(v.restartingPods))
	case sectionIssues:
		v.issueCursor = clampCursor(v.issueCursor+delta, len(v.diagnoses))
	case sectionEvents:
		v.eventCursor = clampCursor(v.eventCursor+delta, min(len(v.events), 15))
	}
}

func (v *HealthView) drillDown() tea.Cmd {
	switch v.sectionFocus {
	case sectionNodes:
		if v.nodeCursor < len(v.displayedNodes) {
			n := v.displayedNodes[v.nodeCursor].node
			return func() tea.Msg {
				return OpenViewMsg{
					TargetView: theme.ViewDescribe,
					Kind:       "Node",
					Resource:   "nodes",
					Name:       n.Name,
					UID:        n.UID,
				}
			}
		}

	case sectionUnhealthy:
		if v.unhealthyCursor < len(v.unhealthyWorkloads) {
			w := v.unhealthyWorkloads[v.unhealthyCursor]
			return func() tea.Msg {
				return OpenViewMsg{
					TargetView: theme.ViewDescribe,
					Kind:       w.Kind,
					Resource:   w.Resource,
					Namespace:  w.Namespace,
					Name:       w.Name,
					UID:        w.UID,
				}
			}
		}

	case sectionFailedJobs:
		if v.failedJobCursor < len(v.failedJobs) {
			j := v.failedJobs[v.failedJobCursor]
			return func() tea.Msg {
				return OpenViewMsg{
					TargetView: theme.ViewDescribe,
					Kind:       "Job",
					Resource:   "jobs",
					Namespace:  j.Namespace,
					Name:       j.Name,
					UID:        j.UID,
				}
			}
		}

	case sectionProblems:
		if v.problemCursor < len(v.problemPods) {
			p := v.problemPods[v.problemCursor]
			return func() tea.Msg {
				return OpenViewMsg{
					TargetView: theme.ViewDescribe,
					Kind:       "Pod",
					Resource:   "pods",
					Namespace:  p.Namespace,
					Name:       p.Name,
					UID:        p.UID,
				}
			}
		}

	case sectionPendingPVCs:
		if v.pendingPVCCursor < len(v.pendingPVCs) {
			pvc := v.pendingPVCs[v.pendingPVCCursor]
			return func() tea.Msg {
				return OpenViewMsg{
					TargetView: theme.ViewDescribe,
					Kind:       "PersistentVolumeClaim",
					Resource:   "persistentvolumeclaims",
					Namespace:  pvc.Namespace,
					Name:       pvc.Name,
					UID:        pvc.UID,
				}
			}
		}

	case sectionRestarts:
		if v.restartCursor < len(v.restartingPods) {
			p := v.restartingPods[v.restartCursor]
			return func() tea.Msg {
				return OpenViewMsg{
					TargetView: theme.ViewDescribe,
					Kind:       "Pod",
					Resource:   "pods",
					Namespace:  p.Namespace,
					Name:       p.Name,
					UID:        p.UID,
				}
			}
		}

	case sectionIssues:
		if v.issueCursor < len(v.diagnoses) {
			d := v.diagnoses[v.issueCursor]
			resource, ok := kindToResource[d.ResourceKind]
			if !ok {
				resource = strings.ToLower(d.ResourceKind) + "s"
			}
			return func() tea.Msg {
				return OpenViewMsg{
					TargetView: theme.ViewDescribe,
					Kind:       d.ResourceKind,
					Resource:   resource,
					Namespace:  d.Namespace,
					Name:       d.ResourceName,
					UID:        d.ResourceUID,
				}
			}
		}

	case sectionEvents:
		maxEvents := min(len(v.events), 15)
		if v.eventCursor < maxEvents {
			e := v.events[v.eventCursor]
			resource, ok := kindToResource[e.ObjectKind]
			if !ok {
				resource = strings.ToLower(e.ObjectKind) + "s"
			}
			return func() tea.Msg {
				return OpenViewMsg{
					TargetView: theme.ViewDescribe,
					Kind:       e.ObjectKind,
					Resource:   resource,
					Namespace:  e.Namespace,
					Name:       e.ObjectName,
					UID:        e.UID,
				}
			}
		}
	}

	return nil
}

// View renders the view
func (v *HealthView) View() string {
	if v.loading {
		return theme.Styles.StatusUnknown.Render("Loading cluster health data...")
	}

	if v.err != nil {
		return theme.Styles.StatusError.Render("Error: " + v.err.Error())
	}

	var b strings.Builder
	b.WriteString(v.viewport.View())
	b.WriteString("\n")
	b.WriteString(v.renderHelpLine())
	return b.String()
}

func (v *HealthView) renderHelpLine() string {
	w := v.viewport.Width()
	var line string
	if v.itemMode {
		name := sectionNames[v.sectionFocus]
		line = theme.Styles.Help.Render("↑↓ navigate " + name + " • Enter describe • Esc back to sections")
	} else if v.sectionFocus == sectionOverview {
		line = theme.Styles.Help.Render("↑↓ sections • Ctrl+R refresh • Esc back")
	} else {
		name := sectionNames[v.sectionFocus]
		line = theme.Styles.Help.Render("↑↓ sections • Enter→" + name + " • Ctrl+R refresh • Esc back")
	}
	return theme.PadToWidth(line, w, theme.ColorBackground)
}

// ensureSectionVisible scrolls the viewport so the focused section header is visible
func (v *HealthView) ensureSectionVisible() {
	v.viewport.SetYOffset(v.sectionLineOffsets[v.sectionFocus])
}

// --- View interface ---

func (v *HealthView) Name() string {
	return "Health"
}

func (v *HealthView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Refresh,
		theme.DefaultKeyMap().Escape,
	}
}

func (v *HealthView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	v.viewport.SetWidth(width)
	v.viewport.SetHeight(height - 1) // reserve 1 line for help footer
	if v.viewport.Height() < 1 {
		v.viewport.SetHeight(1)
	}
	if len(v.pods) > 0 || v.clusterMetrics != nil || len(v.nodes) > 0 {
		v.updateContent()
	}
}

func (v *HealthView) SetClient(client k8s.Client) {
	v.client = client
}

func (v *HealthView) IsLoading() bool {
	return v.loading
}

func (v *HealthView) ResetSelection() {
	v.nodeCursor = 0
	v.unhealthyCursor = 0
	v.failedJobCursor = 0
	v.problemCursor = 0
	v.pendingPVCCursor = 0
	v.restartCursor = 0
	v.issueCursor = 0
	v.eventCursor = 0
	v.sectionFocus = sectionOverview
	v.itemMode = false
}

func (v *HealthView) SelectedName() string {
	if !v.itemMode {
		return ""
	}
	switch v.sectionFocus {
	case sectionNodes:
		if v.nodeCursor < len(v.displayedNodes) {
			return v.displayedNodes[v.nodeCursor].node.Name
		}
	case sectionUnhealthy:
		if v.unhealthyCursor < len(v.unhealthyWorkloads) {
			return v.unhealthyWorkloads[v.unhealthyCursor].Name
		}
	case sectionFailedJobs:
		if v.failedJobCursor < len(v.failedJobs) {
			return v.failedJobs[v.failedJobCursor].Name
		}
	case sectionProblems:
		if v.problemCursor < len(v.problemPods) {
			return v.problemPods[v.problemCursor].Name
		}
	case sectionPendingPVCs:
		if v.pendingPVCCursor < len(v.pendingPVCs) {
			return v.pendingPVCs[v.pendingPVCCursor].Name
		}
	case sectionRestarts:
		if v.restartCursor < len(v.restartingPods) {
			return v.restartingPods[v.restartCursor].Name
		}
	case sectionIssues:
		if v.issueCursor < len(v.diagnoses) {
			return v.diagnoses[v.issueCursor].ResourceName
		}
	case sectionEvents:
		maxEvents := min(len(v.events), 15)
		if v.eventCursor < maxEvents {
			return v.events[v.eventCursor].ObjectName
		}
	}
	return ""
}
