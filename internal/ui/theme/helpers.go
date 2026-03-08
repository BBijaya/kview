package theme

import (
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// StatusStyle returns the appropriate style for a given status
func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "Running", "Succeeded", "Active", "Available", "Ready", "Healthy", "Completed", "Complete", "Normal":
		return Styles.StatusHealthy
	case "Pending", "Waiting", "Progressing":
		return Styles.StatusPending
	case "Warning", "Degraded":
		return Styles.StatusWarning
	case "Failed", "Error", "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull", "Terminating", "OOMKilled":
		return Styles.StatusError
	default:
		return Styles.StatusUnknown
	}
}

// StatusCellStyle returns style with colored background for status cells
func StatusCellStyle(status string) lipgloss.Style {
	switch status {
	case "Running", "Succeeded", "Active", "Available", "Ready", "Healthy", "Completed", "Complete", "Normal":
		return Styles.StatusCellHealthy
	case "Pending", "Waiting", "Progressing":
		return Styles.StatusCellPending
	case "Warning", "Degraded":
		return Styles.StatusCellWarning
	case "Failed", "Error", "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull", "Terminating", "OOMKilled":
		return Styles.StatusCellError
	default:
		return Styles.StatusUnknown.Background(ColorBackground)
	}
}

// IsDeltaErrorStatus returns true if the status represents an unhealthy state
// that should be highlighted with delta error coloring.
func IsDeltaErrorStatus(status string) bool {
	switch status {
	case "Failed", "Error", "CrashLoopBackOff", "ImagePullBackOff",
		"ErrImagePull", "Terminating", "OOMKilled",
		"Warning", "Degraded":
		return true
	default:
		return false
	}
}

// TruncateWithIndicator truncates a string with a styled "..." indicator
func TruncateWithIndicator(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	truncated := s[:maxLen-3]
	indicator := Styles.InfoValueMuted.Render("...")
	return truncated + indicator
}

// StatusIconPrefix returns the plain icon prefix (no ANSI styling) for a given status.
// Use this instead of StatusWithIcon when the result will be passed through TruncateString
// or other byte-based string operations, to avoid corrupting ANSI escape sequences.
func StatusIconPrefix(status string) string {
	switch status {
	case "Running", "Succeeded", "Active", "Available", "Ready", "Healthy", "Completed", "Complete", "Normal":
		return IconSuccess + " "
	case "Pending", "Waiting", "Progressing":
		return IconPending + " "
	case "Warning", "Degraded":
		return IconWarning + " "
	case "Failed", "Error", "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull", "Terminating", "OOMKilled":
		return IconError + " "
	default:
		return IconUnknown + " "
	}
}

// StatusWithIcon returns a status string with an appropriate icon prefix
func StatusWithIcon(status string) string {
	var icon string
	var style lipgloss.Style

	switch status {
	case "Running", "Succeeded", "Active", "Available", "Ready", "Healthy", "Completed", "Complete", "Normal":
		icon = IconSuccess
		style = Styles.StatusHealthy
	case "Pending", "Waiting", "Progressing":
		icon = IconPending
		style = Styles.StatusPending
	case "Warning", "Degraded":
		icon = IconWarning
		style = Styles.StatusWarning
	case "Failed", "Error", "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull", "Terminating", "OOMKilled":
		icon = IconError
		style = Styles.StatusError
	default:
		icon = IconUnknown
		style = Styles.StatusUnknown
	}

	return style.Render(icon + " " + status)
}

// FormatAge formats a duration as a human-readable age string
func FormatAge(d interface{}) string {
	switch v := d.(type) {
	case int64:
		return formatDuration(v)
	case time.Duration:
		return formatDuration(int64(v.Seconds()))
	default:
		return "?"
	}
}

func formatDuration(seconds int64) string {
	if seconds < 60 {
		return intToString(seconds) + "s"
	}
	if seconds < 3600 {
		return intToString(seconds/60) + "m"
	}
	if seconds < 86400 {
		return intToString(seconds/3600) + "h"
	}
	return intToString(seconds/86400) + "d"
}

func intToString(n int64) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte(n%10) + '0'}, result...)
		n /= 10
	}
	return string(result)
}

// TruncateString truncates a string to a maximum visual width with ellipsis.
// Uses lipgloss.Width() for visual width and rune slicing to avoid cutting
// multi-byte UTF-8 characters (e.g., ✓, ✗, ○).
func TruncateString(s string, maxLen int) string {
	if lipgloss.Width(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		runes := []rune(s)
		if len(runes) > maxLen {
			return string(runes[:maxLen])
		}
		return s
	}
	runes := []rune(s)
	// Trim runes from the end until visual width fits within maxLen-3 (room for "...")
	for len(runes) > 0 && lipgloss.Width(string(runes)) > maxLen-3 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
}

// PadToWidth pads a string to a target width with styled background spaces.
// Uses lipgloss.Width() to measure visual width and appends independently
// styled padding to avoid background color leaks.
func PadToWidth(s string, width int, bg color.Color) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	padding := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", width-w))
	return s + padding
}

// PadRight pads a string to the right to a fixed width
func PadRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	result := s
	for len(result) < width {
		result += " "
	}
	return result
}

// PadLeft pads a string to the left to a fixed width
func PadLeft(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	result := s
	for len(result) < width {
		result = " " + result
	}
	return result
}
