package ui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/ui/theme"
)

// Re-export theme items for backward compatibility
var (
	// Colors
	ColorPrimary    = theme.ColorPrimary
	ColorSecondary  = theme.ColorSecondary
	ColorAccent     = theme.ColorAccent
	ColorSuccess    = theme.ColorSuccess
	ColorWarning    = theme.ColorWarning
	ColorError      = theme.ColorError
	ColorInfo       = theme.ColorInfo
	ColorBackground = theme.ColorBackground
	ColorSurface    = theme.ColorSurface
	ColorBorder     = theme.ColorBorder
	ColorText       = theme.ColorText
	ColorMuted      = theme.ColorMuted
	ColorHighlight  = theme.ColorHighlight

	// Styles
	Styles = theme.Styles
)

// StatusStyle returns the appropriate style for a given status
func StatusStyle(status string) lipgloss.Style {
	return theme.StatusStyle(status)
}

// FormatAge formats a duration as a human-readable age string
func FormatAge(d interface{}) string {
	return theme.FormatAge(d)
}

// TruncateString truncates a string to a maximum length with ellipsis
func TruncateString(s string, maxLen int) string {
	return theme.TruncateString(s, maxLen)
}

// PadRight pads a string to the right to a fixed width
func PadRight(s string, width int) string {
	return theme.PadRight(s, width)
}

// PadLeft pads a string to the left to a fixed width
func PadLeft(s string, width int) string {
	return theme.PadLeft(s, width)
}
