package theme

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the application
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding

	// Selection
	Enter  key.Binding
	Select key.Binding

	// View switching
	Pods         key.Binding
	Deployments  key.Binding
	Services     key.Binding
	ConfigMaps   key.Binding
	Secrets      key.Binding
	Ingresses    key.Binding
	PVCs         key.Binding
	StatefulSets key.Binding
	Logs         key.Binding
	Describe     key.Binding
	Timeline     key.Binding
	Xray         key.Binding
	Diagnosis    key.Binding
	NextTab      key.Binding
	PrevTab      key.Binding

	// Panels
	DetailsPanel key.Binding
	AutoRefresh  key.Binding

	// Actions
	Delete      key.Binding
	Restart     key.Binding
	Scale       key.Binding
	Shell       key.Binding
	PortForward key.Binding
	Edit        key.Binding
	Refresh     key.Binding
	Filter    key.Binding
	Search    key.Binding
	Namespace key.Binding
	Context   key.Binding
	Palette   key.Binding
	CopyName  key.Binding
	YAML      key.Binding
	Command   key.Binding

	// Modal/Dialog
	Confirm key.Binding
	Cancel  key.Binding
	Escape  key.Binding

	// Table horizontal scroll
	ScrollLeft  key.Binding
	ScrollRight key.Binding

	// Table sorting
	SortToggle  key.Binding
	SortColPrev key.Binding
	SortColNext key.Binding

	// Secrets
	DecodeSecret key.Binding

	// Helm
	HelmValues   key.Binding
	HelmManifest key.Binding

	// Log viewer
	LogSearch     key.Binding
	LogSearchNext key.Binding
	LogSearchPrev key.Binding
	LogSave       key.Binding
	LogTimestamp  key.Binding
	LogPrevious   key.Binding
	LogTimeRange  key.Binding
	LogWrap       key.Binding

	// General
	Help key.Binding
	Quit key.Binding
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("ctrl+left"),
			key.WithHelp("ctrl+←", "prev category"),
		),
		Right: key.NewBinding(
			key.WithKeys("ctrl+right"),
			key.WithHelp("ctrl+→", "next category"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "go to top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "go to bottom"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Select: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle select"),
		),
		Pods: key.NewBinding(
			key.WithKeys(),
			key.WithHelp("", "pods"),
		),
		Deployments: key.NewBinding(
			key.WithKeys(),
			key.WithHelp("[n]", "by number key"),
		),
		Services: key.NewBinding(
			key.WithKeys(),
			key.WithHelp("[n]", "by number key"),
		),
		ConfigMaps: key.NewBinding(
			key.WithKeys(),
			key.WithHelp("[n]", "by number key"),
		),
		Secrets: key.NewBinding(
			key.WithKeys(),
			key.WithHelp("[n]", "by number key"),
		),
		Ingresses: key.NewBinding(
			key.WithKeys(),
			key.WithHelp("[n]", "by number key"),
		),
		PVCs: key.NewBinding(
			key.WithKeys(),
			key.WithHelp("[n]", "by number key"),
		),
		StatefulSets: key.NewBinding(
			key.WithKeys(),
			key.WithHelp("[n]", "by number key"),
		),
		Logs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "view logs"),
		),
		Describe: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "describe"),
		),
		Timeline: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "timeline"),
		),
		Xray: key.NewBinding(
			key.WithKeys("X"),
			key.WithHelp("X", "xray"),
		),
		Diagnosis: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "diagnosis"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next view"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev view"),
		),
		Delete: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "delete"),
		),
		Restart: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restart"),
		),
		Scale: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "scale"),
		),
		Shell: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "shell"),
		),
		PortForward: key.NewBinding(
			key.WithKeys("F"),
			key.WithHelp("F", "port-fwd"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Search: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "search"),
		),
		Namespace: key.NewBinding(
			key.WithKeys(),
			key.WithHelp("", "namespace"),
		),
		Context: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("ctrl+k", "context"),
		),
		Palette: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "command palette"),
		),
		CopyName: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy"),
		),
		YAML: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yaml"),
		),
		DecodeSecret: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "decode"),
		),
		HelmValues: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "values"),
		),
		HelmManifest: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "manifest"),
		),
		Command: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "command mode"),
		),
		DetailsPanel: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "toggle details"),
		),
		AutoRefresh: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "auto-refresh"),
		),
		ScrollLeft: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "scroll left"),
		),
		ScrollRight: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "scroll right"),
		),
		SortToggle: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "sort"),
		),
		SortColPrev: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "sort prev col"),
		),
		SortColNext: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "sort next col"),
		),
		LogSearch: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		LogSearchNext: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next match"),
		),
		LogSearchPrev: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "prev match"),
		),
		LogSave: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save logs"),
		),
		LogTimestamp: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "timestamps"),
		),
		LogPrevious: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "previous logs"),
		),
		LogTimeRange: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "time range"),
		),
		LogWrap: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "wrap"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n", "esc"),
			key.WithHelp("n/esc", "cancel"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/cancel"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns keybindings for the short help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Palette, k.Help, k.Quit}
}

// FullHelp returns keybindings for the full help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.PageUp, k.PageDown},
		{k.Pods, k.Deployments, k.Services, k.ConfigMaps, k.Secrets},
		{k.Ingresses, k.PVCs, k.StatefulSets, k.NextTab, k.PrevTab},
		{k.Logs, k.Describe, k.YAML, k.Edit, k.Delete, k.Restart, k.Scale, k.Shell, k.PortForward},
		{k.Filter, k.CopyName, k.Namespace, k.Context, k.Refresh, k.AutoRefresh},
		{k.SortToggle, k.SortColPrev, k.SortColNext, k.ScrollLeft, k.ScrollRight},
		{k.LogSearch, k.LogSearchNext, k.LogSearchPrev, k.LogSave, k.LogTimestamp, k.LogPrevious, k.LogTimeRange, k.LogWrap},
		{k.DetailsPanel, k.Palette, k.Help, k.Quit},
	}
}
