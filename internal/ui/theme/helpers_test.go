package theme

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"zero seconds", int64(0), "0s"},
		{"seconds", int64(45), "45s"},
		{"last second before minute", int64(59), "59s"},
		{"exact minute", int64(60), "1m"},
		{"minutes truncate", int64(119), "1m"},
		{"minutes", int64(150), "2m"},
		{"last minute before hour", int64(3599), "59m"},
		{"exact hour", int64(3600), "1h"},
		{"hours", int64(7200), "2h"},
		{"last hour before day", int64(86399), "23h"},
		{"exact day", int64(86400), "1d"},
		{"days", int64(86400 * 5), "5d"},
		{"large days", int64(86400 * 365), "365d"},
		{"time.Duration seconds", 45 * time.Second, "45s"},
		{"time.Duration hours", 3 * time.Hour, "3h"},
		{"unsupported type", "not a duration", "?"},
		{"unsupported int", 60, "?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatAge(tt.input); got != tt.want {
				t.Errorf("FormatAge(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"shorter than max", "hello", 10, "hello"},
		{"exact max", "hello", 5, "hello"},
		{"truncated with ellipsis", "hello world", 8, "hello..."},
		{"maxLen 3 hard cut", "hello", 3, "hel"},
		{"maxLen 1 hard cut", "hello", 1, "h"},
		{"empty string", "", 5, ""},
		{"unicode kept intact", "✓✓✓", 5, "✓✓✓"},
		{"unicode truncated", "✓✓✓✓✓✓✓✓", 6, "✓✓✓..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
			if lipgloss.Width(got) > tt.maxLen {
				t.Errorf("TruncateString(%q, %d) visual width %d exceeds max", tt.input, tt.maxLen, lipgloss.Width(got))
			}
		})
	}
}

func TestPadToWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
	}{
		{"pads short string", "abc", 10},
		{"already at width", "abcde", 5},
		{"wider than target unchanged", "abcdef", 3},
		{"empty string", "", 4},
		{"unicode input", "✓ ok", 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PadToWidth(tt.input, tt.width, ColorBackground)
			inWidth := lipgloss.Width(tt.input)
			wantWidth := tt.width
			if inWidth > tt.width {
				wantWidth = inWidth
			}
			if w := lipgloss.Width(got); w != wantWidth {
				t.Errorf("PadToWidth(%q, %d) visual width = %d, want %d", tt.input, tt.width, w, wantWidth)
			}
			if !strings.HasPrefix(got, tt.input) {
				t.Errorf("PadToWidth(%q, %d) = %q, does not start with input", tt.input, tt.width, got)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{"pads", "ab", 5, "ab   "},
		{"exact width", "abcde", 5, "abcde"},
		{"truncates when longer", "abcdef", 4, "abcd"},
		{"empty", "", 3, "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PadRight(tt.input, tt.width); got != tt.want {
				t.Errorf("PadRight(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{"pads", "ab", 5, "   ab"},
		{"exact width", "abcde", 5, "abcde"},
		{"truncates when longer", "abcdef", 4, "abcd"},
		{"empty", "", 3, "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PadLeft(tt.input, tt.width); got != tt.want {
				t.Errorf("PadLeft(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}

func TestIsDeltaErrorStatus(t *testing.T) {
	errorStatuses := []string{
		"Failed", "Error", "CrashLoopBackOff", "ImagePullBackOff",
		"ErrImagePull", "Terminating", "OOMKilled", "Warning", "Degraded",
	}
	for _, s := range errorStatuses {
		if !IsDeltaErrorStatus(s) {
			t.Errorf("IsDeltaErrorStatus(%q) = false, want true", s)
		}
	}

	healthyStatuses := []string{"Running", "Succeeded", "Pending", "Completed", "", "SomethingElse"}
	for _, s := range healthyStatuses {
		if IsDeltaErrorStatus(s) {
			t.Errorf("IsDeltaErrorStatus(%q) = true, want false", s)
		}
	}
}
