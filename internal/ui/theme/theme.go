package theme

import "charm.land/lipgloss/v2"

// Styles provides common styling for UI components
var Styles = struct {
	// Base styles
	Base     lipgloss.Style
	Focused  lipgloss.Style
	Selected lipgloss.Style

	// Header and footer
	Header    lipgloss.Style
	StatusBar lipgloss.Style

	// Table styles
	TableHeader      lipgloss.Style
	TableRow         lipgloss.Style
	TableRowSelected lipgloss.Style

	// Status styles
	StatusHealthy lipgloss.Style
	StatusWarning lipgloss.Style
	StatusError   lipgloss.Style
	StatusPending lipgloss.Style
	StatusUnknown lipgloss.Style

	// Component styles
	Tab          lipgloss.Style
	TabActive    lipgloss.Style
	Dialog       lipgloss.Style
	DialogTitle  lipgloss.Style
	Input        lipgloss.Style
	InputFocused lipgloss.Style
	Help         lipgloss.Style
	HelpKey      lipgloss.Style
	HelpDesc     lipgloss.Style

	// Palette styles
	PaletteContainer lipgloss.Style
	PaletteInput     lipgloss.Style
	PaletteItem      lipgloss.Style
	PaletteSelected  lipgloss.Style

	// Panel styles
	Panel       lipgloss.Style
	PanelTitle  lipgloss.Style
	PanelBorder lipgloss.Style

	// Frame styles (k9s-like layout)
	Frame        lipgloss.Style
	FrameTitle   lipgloss.Style
	FrameVersion lipgloss.Style
	FrameDivider lipgloss.Style

	// TabBar styles (numbered tabs)
	TabBar           lipgloss.Style
	TabBarItem       lipgloss.Style
	TabBarItemActive lipgloss.Style
	TabBarNumber     lipgloss.Style

	// Command input styles
	CommandLine   lipgloss.Style
	CommandPrefix lipgloss.Style
	CommandInput  lipgloss.Style

	// Row indicator
	RowIndicator     lipgloss.Style
	RowIndicatorNone lipgloss.Style

	// Compact header
	HeaderCompact   lipgloss.Style
	HeaderContext   lipgloss.Style
	HeaderNamespace lipgloss.Style
	HeaderCluster   lipgloss.Style
	HeaderSeparator lipgloss.Style

	// Alternating rows
	TableRowAlt lipgloss.Style

	// Content area (solid background)
	ContentArea lipgloss.Style

	// Divider with background
	DividerLine lipgloss.Style

	// Category tabs (two-tier navigation)
	CategoryRow        lipgloss.Style
	CategoryItem       lipgloss.Style
	CategoryItemActive lipgloss.Style
	CategoryIndicator  lipgloss.Style
	ResourceRow        lipgloss.Style
	ResourceItem       lipgloss.Style
	ResourceItemActive lipgloss.Style

	// Info pane styles (k9s-like top pane)
	InfoPane       lipgloss.Style
	InfoLabel      lipgloss.Style
	InfoValue      lipgloss.Style
	InfoValueMuted lipgloss.Style
	ThinDivider    lipgloss.Style

	// Shortcuts footer
	ShortcutsBar lipgloss.Style
	ShortcutKey  lipgloss.Style
	ShortcutDesc lipgloss.Style

	// Row numbers
	RowNumber lipgloss.Style

	// Status cell backgrounds
	StatusCellHealthy lipgloss.Style
	StatusCellWarning lipgloss.Style
	StatusCellError   lipgloss.Style
	StatusCellPending lipgloss.Style

	// Selection states
	TableRowSelectedFocused lipgloss.Style
	TableRowSelectedBlurred lipgloss.Style
	RowIndicatorFocused     lipgloss.Style
	RowIndicatorBlurred     lipgloss.Style

	// Empty states
	EmptyStateIcon    lipgloss.Style
	EmptyStateTitle   lipgloss.Style
	EmptyStateMessage lipgloss.Style
	EmptyStateHint    lipgloss.Style

	// Header enhancements
	InfoLabelPrefix lipgloss.Style
	InfoValueNA     lipgloss.Style
}{}

// onComputeStylesHooks are callbacks invoked after ComputeStyles() completes.
// Used by packages that cache theme-derived state (e.g., chroma syntax styles).
var onComputeStylesHooks []func()

// OnComputeStyles registers a callback to run after ComputeStyles().
func OnComputeStyles(fn func()) {
	onComputeStylesHooks = append(onComputeStylesHooks, fn)
}

func init() {
	ComputeStyles()
}

// ComputeStyles rebuilds all Styles from the current Color* variables.
// Called at init time for defaults, and again from main() after Apply().
func ComputeStyles() {
	Styles.Base = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorBackground)

	Styles.Focused = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Bold(true)

	Styles.Selected = lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(ColorText)

	Styles.Header = lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(ColorText).
		Padding(0, 1).
		Bold(true)

	Styles.StatusBar = lipgloss.NewStyle().
		Background(ColorSurface).
		Foreground(ColorMuted).
		Padding(0, 1)

	Styles.TableHeader = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Background(ColorBackground).
		Padding(0, 1)

	Styles.TableRow = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorBackground).
		Padding(0, 1)

	Styles.TableRowSelected = lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(ColorText).
		Padding(0, 1)

	Styles.StatusHealthy = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	Styles.StatusWarning = lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true)

	Styles.StatusError = lipgloss.NewStyle().
		Foreground(ColorError).
		Bold(true)

	Styles.StatusPending = lipgloss.NewStyle().
		Foreground(ColorInfo).
		Bold(true)

	Styles.StatusUnknown = lipgloss.NewStyle().
		Foreground(ColorMuted)

	Styles.Tab = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Padding(0, 2)

	Styles.TabActive = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorSurface).
		Padding(0, 2).
		Bold(true)

	Styles.Dialog = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	Styles.DialogTitle = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Bold(true).
		MarginBottom(1)

	Styles.Input = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	Styles.InputFocused = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1)

	Styles.Help = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground)

	Styles.HelpKey = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorBackground)

	Styles.HelpDesc = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground)

	Styles.PaletteContainer = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Width(60)

	Styles.PaletteInput = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorSurface).
		Padding(0, 1)

	Styles.PaletteItem = lipgloss.NewStyle().
		Foreground(ColorText).
		Padding(0, 1)

	Styles.PaletteSelected = lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(ColorText).
		Padding(0, 1)

	Styles.Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder)

	Styles.PanelTitle = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorBackground).
		Bold(true)

	Styles.PanelBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder)

	// Frame styles (k9s-like layout)
	Styles.Frame = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder)

	Styles.FrameTitle = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Bold(true)

	Styles.FrameVersion = lipgloss.NewStyle().
		Foreground(ColorMuted)

	Styles.FrameDivider = lipgloss.NewStyle().
		Foreground(ColorBorder).
		Background(ColorBackground)

	// TabBar styles (numbered tabs)
	Styles.TabBar = lipgloss.NewStyle().
		Background(ColorSurface)

	Styles.TabBarItem = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Padding(0, 1)

	Styles.TabBarItemActive = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorSurface).
		Bold(true).
		Padding(0, 1)

	Styles.TabBarNumber = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Background(ColorBackground).
		Bold(true)

	// Command input styles
	Styles.CommandLine = lipgloss.NewStyle().
		Background(ColorBackground).
		Foreground(ColorText)

	Styles.CommandPrefix = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorBackground).
		Bold(true)

	Styles.CommandInput = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorBackground)

	// Row indicator
	Styles.RowIndicator = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorBackground).
		Bold(true)

	Styles.RowIndicatorNone = lipgloss.NewStyle().
		Foreground(ColorSurface).
		Background(ColorBackground)

	// Compact header
	Styles.HeaderCompact = lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(ColorText).
		Padding(0, 1).
		Bold(true)

	Styles.HeaderContext = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Background(ColorBackground)

	Styles.HeaderNamespace = lipgloss.NewStyle().
		Foreground(ColorWarning).
		Background(ColorBackground)

	Styles.HeaderCluster = lipgloss.NewStyle().
		Foreground(ColorInfo).
		Background(ColorBackground)

	Styles.HeaderSeparator = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground)

	// Alternating row
	Styles.TableRowAlt = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorSurfaceAlt).
		Padding(0, 1)

	// Content area (solid background for filling empty space)
	Styles.ContentArea = lipgloss.NewStyle().
		Background(ColorBackground)

	// Divider line with background
	Styles.DividerLine = lipgloss.NewStyle().
		Foreground(ColorBorder).
		Background(ColorBackground)

	// Category tabs (two-tier navigation)
	Styles.CategoryRow = lipgloss.NewStyle().
		Background(ColorSurface)

	Styles.CategoryItem = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground).
		Padding(0, 2)

	Styles.CategoryItemActive = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorBackground).
		Bold(true).
		Padding(0, 2)

	Styles.CategoryIndicator = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Background(ColorBackground).
		Bold(true)

	Styles.ResourceRow = lipgloss.NewStyle().
		Background(ColorBackground)

	Styles.ResourceItem = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground).
		Padding(0, 1)

	Styles.ResourceItemActive = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorSurface).
		Bold(true).
		Padding(0, 1)

	// Info pane styles (k9s-like top pane)
	Styles.InfoPane = lipgloss.NewStyle().
		Background(ColorBackground)

	Styles.InfoLabel = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground)

	Styles.InfoValue = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Background(ColorBackground)

	Styles.InfoValueMuted = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground)

	Styles.ThinDivider = lipgloss.NewStyle().
		Foreground(ColorBorder).
		Background(ColorBackground)

	// Shortcuts footer
	Styles.ShortcutsBar = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground)

	Styles.ShortcutKey = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorBackground)

	Styles.ShortcutDesc = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground)

	// Row numbers
	Styles.RowNumber = lipgloss.NewStyle().
		Foreground(ColorRowNumber).
		Background(ColorBackground).
		Align(lipgloss.Right).
		PaddingRight(1)

	// Status cell styles (no special background — use main background)
	Styles.StatusCellHealthy = lipgloss.NewStyle().
		Background(ColorBackground).
		Foreground(ColorSuccess)

	Styles.StatusCellWarning = lipgloss.NewStyle().
		Background(ColorBackground).
		Foreground(ColorWarning)

	Styles.StatusCellError = lipgloss.NewStyle().
		Background(ColorBackground).
		Foreground(ColorError)

	Styles.StatusCellPending = lipgloss.NewStyle().
		Background(ColorBackground).
		Foreground(ColorInfo)

	// Selection states
	Styles.TableRowSelectedFocused = lipgloss.NewStyle().
		Background(ColorSelectionBg).
		Foreground(ColorSelectionFg).
		Bold(true).
		Padding(0, 1)

	Styles.TableRowSelectedBlurred = lipgloss.NewStyle().
		Background(ColorSurface).
		Foreground(ColorMuted).
		Padding(0, 1)

	Styles.RowIndicatorFocused = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorSelectionBg).
		Bold(true)

	Styles.RowIndicatorBlurred = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorSurface)

	// Empty states
	Styles.EmptyStateIcon = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground).
		Align(lipgloss.Center)

	Styles.EmptyStateTitle = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorBackground).
		Bold(true).
		Align(lipgloss.Center)

	Styles.EmptyStateMessage = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground).
		Align(lipgloss.Center)

	Styles.EmptyStateHint = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorBackground).
		Italic(true).
		Align(lipgloss.Center)

	// Header enhancements
	Styles.InfoLabelPrefix = lipgloss.NewStyle().
		Foreground(ColorLabelPrefix).
		Background(ColorBackground).
		Bold(true)

	Styles.InfoValueNA = lipgloss.NewStyle().
		Foreground(ColorNAValue).
		Background(ColorBackground).
		Italic(true)

	// Notify registered hooks (e.g., chroma style rebuild).
	for _, fn := range onComputeStylesHooks {
		fn()
	}
}
