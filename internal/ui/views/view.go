package views

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// ClientSetter is an optional interface for views that can receive a new client
type ClientSetter interface {
	SetClient(client k8s.Client)
}

// TableAccess is an optional interface for views backed by a scrollable table.
// The app uses this to access the table for high-performance scroll regions.
type TableAccess interface {
	GetTable() *components.Table
}

// FilterChecker is an optional interface for views that can report whether a filter is active.
type FilterChecker interface {
	IsFilterVisible() bool
}

// DrillDownViewMsg requests a drill-down navigation to a view type.
// Unlike SwitchViewMsg (which clears the navigation stack), this pushes
// the current view onto the stack so Escape/GoBackMsg returns to it.
type DrillDownViewMsg struct {
	View theme.ViewType
}

// DrillDownDeploymentMsg requests drill-down from deployment to filtered pods
type DrillDownDeploymentMsg struct {
	DeploymentName string
	Namespace      string
}

// DrillDownNodeMsg requests drill-down from node to its pods (all namespaces)
type DrillDownNodeMsg struct {
	NodeName string
}

// DrillDownContainersMsg requests drill-down from pod to containers
type DrillDownContainersMsg struct {
	Pod k8s.PodInfo
}

// View defines the interface for all views
type View interface {
	// Init initializes the view
	Init() tea.Cmd

	// Update handles messages and updates the view state
	Update(msg tea.Msg) (View, tea.Cmd)

	// View renders the view
	View() string

	// Name returns the display name of the view
	Name() string

	// ShortHelp returns the key bindings for the short help view
	ShortHelp() []key.Binding

	// SetSize sets the view dimensions
	SetSize(width, height int)

	// SetNamespace sets the namespace filter for the view
	SetNamespace(ns string)

	// Refresh triggers a refresh of the view data
	Refresh() tea.Cmd

	// ResetSelection resets the selection cursor to the top
	ResetSelection()

	// SelectedName returns the name of the currently selected resource
	SelectedName() string
}

// BaseView provides common functionality for views
type BaseView struct {
	width     int
	height    int
	namespace string
	focused   bool
}

// SetSize sets the view dimensions
func (v *BaseView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Width returns the view width
func (v *BaseView) Width() int {
	return v.width
}

// Height returns the view height
func (v *BaseView) Height() int {
	return v.height
}

// SetNamespace sets the namespace filter
func (v *BaseView) SetNamespace(ns string) {
	v.namespace = ns
}

// Namespace returns the current namespace filter
func (v *BaseView) Namespace() string {
	return v.namespace
}

// ResetSelection is a no-op default; table-based views override this
func (v *BaseView) ResetSelection() {}

// SelectedName returns empty string by default; table-based views override this
func (v *BaseView) SelectedName() string { return "" }

// Focus focuses the view
func (v *BaseView) Focus() {
	v.focused = true
}

// Blur unfocuses the view
func (v *BaseView) Blur() {
	v.focused = false
}

// IsFocused returns whether the view is focused
func (v *BaseView) IsFocused() bool {
	return v.focused
}
