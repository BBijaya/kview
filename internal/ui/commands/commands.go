package commands

// Command represents an executable command
type Command struct {
	ID          string
	Name        string
	Description string
	Category    string
	Shortcut    string
	Action      func() interface{} // Returns a tea.Msg
}

// ViewType mirrors the ui.ViewType for command actions
type ViewType int

const (
	ViewPods ViewType = iota
	ViewDeployments
	ViewServices
	ViewLogs
	ViewDescribe
	ViewTimeline
	ViewXray
	ViewDiagnosis
	ViewMultiCluster
)

// SwitchViewMsg is returned when a command wants to switch views
type SwitchViewMsg struct {
	View ViewType
}

// RefreshMsg is returned when a command wants to refresh
type RefreshMsg struct{}

// ExecuteCommandMsg is returned when a command needs further processing
type ExecuteCommandMsg struct {
	CommandID string
	Args      map[string]interface{}
}

// SwitchNamespaceMsg is returned when switching namespace
type SwitchNamespaceMsg struct {
	Namespace string
}

// DefaultCommands returns the built-in commands
func DefaultCommands() []Command {
	return []Command{
		// Navigation
		{
			ID:          "view.pods",
			Name:        "View Pods",
			Description: "Switch to pods view",
			Category:    "Navigation",
			Shortcut:    "1",
			Action: func() interface{} {
				return SwitchViewMsg{View: ViewPods}
			},
		},
		{
			ID:          "view.deployments",
			Name:        "View Deployments",
			Description: "Switch to deployments view",
			Category:    "Navigation",
			Shortcut:    "2",
			Action: func() interface{} {
				return SwitchViewMsg{View: ViewDeployments}
			},
		},
		{
			ID:          "view.services",
			Name:        "View Services",
			Description: "Switch to services view",
			Category:    "Navigation",
			Shortcut:    "3",
			Action: func() interface{} {
				return SwitchViewMsg{View: ViewServices}
			},
		},
		{
			ID:          "view.logs",
			Name:        "View Logs",
			Description: "View logs for selected pod",
			Category:    "Navigation",
			Shortcut:    "L",
			Action: func() interface{} {
				return SwitchViewMsg{View: ViewLogs}
			},
		},
		{
			ID:          "view.describe",
			Name:        "Describe Resource",
			Description: "Show details for selected resource",
			Category:    "Navigation",
			Shortcut:    "d",
			Action: func() interface{} {
				return SwitchViewMsg{View: ViewDescribe}
			},
		},
		{
			ID:          "view.timeline",
			Name:        "View Timeline",
			Description: "Show event timeline",
			Category:    "Navigation",
			Shortcut:    "t",
			Action: func() interface{} {
				return SwitchViewMsg{View: ViewTimeline}
			},
		},
		{
			ID:          "view.xray",
			Name:        "View Xray",
			Description: "Show resource relationships (xray tree)",
			Category:    "Navigation",
			Shortcut:    "X",
			Action: func() interface{} {
				return SwitchViewMsg{View: ViewXray}
			},
		},
		{
			ID:          "view.diagnosis",
			Name:        "View Diagnosis",
			Description: "Show problem diagnosis",
			Category:    "Navigation",
			Shortcut:    "D",
			Action: func() interface{} {
				return SwitchViewMsg{View: ViewDiagnosis}
			},
		},

		// Actions
		{
			ID:          "action.refresh",
			Name:        "Refresh",
			Description: "Refresh current view",
			Category:    "Actions",
			Shortcut:    "ctrl+r",
			Action: func() interface{} {
				return RefreshMsg{}
			},
		},
		{
			ID:          "action.delete",
			Name:        "Delete Resource",
			Description: "Delete the selected resource",
			Category:    "Actions",
			Shortcut:    "ctrl+d",
			Action: func() interface{} {
				return ExecuteCommandMsg{CommandID: "action.delete"}
			},
		},
		{
			ID:          "action.restart",
			Name:        "Restart Deployment",
			Description: "Restart the selected deployment",
			Category:    "Actions",
			Shortcut:    "r",
			Action: func() interface{} {
				return ExecuteCommandMsg{CommandID: "action.restart"}
			},
		},
		{
			ID:          "action.scale",
			Name:        "Scale Deployment",
			Description: "Scale the selected deployment",
			Category:    "Actions",
			Shortcut:    "s",
			Action: func() interface{} {
				return ExecuteCommandMsg{CommandID: "action.scale"}
			},
		},

		// Context/Namespace
		{
			ID:          "switch.namespace",
			Name:        "Switch Namespace",
			Description: "Change the current namespace",
			Category:    "Context",
			Shortcut:    "n",
			Action: func() interface{} {
				return ExecuteCommandMsg{CommandID: "switch.namespace"}
			},
		},
		{
			ID:          "switch.context",
			Name:        "Switch Context",
			Description: "Change the Kubernetes context",
			Category:    "Context",
			Shortcut:    "c",
			Action: func() interface{} {
				return ExecuteCommandMsg{CommandID: "switch.context"}
			},
		},
		{
			ID:          "switch.allnamespaces",
			Name:        "All Namespaces",
			Description: "Show resources from all namespaces",
			Category:    "Context",
			Action: func() interface{} {
				return SwitchNamespaceMsg{Namespace: ""}
			},
		},
	}
}
