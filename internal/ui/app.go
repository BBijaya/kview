package ui

import (
	"context"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/store"
	"github.com/bijaya/kview/internal/ui/commands"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/views"
)

// RowCounter is an optional interface for views that can report their row count
type RowCounter interface {
	RowCount() int
}

// LoadingChecker is an optional interface for views that can report loading state
type LoadingChecker interface {
	IsLoading() bool
}

// InputMode represents the current input mode
type InputMode int

const (
	ModeNormal InputMode = iota
	ModeCommand
	ModeFilter
)

// actionResult holds the result of a dialog callback action for safe toast dispatch
type actionResult struct {
	title   string
	errMsg  string
	success bool
}

// drillDownEntry records a view in the navigation stack with its context.
type drillDownEntry struct {
	View         ViewType
	DrillContext string // context that was active when this view was current
}

// App is the main application model
type App struct {
	// Kubernetes
	client    k8s.Client
	context   string
	namespace string

	// UI state
	width        int
	height       int
	activeView   ViewType
	previousView ViewType
	viewStack        []drillDownEntry    // drill-down navigation stack
	savedFilters     map[ViewType]string // saved filter text for drill-down persistence
	drillDownSavedNS string              // saved namespace for node drill-down restore
	drillContext     string              // parent context shown in body label (e.g., "nodename", "ns/podname")
	views            map[ViewType]views.View
	inputMode    InputMode

	// Special views (not in resource list)
	describeView    *views.DescribeView
	logsView        *views.LogsView
	yamlView        *views.YAMLView
	nsSelectView    *views.NamespaceSelectView
	xrayView        *views.XrayView
	diagnosisView   *views.DiagnosisView
	timelineView    *views.TimelineView
	containersView  *views.ContainersView
	healthView      *views.HealthView
	pulseView       *views.PulseView
	helmHistoryView    *views.HelmHistoryView
	helmValuesView     *views.HelmContentView
	helmManifestView   *views.HelmContentView
	secretDecodeView   *views.SecretDecodeView
	helpView           *views.HelpView

	// Persistence
	store           store.Store

	// Components
	header        *components.Header
	statusBar     *components.StatusBar
	tabs          *components.Tabs
	tabBar        *components.TabBar
	categoryTabs  *components.CategoryTabs
	palette       *commands.Palette
	dialog        *components.Dialog
	registry      *commands.Registry
	frame         *components.Frame
	commandInput  *components.CommandInput
	searchInput   *components.SearchInput

	// New components (Phase 1)
	namespacePicker *components.Picker
	contextPicker   *components.Picker
	detailsPanel    *components.DetailsPanel
	toasts          *components.ToastStack

	// Generic resource view
	genericView         *views.GenericResourceView
	genericResourceKind string // current resource plural name
	apiResourcePicker   *components.Picker

	// Port forwarding
	pfManager *k8s.PortForwardManager
	pfPicker  *components.PortForwardPicker
	pfView    *views.PortForwardsView

	// Scaling
	scalePicker *components.ScalePicker

	// State
	statusMessage    string
	statusIsError    bool
	statusExpiresAt  time.Time
	showDetailsPanel bool
	autoRefresh      bool
	refreshInterval  time.Duration
	quitting         bool
	execing          bool
	loading          bool

	// Footer center message (shown in footer bar, auto-expires)
	footerMessage   string
	footerExpiresAt time.Time

	// Informer state (k9s-style background cache)
	informers   map[ViewType]*k8s.ResourceInformer
	lastSeenGen map[ViewType]uint64

	// Pending action result (for safe toast dispatch from dialog callbacks)
	pendingActionResult *actionResult

	// Pod metrics backoff (skip N ticks after failure)
	podMetricsBackoff int

	// Node metrics backoff (skip N ticks after failure)
	nodeMetricsBackoff int

	// Connection state
	connectionError  string
	isDisconnected   bool

	// Startup warning (e.g., unknown theme name)
	startupWarning string

	// Selected resource (for cross-view communication)
	selectedResource *ResourceSelectedMsg

	// Header cache (avoids re-rendering identical header during scrolls)
	cachedHeader       string
	headerDirtyFlag    uint64
	lastRenderedHeader uint64

}

// NewApp creates a new application instance
func NewApp(client k8s.Client) *App {
	registry := commands.NewRegistry()

	// Create tabs (expanded for new views)
	tabList := []components.Tab{
		{ID: "pods", Title: "Pods"},
		{ID: "deployments", Title: "Deployments"},
		{ID: "services", Title: "Services"},
		{ID: "configmaps", Title: "ConfigMaps"},
		{ID: "secrets", Title: "Secrets"},
		{ID: "ingresses", Title: "Ingresses"},
		{ID: "pvcs", Title: "PVCs"},
		{ID: "statefulsets", Title: "StatefulSets"},
		{ID: "nodes", Title: "Nodes"},
		{ID: "events", Title: "Events"},
		{ID: "replicasets", Title: "ReplicaSets"},
		{ID: "daemonsets", Title: "DaemonSets"},
		{ID: "jobs", Title: "Jobs"},
		{ID: "cronjobs", Title: "CronJobs"},
	}

	// Get context info
	contextName, _ := k8s.GetCurrentContext()

	app := &App{
		client:          client,
		context:         contextName,
		namespace:       "", // All namespaces by default
		activeView:      ViewPods,
		views:           make(map[ViewType]views.View),
		savedFilters:    make(map[ViewType]string),
		inputMode:       ModeNormal,
		header:          components.NewHeader(),
		statusBar:       components.NewStatusBar(),
		tabs:            components.NewTabs(tabList),
		tabBar:          components.NewTabBar([]string{"Pods", "Deploy", "Svc", "CM", "Sec", "Ing", "PVC", "STS", "Nodes", "Events", "RS", "DS", "Jobs", "CJ"}),
		categoryTabs:    components.NewCategoryTabs(),
		palette:         commands.NewPalette(registry),
		dialog:          components.NewDialog(),
		registry:        registry,
		frame:           components.NewFrame(),
		commandInput:    components.NewCommandInput(),
		searchInput:     components.NewSearchInput(),
		namespacePicker: components.NewPicker("namespace", "Select Namespace"),
		contextPicker:   components.NewPicker("context", "Select Context"),
		detailsPanel:    components.NewDetailsPanel(client),
		toasts:          components.NewToastStack(),
		refreshInterval: 5 * time.Second,
	}

	// Initialize generic resource view and API resource picker
	app.genericView = views.NewGenericResourceView(client)
	app.views[ViewGenericResource] = app.genericView
	app.apiResourcePicker = components.NewPicker("api-resource", "Select API Resource")

	// Initialize port forward manager, picker, and view
	app.pfManager = k8s.NewPortForwardManager(client.GetRestConfig(), client.GetClientset())
	app.pfPicker = components.NewPortForwardPicker()
	app.pfView = views.NewPortForwardsView(app.pfManager)
	app.views[ViewPortForwards] = app.pfView

	// Initialize scale picker
	app.scalePicker = components.NewScalePicker()

	// Initialize special views
	app.describeView = views.NewDescribeView(client)
	app.logsView = views.NewLogsView(client)
	app.yamlView = views.NewYAMLView(client)
	app.nsSelectView = views.NewNamespaceSelectView(client)
	app.containersView = views.NewContainersView(client)

	// Initialize views
	app.views[ViewDescribe] = app.describeView
	app.views[ViewLogs] = app.logsView
	app.views[ViewYAML] = app.yamlView
	app.views[ViewNamespaceSelect] = app.nsSelectView
	app.views[ViewContainers] = app.containersView
	podsView := views.NewPodsView(client)
	podsView.SetPortForwardManager(app.pfManager)
	app.views[ViewPods] = podsView
	app.views[ViewDeployments] = views.NewDeploymentsView(client)
	app.views[ViewServices] = views.NewServicesView(client)
	app.views[ViewConfigMaps] = views.NewConfigMapsView(client)
	app.views[ViewSecrets] = views.NewSecretsView(client)
	app.views[ViewIngresses] = views.NewIngressesView(client)
	app.views[ViewPVCs] = views.NewPVCsView(client)
	app.views[ViewStatefulSets] = views.NewStatefulSetsView(client)
	app.views[ViewNodes] = views.NewNodesView(client)
	app.views[ViewEvents] = views.NewEventsView(client)
	app.views[ViewReplicaSets] = views.NewReplicaSetsView(client)
	app.views[ViewDaemonSets] = views.NewDaemonSetsView(client)
	app.views[ViewJobs] = views.NewJobsView(client)
	app.views[ViewCronJobs] = views.NewCronJobsView(client)
	app.views[ViewHPAs] = views.NewHPAsView(client)
	app.views[ViewPVs] = views.NewPVsView(client)
	app.views[ViewRoleBindings] = views.NewRoleBindingsView(client)
	app.views[ViewHelmReleases] = views.NewHelmReleasesView(client)
	app.helmHistoryView = views.NewHelmHistoryView(client)
	app.views[ViewHelmHistory] = app.helmHistoryView
	app.helmValuesView = views.NewHelmContentView(client, views.HelmContentValues)
	app.views[ViewHelmValues] = app.helmValuesView
	app.helmManifestView = views.NewHelmContentView(client, views.HelmContentManifest)
	app.views[ViewHelmManifest] = app.helmManifestView
	app.secretDecodeView = views.NewSecretDecodeView(client)
	app.views[ViewSecretDecode] = app.secretDecodeView
	app.helpView = views.NewHelpView()
	app.views[ViewHelp] = app.helpView

	// Initialize advanced views
	app.xrayView = views.NewXrayView(client)
	app.diagnosisView = views.NewDiagnosisView(client)
	app.views[ViewXray] = app.xrayView
	app.views[ViewDiagnosis] = app.diagnosisView
	app.healthView = views.NewHealthView(client)
	app.views[ViewHealth] = app.healthView
	app.pulseView = views.NewPulseView(client)
	app.views[ViewPulse] = app.pulseView

	// Initialize SQLite store for timeline (optional, fails gracefully)
	dataDir := filepath.Join(os.Getenv("HOME"), ".kview")
	if s, err := store.NewSQLiteStore(filepath.Join(dataDir, "kview.db")); err == nil {
		app.store = s
		app.timelineView = views.NewTimelineView(s)
		app.views[ViewTimeline] = app.timelineView
	}

	// Initialize informers
	app.initInformers()

	// Set header info
	app.header.SetContext(contextName)
	app.header.SetNamespace("all")
	app.header.SetServerVersion(client.ServerVersion())
	app.header.SetViewName("Pods")

	// Set user and cluster info for k9s-style header
	app.header.SetUser(k8s.GetCurrentUser())
	app.header.SetClusterName(k8s.GetCurrentClusterName())
	app.header.SetCPUUsage("n/a") // Placeholder - metrics API integration is future work
	app.header.SetMemUsage("n/a") // Placeholder - metrics API integration is future work
	app.header.SetAccessMode(client.CheckWriteAccess(context.Background()))

	return app
}

// SetStartupWarning appends a warning message to be shown as a toast on Init().
func (a *App) SetStartupWarning(msg string) {
	if a.startupWarning != "" {
		a.startupWarning += "; " + msg
	} else {
		a.startupWarning = msg
	}
}

// NewAppWithError creates a new application instance with a connection error
func NewAppWithError(client k8s.Client, errorMsg string) *App {
	app := NewApp(client)
	app.connectionError = errorMsg
	app.isDisconnected = true
	app.header.SetContext("disconnected")
	app.header.SetServerVersion("N/A")
	return app
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	var cmds []tea.Cmd

	a.loading = true

	// For informer-backed views, start the informer and poll immediately
	if a.isInformerView(a.activeView) {
		a.ensureInformer(a.activeView)
		cmds = append(cmds, a.dataPollImmediateCmd())
		cmds = append(cmds, a.dataPollTickCmd())
	} else {
		// Non-informer views (describe, logs, etc.) use Init() as before
		if view, ok := a.views[a.activeView]; ok {
			cmds = append(cmds, view.Init())
		}
	}

	// Start tick for status message expiration
	cmds = append(cmds, a.tickCmd())

	// Start initial metrics fetch
	cmds = append(cmds, a.fetchMetrics())

	// Start pod metrics tick (15s interval)
	cmds = append(cmds, a.podMetricsTickCmd())

	// Start node metrics tick (15s interval)
	cmds = append(cmds, a.nodeMetricsTickCmd())

	// Show startup warning toast if set (e.g., unknown theme name)
	if a.startupWarning != "" {
		cmds = append(cmds, a.toasts.PushWarning("Theme", a.startupWarning))
	}

	return tea.Batch(cmds...)
}
