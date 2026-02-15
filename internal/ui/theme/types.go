package theme

// DialogClosedMsg is sent when a dialog is closed
type DialogClosedMsg struct {
	Confirmed bool
}

// ViewType represents different view types
type ViewType int

const (
	ViewPods ViewType = iota
	ViewDeployments
	ViewServices
	ViewConfigMaps
	ViewSecrets
	ViewIngresses
	ViewPVCs
	ViewStatefulSets
	ViewNodes
	ViewEvents
	ViewReplicaSets
	ViewDaemonSets
	ViewJobs
	ViewCronJobs
	ViewLogs
	ViewDescribe
	ViewTimeline
	ViewXray
	ViewDiagnosis
	ViewMultiCluster
	ViewYAML
	ViewNamespaceSelect
	ViewContainers
	ViewGenericResource
	ViewHPAs
	ViewPVs
	ViewRoleBindings
	ViewHealth
	ViewPulse
	ViewHelmReleases
	ViewHelmHistory
	ViewHelmValues
	ViewHelmManifest
	ViewSecretDecode
	ViewHelp
	ViewPortForwards
)

// SwitchViewMsg requests switching to a different view
type SwitchViewMsg struct {
	View ViewType
}
