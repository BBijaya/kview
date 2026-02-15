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

// ThemeColors holds the color values for a theme
type ThemeColors struct {
	Primary    string `yaml:"primary"`
	Secondary  string `yaml:"secondary"`
	Accent     string `yaml:"accent"`
	Background string `yaml:"background"`
	Surface    string `yaml:"surface"`
	Text       string `yaml:"text"`
	TextMuted  string `yaml:"textMuted"`
	Border     string `yaml:"border"`
	Success    string `yaml:"success"`
	Warning    string `yaml:"warning"`
	Error      string `yaml:"error"`
	Info       string `yaml:"info"`
}
