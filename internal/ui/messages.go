package ui

import (
	"github.com/bijaya/kview/internal/ui/theme"
)

// Message types for Bubble Tea

// NamespacesLoadedMsg is sent when namespaces are loaded
type NamespacesLoadedMsg struct {
	Namespaces []string
	Err        error
}

// RefreshMsg triggers a refresh of the current view
type RefreshMsg struct{}

// ErrorMsg represents an error message
type ErrorMsg struct {
	Err error
}

// StatusMsg represents a status message to display
type StatusMsg struct {
	Message string
	IsError bool
}

// ViewType re-exports theme.ViewType
type ViewType = theme.ViewType

// View type constants re-exported from theme
const (
	ViewPods         = theme.ViewPods
	ViewDeployments  = theme.ViewDeployments
	ViewServices     = theme.ViewServices
	ViewConfigMaps   = theme.ViewConfigMaps
	ViewSecrets      = theme.ViewSecrets
	ViewIngresses    = theme.ViewIngresses
	ViewPVCs         = theme.ViewPVCs
	ViewStatefulSets = theme.ViewStatefulSets
	ViewNodes        = theme.ViewNodes
	ViewEvents       = theme.ViewEvents
	ViewReplicaSets  = theme.ViewReplicaSets
	ViewDaemonSets   = theme.ViewDaemonSets
	ViewJobs         = theme.ViewJobs
	ViewCronJobs     = theme.ViewCronJobs
	ViewLogs         = theme.ViewLogs
	ViewDescribe     = theme.ViewDescribe
	ViewTimeline     = theme.ViewTimeline
	ViewXray        = theme.ViewXray
	ViewDiagnosis    = theme.ViewDiagnosis
	ViewMultiCluster = theme.ViewMultiCluster
	ViewYAML             = theme.ViewYAML
	ViewNamespaceSelect  = theme.ViewNamespaceSelect
	ViewContainers       = theme.ViewContainers
	ViewGenericResource  = theme.ViewGenericResource
	ViewHPAs             = theme.ViewHPAs
	ViewPVs              = theme.ViewPVs
	ViewRoleBindings     = theme.ViewRoleBindings
	ViewHealth           = theme.ViewHealth
	ViewPulse            = theme.ViewPulse
	ViewHelmReleases     = theme.ViewHelmReleases
	ViewHelmHistory      = theme.ViewHelmHistory
	ViewHelmValues       = theme.ViewHelmValues
	ViewHelmManifest     = theme.ViewHelmManifest
	ViewSecretDecode     = theme.ViewSecretDecode
	ViewHelp             = theme.ViewHelp
	ViewPortForwards     = theme.ViewPortForwards
	ViewContextSelect    = theme.ViewContextSelect
)


// ResourceSelectedMsg is sent when a resource is selected
type ResourceSelectedMsg struct {
	Kind      string
	Resource  string // plural API resource name (e.g., "pods", "deployments")
	Namespace string
	Name      string
	UID       string
}

// ActionCompletedMsg is sent when an action completes
type ActionCompletedMsg struct {
	Action  string
	Success bool
	Message string
}

// OpenPaletteMsg opens the command palette
type OpenPaletteMsg struct{}

// ClosePaletteMsg closes the command palette
type ClosePaletteMsg struct{}

// ExecuteCommandMsg requests execution of a command
type ExecuteCommandMsg struct {
	CommandID string
	Args      map[string]interface{}
}

// ConfirmActionMsg requests confirmation for an action
type ConfirmActionMsg struct {
	Title   string
	Message string
	Action  func() error
}

// DialogClosedMsg is re-exported from theme
type DialogClosedMsg = theme.DialogClosedMsg

// ViewName returns the name of a view type
func ViewName(v ViewType) string {
	switch v {
	case ViewPods:
		return "Pods"
	case ViewDeployments:
		return "Deployments"
	case ViewServices:
		return "Services"
	case ViewConfigMaps:
		return "ConfigMaps"
	case ViewSecrets:
		return "Secrets"
	case ViewIngresses:
		return "Ingresses"
	case ViewPVCs:
		return "PVCs"
	case ViewStatefulSets:
		return "StatefulSets"
	case ViewNodes:
		return "Nodes"
	case ViewEvents:
		return "Events"
	case ViewReplicaSets:
		return "ReplicaSets"
	case ViewDaemonSets:
		return "DaemonSets"
	case ViewJobs:
		return "Jobs"
	case ViewCronJobs:
		return "CronJobs"
	case ViewLogs:
		return "Logs"
	case ViewDescribe:
		return "Describe"
	case ViewTimeline:
		return "Timeline"
	case ViewXray:
		return "Xray"
	case ViewDiagnosis:
		return "Diagnosis"
	case ViewMultiCluster:
		return "Multi-Cluster"
	case ViewYAML:
		return "YAML"
	case ViewNamespaceSelect:
		return "Namespaces"
	case ViewContainers:
		return "Containers"
	case ViewGenericResource:
		return "Resources"
	case ViewHPAs:
		return "HPAs"
	case ViewPVs:
		return "PVs"
	case ViewRoleBindings:
		return "RoleBindings"
	case ViewHealth:
		return "Health"
	case ViewPulse:
		return "Pulse"
	case ViewHelmReleases:
		return "Helm Releases"
	case ViewHelmHistory:
		return "Helm History"
	case ViewHelmValues:
		return "Helm Values"
	case ViewHelmManifest:
		return "Helm Manifest"
	case ViewSecretDecode:
		return "Secret Decoded"
	case ViewHelp:
		return "Help"
	case ViewPortForwards:
		return "Port Forwards"
	case ViewContextSelect:
		return "Contexts"
	default:
		return "Unknown"
	}
}

// CommandCompletionsMsg carries dynamic completion data for the command input
type CommandCompletionsMsg struct {
	Namespaces     []string
	Contexts       []string
	PortForwardIDs []string
}

// ContextSwitchedMsg is sent when context switch completes
type ContextSwitchedMsg struct {
	Context string
}

// DataPollMsg triggers a poll of informer snapshots
type DataPollMsg struct{}

// MetricsUpdatedMsg is sent when cluster metrics are updated
type MetricsUpdatedMsg struct {
	CPU string
	MEM string
	Err error
}

// MetricsTickMsg triggers a periodic metrics refresh
type MetricsTickMsg struct{}

// PodMetricsTickMsg triggers a periodic pod metrics refresh
type PodMetricsTickMsg struct{}

// NodeMetricsTickMsg triggers a periodic node metrics refresh
type NodeMetricsTickMsg struct{}

// TickMsg is sent periodically for updates
type TickMsg struct{}

// WindowSizeMsg is sent when the window is resized
type WindowSizeMsg struct {
	Width  int
	Height int
}

// ShellExitMsg is sent when a shell session ends
type ShellExitMsg struct {
	Err error
}

// EditExitMsg is sent when an edit session ends
type EditExitMsg struct {
	Err     error
	Applied bool // true if changes were applied to the cluster
}

// PortForwardStartedMsg is sent when a port forward starts successfully
type PortForwardStartedMsg struct {
	ID                                int
	Namespace, ResourceType, ResourceName string
	LocalPort, RemotePort             int
	Address                           string
}

// PortForwardStoppedMsg is sent when a port forward is stopped
type PortForwardStoppedMsg struct {
	ID                                int
	Namespace, ResourceType, ResourceName string
	LocalPort, RemotePort             int
}

// PortForwardErrorMsg is sent when a port forward fails
type PortForwardErrorMsg struct {
	Namespace, ResourceType, ResourceName string
	Err                                   error
}
