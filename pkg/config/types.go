package config

// Workflow represents a saved workflow/runbook
type Workflow struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Steps       []WorkflowStep `yaml:"steps"`
}

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Action      string            `yaml:"action"`
	Args        map[string]string `yaml:"args"`
	Confirm     bool              `yaml:"confirm"`
	OnError     string            `yaml:"onError"` // continue, stop, retry
}

// Shortcut represents a keyboard shortcut
type Shortcut struct {
	Key     string `yaml:"key"`
	Command string `yaml:"command"`
}

// ClusterConfig holds per-cluster configuration
type ClusterConfig struct {
	Name             string `yaml:"name"`
	Context          string `yaml:"context"`
	DefaultNamespace string `yaml:"defaultNamespace"`
	Color            string `yaml:"color"`
}

// Theme represents a color theme
type Theme struct {
	Name       string      `yaml:"name"`
	Colors     ThemeColors `yaml:"colors"`
	IsDark     bool        `yaml:"isDark"`
}

// ThemeColors holds the color values for a theme.
// All fields are optional (omitempty) to allow partial overrides.
type ThemeColors struct {
	Primary    string `yaml:"primary,omitempty"`
	Accent     string `yaml:"accent,omitempty"`
	Background string `yaml:"background,omitempty"`
	Surface    string `yaml:"surface,omitempty"`
	Text       string `yaml:"text,omitempty"`
	Muted      string `yaml:"muted,omitempty"`
	Border     string `yaml:"border,omitempty"`
	Highlight  string `yaml:"highlight,omitempty"`
	Success    string `yaml:"success,omitempty"`
	Warning    string `yaml:"warning,omitempty"`
	Error      string `yaml:"error,omitempty"`
	Info       string `yaml:"info,omitempty"`

	// Optional overrides for derived colors (normally computed from base colors).
	SelectionBg string `yaml:"selectionBg,omitempty"`
	SelectionFg string `yaml:"selectionFg,omitempty"`
	FrameBorder string `yaml:"frameBorder,omitempty"`
	SurfaceAlt  string `yaml:"surfaceAlt,omitempty"`
}
