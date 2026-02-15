package theme

import "github.com/charmbracelet/lipgloss"

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
	Frame            lipgloss.Style
	FrameTitle       lipgloss.Style
	FrameVersion     lipgloss.Style
	FrameDivider     lipgloss.Style

	// TabBar styles (numbered tabs)
	TabBar           lipgloss.Style
	TabBarItem       lipgloss.Style
	TabBarItemActive lipgloss.Style
	TabBarNumber     lipgloss.Style

	// Command input styles
	CommandLine      lipgloss.Style
	CommandPrefix    lipgloss.Style
	CommandInput     lipgloss.Style

	// Row indicator
	RowIndicator     lipgloss.Style
	RowIndicatorNone lipgloss.Style

	// Compact header
	HeaderCompact    lipgloss.Style
	HeaderContext    lipgloss.Style
	HeaderNamespace  lipgloss.Style
	HeaderCluster    lipgloss.Style
	HeaderSeparator  lipgloss.Style

	// Alternating rows
	TableRowAlt lipgloss.Style

	// Content area (solid background)
	ContentArea lipgloss.Style

	// Divider with background
	DividerLine lipgloss.Style

	// Category tabs (two-tier navigation)
	CategoryRow         lipgloss.Style
	CategoryItem        lipgloss.Style
	CategoryItemActive  lipgloss.Style
	CategoryIndicator   lipgloss.Style
	ResourceRow         lipgloss.Style
	ResourceItem        lipgloss.Style
	ResourceItemActive  lipgloss.Style

	// Info pane styles (k9s-like top pane)
	InfoPane       lipgloss.Style
	InfoLabel      lipgloss.Style
	InfoValue      lipgloss.Style
	InfoValueMuted lipgloss.Style
	ThinDivider    lipgloss.Style

	// Shortcuts footer
	ShortcutsBar   lipgloss.Style
	ShortcutKey    lipgloss.Style
	ShortcutDesc   lipgloss.Style

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
}{
	Base: lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorBackground),

	Focused: lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Bold(true),

	Selected: lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(ColorText),

	Header: lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(ColorText).
		Padding(0, 1).
		Bold(true),

	StatusBar: lipgloss.NewStyle().
		Background(ColorSurface).
		Foreground(ColorMuted).
		Padding(0, 1),

	TableHeader: lipgloss.NewStyle().
		Foreground(ColorAccent).
		Background(ColorBackground).
		Padding(0, 1),

	TableRow: lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorBackground).
		Padding(0, 1),

	TableRowSelected: lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(ColorText).
		Padding(0, 1),

	StatusHealthy: lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true),

	StatusWarning: lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true),

	StatusError: lipgloss.NewStyle().
		Foreground(ColorError).
		Bold(true),

	StatusPending: lipgloss.NewStyle().
		Foreground(ColorInfo).
		Bold(true),

	StatusUnknown: lipgloss.NewStyle().
		Foreground(ColorMuted),

	Tab: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Padding(0, 2),

	TabActive: lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorSurface).
		Padding(0, 2).
		Bold(true),

	Dialog: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2),

	DialogTitle: lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Bold(true).
		MarginBottom(1),

	Input: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1),

	InputFocused: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1),

	Help: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground),

	HelpKey: lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorBackground),

	HelpDesc: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground),

	PaletteContainer: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Width(60),

	PaletteInput: lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorSurface).
		Padding(0, 1),

	PaletteItem: lipgloss.NewStyle().
		Foreground(ColorText).
		Padding(0, 1),

	PaletteSelected: lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(ColorText).
		Padding(0, 1),

	Panel: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder),

	PanelTitle: lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorBackground).
		Bold(true),

	PanelBorder: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder),

	// Frame styles (k9s-like layout)
	Frame: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder),

	FrameTitle: lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Bold(true),

	FrameVersion: lipgloss.NewStyle().
		Foreground(ColorMuted),

	FrameDivider: lipgloss.NewStyle().
		Foreground(ColorBorder).
		Background(ColorBackground),

	// TabBar styles (numbered tabs)
	TabBar: lipgloss.NewStyle().
		Background(ColorSurface),

	TabBarItem: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Padding(0, 1),

	TabBarItemActive: lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorSurface).
		Bold(true).
		Padding(0, 1),

	TabBarNumber: lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Background(ColorBackground).
		Bold(true),

	// Command input styles
	CommandLine: lipgloss.NewStyle().
		Background(ColorBackground).
		Foreground(ColorText),

	CommandPrefix: lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorBackground).
		Bold(true),

	CommandInput: lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorBackground),

	// Row indicator
	RowIndicator: lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorBackground).
		Bold(true),

	RowIndicatorNone: lipgloss.NewStyle().
		Foreground(ColorSurface).
		Background(ColorBackground),

	// Compact header
	HeaderCompact: lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(ColorText).
		Padding(0, 1).
		Bold(true),

	HeaderContext: lipgloss.NewStyle().
		Foreground(ColorAccent).
		Background(ColorBackground),

	HeaderNamespace: lipgloss.NewStyle().
		Foreground(ColorWarning).
		Background(ColorBackground),

	HeaderCluster: lipgloss.NewStyle().
		Foreground(ColorInfo).
		Background(ColorBackground),

	HeaderSeparator: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground),

	// Alternating row
	TableRowAlt: lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorSurfaceAlt).
		Padding(0, 1),

	// Content area (solid background for filling empty space)
	ContentArea: lipgloss.NewStyle().
		Background(ColorBackground),

	// Divider line with background
	DividerLine: lipgloss.NewStyle().
		Foreground(ColorBorder).
		Background(ColorBackground),

	// Category tabs (two-tier navigation)
	CategoryRow: lipgloss.NewStyle().
		Background(ColorSurface),

	CategoryItem: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground).
		Padding(0, 2),

	CategoryItemActive: lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorBackground).
		Bold(true).
		Padding(0, 2),

	CategoryIndicator: lipgloss.NewStyle().
		Foreground(ColorAccent).
		Background(ColorBackground).
		Bold(true),

	ResourceRow: lipgloss.NewStyle().
		Background(ColorBackground),

	ResourceItem: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground).
		Padding(0, 1),

	ResourceItemActive: lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorSurface).
		Bold(true).
		Padding(0, 1),

	// Info pane styles (k9s-like top pane)
	InfoPane: lipgloss.NewStyle().
		Background(ColorBackground),

	InfoLabel: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground),

	InfoValue: lipgloss.NewStyle().
		Foreground(ColorAccent).
		Background(ColorBackground),

	InfoValueMuted: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground),

	ThinDivider: lipgloss.NewStyle().
		Foreground(ColorBorder).
		Background(ColorBackground),

	// Shortcuts footer
	ShortcutsBar: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground),

	ShortcutKey: lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorBackground),

	ShortcutDesc: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground),

	// Row numbers
	RowNumber: lipgloss.NewStyle().
		Foreground(ColorRowNumber).
		Background(ColorBackground).
		Align(lipgloss.Right).
		PaddingRight(1),

	// Status cell styles (no special background — use main background)
	StatusCellHealthy: lipgloss.NewStyle().
		Background(ColorBackground).
		Foreground(ColorSuccess),

	StatusCellWarning: lipgloss.NewStyle().
		Background(ColorBackground).
		Foreground(ColorWarning),

	StatusCellError: lipgloss.NewStyle().
		Background(ColorBackground).
		Foreground(ColorError),

	StatusCellPending: lipgloss.NewStyle().
		Background(ColorBackground).
		Foreground(ColorInfo),

	// Selection states
	TableRowSelectedFocused: lipgloss.NewStyle().
		Background(ColorSelectionBg).
		Foreground(ColorSelectionFg).
		Bold(true).
		Padding(0, 1),

	TableRowSelectedBlurred: lipgloss.NewStyle().
		Background(ColorSurface).
		Foreground(ColorMuted).
		Padding(0, 1),

	RowIndicatorFocused: lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorSelectionBg).
		Bold(true),

	RowIndicatorBlurred: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorSurface),

	// Empty states
	EmptyStateIcon: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground).
		Align(lipgloss.Center),

	EmptyStateTitle: lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorBackground).
		Bold(true).
		Align(lipgloss.Center),

	EmptyStateMessage: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBackground).
		Align(lipgloss.Center),

	EmptyStateHint: lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorBackground).
		Italic(true).
		Align(lipgloss.Center),

	// Header enhancements
	InfoLabelPrefix: lipgloss.NewStyle().
		Foreground(ColorLabelPrefix).
		Background(ColorBackground).
		Bold(true),

	InfoValueNA: lipgloss.NewStyle().
		Foreground(ColorNAValue).
		Background(ColorBackground).
		Italic(true),
}
