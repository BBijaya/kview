package k8s

import "testing"

func TestParseCPU(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64 // nanocores
	}{
		{"nanocores", "100n", 100},
		{"microcores", "5u", 5000},
		{"millicores", "250m", 250_000_000},
		{"whole cores", "2", 2_000_000_000},
		{"single core", "1", 1_000_000_000},
		{"zero", "0", 0},
		{"whitespace trimmed", " 100m ", 100_000_000},
		{"empty string", "", 0},
		{"garbage", "abc", 0},
		{"fractional not supported", "1.5", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseCPU(tt.input); got != tt.want {
				t.Errorf("parseCPU(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseMemory(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64 // bytes
	}{
		{"kibibytes", "4Ki", 4 * 1024},
		{"mebibytes", "128Mi", 128 * 1024 * 1024},
		{"gibibytes", "2Gi", 2 * 1024 * 1024 * 1024},
		{"tebibytes", "1Ti", 1024 * 1024 * 1024 * 1024},
		{"kilobytes decimal", "5k", 5000},
		{"megabytes decimal", "3M", 3_000_000},
		{"gigabytes decimal", "1G", 1_000_000_000},
		{"plain bytes", "1024", 1024},
		{"zero", "0", 0},
		{"whitespace trimmed", " 64Mi ", 64 * 1024 * 1024},
		{"empty string", "", 0},
		{"garbage", "abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseMemory(tt.input); got != tt.want {
				t.Errorf("parseMemory(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatCPU(t *testing.T) {
	tests := []struct {
		name  string
		nanos int64
		want  string
	}{
		{"zero", 0, "0m"},
		{"millicores", 250_000_000, "250m"},
		{"just below a core", 999_000_000, "999m"},
		{"exactly one core", 1_000_000_000, "1.0"},
		{"one and a half cores", 1_500_000_000, "1.5"},
		{"many cores", 8_000_000_000, "8.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatCPU(tt.nanos); got != tt.want {
				t.Errorf("FormatCPU(%d) = %q, want %q", tt.nanos, got, tt.want)
			}
		})
	}
}

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero", 0, "0Mi"},
		{"mebibytes", 128 * 1024 * 1024, "128Mi"},
		{"below one Mi", 512 * 1024, "0Mi"},
		{"just below one Gi", 1023 * 1024 * 1024, "1023Mi"},
		{"gibibytes", 2 * 1024 * 1024 * 1024, "2Gi"},
		{"partial Gi truncates", 1536 * 1024 * 1024, "1Gi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatMemory(tt.bytes); got != tt.want {
				t.Errorf("FormatMemory(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}
