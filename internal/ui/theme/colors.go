package theme

import "github.com/charmbracelet/lipgloss"

// Colors for the UI (k9s-inspired palette)
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#2563EB") // Blue
	ColorAccent    = lipgloss.Color("#06B6D4") // Teal/Cyan - for highlights

	// Status colors
	ColorSuccess = lipgloss.Color("#10B981") // Green
	ColorWarning = lipgloss.Color("#F59E0B") // Amber
	ColorError   = lipgloss.Color("#EF4444") // Red
	ColorInfo    = lipgloss.Color("#3B82F6") // Blue

	// Neutral colors
	ColorBackground  = lipgloss.Color("#1B1B3A") // Dark indigo (clearly visible on black terminal)
	ColorSurface     = lipgloss.Color("#1A1A2E") // For elevated elements
	ColorSurfaceAlt  = lipgloss.Color("#1F1F35") // Alternating row color
	ColorBorder      = lipgloss.Color("#3D3D5C") // More visible borders
	ColorText        = lipgloss.Color("#E2E8F0") // Slightly warmer text
	ColorMuted       = lipgloss.Color("#64748B") // Secondary text
	ColorHighlight   = lipgloss.Color("#89B4FA") // Highlight
	ColorFrameBorder = lipgloss.Color("#5D5D8C") // Frame border (brighter)

	// Status cell backgrounds (subtle, dark variants)

	// Selection enhancements
	ColorSelectionBg = lipgloss.Color("#005F87") // Dark teal/cyan (k9s-style)
	ColorSelectionFg = lipgloss.Color("#FFFFFF") // White text

	// Other
	ColorLabelPrefix = lipgloss.Color("#6366F1") // Indigo for prefixes
	ColorRowNumber   = lipgloss.Color("#6B7B8F") // Gray for row numbers
	ColorNAValue       = lipgloss.Color("#6B7B8F") // Dim for n/a values
	ColorCompletedText = lipgloss.Color("#8B95A5") // Light grey for completed row text
)

// Status icons
const (
	IconSuccess = "✓"
	IconWarning = "⚠"
	IconError   = "✗"
	IconRunning = "●"
	IconPending = "○"
	IconUnknown = "?"

	// Enhanced icons (larger/more visible)
	IconSuccessLarge = "✔"  // Heavy checkmark
	IconWarningLarge = "⚠"  // Warning triangle
	IconErrorLarge   = "✖"  // Heavy X
	IconRunningLarge = "●"  // Filled circle
	IconPendingLarge = "◌"  // Dashed circle
	IconEmptyBox     = "📦" // Empty state icon
)
