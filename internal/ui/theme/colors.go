package theme

import "charm.land/lipgloss/v2"

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

	// Search highlight
	ColorSearchHighlightBg = lipgloss.Color("#F59E0B") // Warning amber
	ColorSearchHighlightFg = lipgloss.Color("#1B1B3A") // Background dark

	// Delta row coloring
	ColorDeltaAdd    = lipgloss.Color("#87CEEB") // Sky blue - new resource
	ColorDeltaModify = lipgloss.Color("#B0C4DE") // Light steel blue - changed resource
	ColorDeltaError  = lipgloss.Color("#E08080") // Soft coral - unhealthy resource
	ColorDeltaDelete = lipgloss.Color("#708090") // Slate gray - future use
)

// Apply reassigns all 21 color variables from a ThemeDefinition.
// The 12 base colors come directly from td; the remaining 9 are derived
// unless explicitly overridden in the definition.
func Apply(td ThemeDefinition) {
	// 12 base colors
	ColorBackground = lipgloss.Color(td.Background)
	ColorSurface = lipgloss.Color(td.Surface)
	ColorText = lipgloss.Color(td.Text)
	ColorMuted = lipgloss.Color(td.Muted)
	ColorBorder = lipgloss.Color(td.Border)
	ColorHighlight = lipgloss.Color(td.Highlight)
	ColorPrimary = lipgloss.Color(td.Primary)
	ColorAccent = lipgloss.Color(td.Accent)
	ColorSuccess = lipgloss.Color(td.Success)
	ColorWarning = lipgloss.Color(td.Warning)
	ColorError = lipgloss.Color(td.Error)
	ColorInfo = lipgloss.Color(td.Info)

	// 9 derived colors (use override when provided, otherwise compute)
	ColorSecondary = lipgloss.Color(td.Info)

	if td.SurfaceAlt != "" {
		ColorSurfaceAlt = lipgloss.Color(td.SurfaceAlt)
	} else {
		ColorSurfaceAlt = blendColor(td.Background, td.Surface, 0.5)
	}

	if td.FrameBorder != "" {
		ColorFrameBorder = lipgloss.Color(td.FrameBorder)
	} else {
		ColorFrameBorder = lightenColor(td.Border, 0.3)
	}

	if td.SelectionBg != "" {
		ColorSelectionBg = lipgloss.Color(td.SelectionBg)
	} else {
		ColorSelectionBg = darkenColor(td.Accent, 0.6)
	}

	if td.SelectionFg != "" {
		ColorSelectionFg = lipgloss.Color(td.SelectionFg)
	} else {
		ColorSelectionFg = contrastForeground(td.SelectionBg, td.Accent)
	}

	ColorLabelPrefix = lightenColor(td.Primary, 0.2)
	ColorRowNumber = lipgloss.Color(td.Muted)
	ColorNAValue = lipgloss.Color(td.Muted)
	ColorCompletedText = lightenColor(td.Muted, 0.15)

	// Search highlight colors
	ColorSearchHighlightBg = lipgloss.Color(td.Warning)
	if td.SearchHighlightBg != "" {
		ColorSearchHighlightBg = lipgloss.Color(td.SearchHighlightBg)
	}
	if td.SearchHighlightFg != "" {
		ColorSearchHighlightFg = lipgloss.Color(td.SearchHighlightFg)
	} else {
		// Luminance-based: dark text on bright warning, bright text on dark warning
		if luminance(td.Warning) > 0.179 {
			ColorSearchHighlightFg = lipgloss.Color("#000000")
		} else {
			ColorSearchHighlightFg = lipgloss.Color("#FFFFFF")
		}
	}

	// Delta row colors
	if td.DeltaAdd != "" {
		ColorDeltaAdd = lipgloss.Color(td.DeltaAdd)
	} else {
		ColorDeltaAdd = lightenColor(td.Info, 0.3)
	}
	if td.DeltaModify != "" {
		ColorDeltaModify = lipgloss.Color(td.DeltaModify)
	} else {
		ColorDeltaModify = blendColor(td.Highlight, td.Muted, 0.4)
	}
	if td.DeltaError != "" {
		ColorDeltaError = lipgloss.Color(td.DeltaError)
	} else {
		ColorDeltaError = lightenColor(td.Error, 0.2)
	}
	if td.DeltaDelete != "" {
		ColorDeltaDelete = lipgloss.Color(td.DeltaDelete)
	} else {
		ColorDeltaDelete = lipgloss.Color(td.Muted)
	}
}

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
