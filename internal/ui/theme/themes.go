package theme

import "github.com/bijaya/kview/pkg/config"

// ThemeDefinition holds the 12 user-facing base colors for a theme,
// plus optional overrides for derived colors.
type ThemeDefinition struct {
	Background string
	Surface    string
	Text       string
	Muted      string
	Border     string
	Highlight  string
	Primary    string
	Accent     string
	Success    string
	Warning    string
	Error      string
	Info       string

	// Optional overrides for derived colors. When empty, these are
	// computed algorithmically from the base colors in Apply().
	SelectionBg string
	SelectionFg string
	FrameBorder string
	SurfaceAlt  string

	// Search highlight overrides
	SearchHighlightBg string
	SearchHighlightFg string

	// Delta row color overrides
	DeltaAdd    string
	DeltaModify string
	DeltaError  string
	DeltaDelete string
}

// ActiveThemeName tracks the currently active theme name.
var ActiveThemeName string

// BuiltinThemes maps theme names to their color definitions.
var BuiltinThemes = map[string]ThemeDefinition{
	"default": {
		Background: "#1B1B3A",
		Surface:    "#1A1A2E",
		Text:       "#E2E8F0",
		Muted:      "#64748B",
		Border:     "#3D3D5C",
		Highlight:  "#89B4FA",
		Primary:    "#7C3AED",
		Accent:     "#06B6D4",
		Success:    "#10B981",
		Warning:    "#F59E0B",
		Error:      "#EF4444",
		Info:       "#3B82F6",
	},
	"dracula": {
		Background: "#282A36",
		Surface:    "#21222C",
		Text:       "#F8F8F2",
		Muted:      "#6272A4",
		Border:     "#44475A",
		Highlight:  "#BD93F9",
		Primary:    "#BD93F9",
		Accent:     "#8BE9FD",
		Success:    "#50FA7B",
		Warning:    "#F1FA8C",
		Error:      "#FF5555",
		Info:       "#8BE9FD",
	},
	"catppuccin": {
		Background: "#1E1E2E",
		Surface:    "#181825",
		Text:       "#CDD6F4",
		Muted:      "#6C7086",
		Border:     "#45475A",
		Highlight:  "#89B4FA",
		Primary:    "#CBA6F7",
		Accent:     "#94E2D5",
		Success:    "#A6E3A1",
		Warning:    "#F9E2AF",
		Error:      "#F38BA8",
		Info:       "#89B4FA",
	},
	"tokyo-night": {
		Background: "#1A1B26",
		Surface:    "#16161E",
		Text:       "#C0CAF5",
		Muted:      "#565F89",
		Border:     "#3B4261",
		Highlight:  "#7AA2F7",
		Primary:    "#7AA2F7",
		Accent:     "#7DCFFF",
		Success:    "#9ECE6A",
		Warning:    "#E0AF68",
		Error:      "#F7768E",
		Info:       "#7AA2F7",
	},
	"nord": {
		Background: "#2E3440",
		Surface:    "#3B4252",
		Text:       "#ECEFF4",
		Muted:      "#4C566A",
		Border:     "#434C5E",
		Highlight:  "#81A1C1",
		Primary:    "#81A1C1",
		Accent:     "#88C0D0",
		Success:    "#A3BE8C",
		Warning:    "#EBCB8B",
		Error:      "#BF616A",
		Info:       "#81A1C1",
	},
	"gruvbox": {
		Background: "#282828",
		Surface:    "#1D2021",
		Text:       "#EBDBB2",
		Muted:      "#928374",
		Border:     "#3C3836",
		Highlight:  "#83A598",
		Primary:    "#D3869B",
		Accent:     "#8EC07C",
		Success:    "#B8BB26",
		Warning:    "#FABD2F",
		Error:      "#FB4934",
		Info:       "#83A598",
	},
	"solarized": {
		Background: "#002B36",
		Surface:    "#073642",
		Text:       "#839496",
		Muted:      "#586E75",
		Border:     "#2E4F5C",
		Highlight:  "#268BD2",
		Primary:    "#6C71C4",
		Accent:     "#2AA198",
		Success:    "#859900",
		Warning:    "#B58900",
		Error:      "#DC322F",
		Info:       "#268BD2",
	},
	"one-dark": {
		Background: "#282C34",
		Surface:    "#21252B",
		Text:       "#ABB2BF",
		Muted:      "#5C6370",
		Border:     "#3E4451",
		Highlight:  "#61AFEF",
		Primary:    "#C678DD",
		Accent:     "#56B6C2",
		Success:    "#98C379",
		Warning:    "#E5C07B",
		Error:      "#E06C75",
		Info:       "#61AFEF",
	},
	"monokai": {
		Background: "#2D2A2E",
		Surface:    "#221F22",
		Text:       "#FCFCFA",
		Muted:      "#727072",
		Border:     "#403E41",
		Highlight:  "#78DCE8",
		Primary:    "#AB9DF2",
		Accent:     "#78DCE8",
		Success:    "#A9DC76",
		Warning:    "#FFD866",
		Error:      "#FF6188",
		Info:       "#78DCE8",
	},
	"rose-pine": {
		Background: "#191724",
		Surface:    "#1F1D2E",
		Text:       "#E0DEF4",
		Muted:      "#6E6A86",
		Border:     "#26233A",
		Highlight:  "#C4A7E7",
		Primary:    "#C4A7E7",
		Accent:     "#9CCFD8",
		Success:    "#31748F",
		Warning:    "#F6C177",
		Error:      "#EB6F92",
		Info:       "#9CCFD8",
	},
	"kanagawa": {
		Background: "#1F1F28",
		Surface:    "#16161D",
		Text:       "#DCD7BA",
		Muted:      "#727169",
		Border:     "#2A2A37",
		Highlight:  "#7E9CD8",
		Primary:    "#957FB8",
		Accent:     "#7FB4CA",
		Success:    "#76946A",
		Warning:    "#DCA561",
		Error:      "#C34043",
		Info:       "#7E9CD8",
	},
	"everforest": {
		Background: "#2D353B",
		Surface:    "#232A2E",
		Text:       "#D3C6AA",
		Muted:      "#859289",
		Border:     "#475258",
		Highlight:  "#A7C080",
		Primary:    "#D699B6",
		Accent:     "#83C092",
		Success:    "#A7C080",
		Warning:    "#DBBC7F",
		Error:      "#E67E80",
		Info:       "#7FBBB3",
	},
	"palenight": {
		Background: "#292D3E",
		Surface:    "#1B1E2B",
		Text:       "#A6ACCD",
		Muted:      "#676E95",
		Border:     "#3A3F58",
		Highlight:  "#82AAFF",
		Primary:    "#C792EA",
		Accent:     "#89DDFF",
		Success:    "#C3E88D",
		Warning:    "#FFCB6B",
		Error:      "#F07178",
		Info:       "#82AAFF",
	},
	"ayu": {
		Background: "#0B0E14",
		Surface:    "#0D1017",
		Text:       "#BFBDB6",
		Muted:      "#565B66",
		Border:     "#1C212B",
		Highlight:  "#73B8FF",
		Primary:    "#D2A6FF",
		Accent:     "#95E6CB",
		Success:    "#AAD94C",
		Warning:    "#FFB454",
		Error:      "#D95757",
		Info:       "#73B8FF",
	},
	"horizon": {
		Background: "#1C1E26",
		Surface:    "#16161C",
		Text:       "#D5D8DA",
		Muted:      "#6C6F93",
		Border:     "#2E303E",
		Highlight:  "#E95678",
		Primary:    "#B877DB",
		Accent:     "#25B0BC",
		Success:    "#09F7A0",
		Warning:    "#FAB795",
		Error:      "#E95678",
		Info:       "#26BBD9",
	},
	"midnight": {
		Background: "#000000",
		Surface:    "#0D0D0D",
		Text:       "#CCCCCC",
		Muted:      "#505050",
		Border:     "#1A1A1A",
		Highlight:  "#6CB6FF",
		Primary:    "#D2A8FF",
		Accent:     "#7EE787",
		Success:    "#56D364",
		Warning:    "#E3B341",
		Error:      "#F85149",
		Info:       "#6CB6FF",
	},
	"night-owl": {
		Background: "#011627",
		Surface:    "#0B2942",
		Text:       "#D6DEEB",
		Muted:      "#637777",
		Border:     "#1D3B53",
		Highlight:  "#82AAFF",
		Primary:    "#C792EA",
		Accent:     "#7FDBCA",
		Success:    "#ADDB67",
		Warning:    "#ECC48D",
		Error:      "#EF5350",
		Info:       "#82AAFF",
	},
	"synthwave": {
		Background: "#262335",
		Surface:    "#1E1A31",
		Text:       "#E0DEF4",
		Muted:      "#848BBD",
		Border:     "#34294F",
		Highlight:  "#36F9F6",
		Primary:    "#FF7EDB",
		Accent:     "#36F9F6",
		Success:    "#72F1B8",
		Warning:    "#FEDE5D",
		Error:      "#FE4450",
		Info:       "#36F9F6",
	},
	"oxocarbon": {
		Background: "#161616",
		Surface:    "#0E0E0E",
		Text:       "#F2F4F8",
		Muted:      "#525252",
		Border:     "#262626",
		Highlight:  "#78A9FF",
		Primary:    "#BE95FF",
		Accent:     "#08BDBA",
		Success:    "#42BE65",
		Warning:    "#F1C21B",
		Error:      "#EE5396",
		Info:       "#78A9FF",
	},
	"github-dark": {
		Background: "#0D1117",
		Surface:    "#161B22",
		Text:       "#E6EDF3",
		Muted:      "#7D8590",
		Border:     "#30363D",
		Highlight:  "#58A6FF",
		Primary:    "#BC8CFF",
		Accent:     "#79C0FF",
		Success:    "#3FB950",
		Warning:    "#D29922",
		Error:      "#F85149",
		Info:       "#58A6FF",
	},
	"github-light": {
		Background:  "#FFFFFF",
		Surface:     "#F6F8FA",
		Text:        "#1F2328",
		Muted:       "#656D76",
		Border:      "#D0D7DE",
		Highlight:   "#0969DA",
		Primary:     "#8250DF",
		Accent:      "#0969DA",
		Success:     "#1A7F37",
		Warning:     "#9A6700",
		Error:       "#CF222E",
		Info:        "#0969DA",
		SelectionBg: "#0969DA",
		SelectionFg: "#FFFFFF",
	},
}

// ThemeNames provides a stable, ordered list of all built-in theme names.
// Used by :themes command and anywhere deterministic iteration is needed.
var ThemeNames = []string{
	"default",
	"dracula",
	"catppuccin",
	"tokyo-night",
	"nord",
	"gruvbox",
	"solarized",
	"one-dark",
	"monokai",
	"rose-pine",
	"kanagawa",
	"everforest",
	"palenight",
	"ayu",
	"horizon",
	"midnight",
	"night-owl",
	"synthwave",
	"oxocarbon",
	"github-dark",
	"github-light",
}

// ResolveTheme looks up a built-in theme by name and overlays any non-empty
// override fields. Unknown names fall back to "default".
// Returns the resolved theme and whether the name matched a built-in theme.
func ResolveTheme(name string, overrides ThemeDefinition) (ThemeDefinition, bool) {
	td, ok := BuiltinThemes[name]
	if !ok {
		td = BuiltinThemes["default"]
	}

	if overrides.Background != "" {
		td.Background = overrides.Background
	}
	if overrides.Surface != "" {
		td.Surface = overrides.Surface
	}
	if overrides.Text != "" {
		td.Text = overrides.Text
	}
	if overrides.Muted != "" {
		td.Muted = overrides.Muted
	}
	if overrides.Border != "" {
		td.Border = overrides.Border
	}
	if overrides.Highlight != "" {
		td.Highlight = overrides.Highlight
	}
	if overrides.Primary != "" {
		td.Primary = overrides.Primary
	}
	if overrides.Accent != "" {
		td.Accent = overrides.Accent
	}
	if overrides.Success != "" {
		td.Success = overrides.Success
	}
	if overrides.Warning != "" {
		td.Warning = overrides.Warning
	}
	if overrides.Error != "" {
		td.Error = overrides.Error
	}
	if overrides.Info != "" {
		td.Info = overrides.Info
	}
	if overrides.SelectionBg != "" {
		td.SelectionBg = overrides.SelectionBg
	}
	if overrides.SelectionFg != "" {
		td.SelectionFg = overrides.SelectionFg
	}
	if overrides.FrameBorder != "" {
		td.FrameBorder = overrides.FrameBorder
	}
	if overrides.SurfaceAlt != "" {
		td.SurfaceAlt = overrides.SurfaceAlt
	}
	if overrides.SearchHighlightBg != "" {
		td.SearchHighlightBg = overrides.SearchHighlightBg
	}
	if overrides.SearchHighlightFg != "" {
		td.SearchHighlightFg = overrides.SearchHighlightFg
	}
	if overrides.DeltaAdd != "" {
		td.DeltaAdd = overrides.DeltaAdd
	}
	if overrides.DeltaModify != "" {
		td.DeltaModify = overrides.DeltaModify
	}
	if overrides.DeltaError != "" {
		td.DeltaError = overrides.DeltaError
	}
	if overrides.DeltaDelete != "" {
		td.DeltaDelete = overrides.DeltaDelete
	}

	return td, ok
}

// ThemeDefinitionFromConfig converts config.ThemeColors to a ThemeDefinition.
// Empty fields are left as zero values so ResolveTheme treats them as no-override.
func ThemeDefinitionFromConfig(tc config.ThemeColors) ThemeDefinition {
	return ThemeDefinition{
		Background:  tc.Background,
		Surface:     tc.Surface,
		Text:        tc.Text,
		Muted:       tc.Muted,
		Border:      tc.Border,
		Highlight:   tc.Highlight,
		Primary:     tc.Primary,
		Accent:      tc.Accent,
		Success:     tc.Success,
		Warning:     tc.Warning,
		Error:       tc.Error,
		Info:        tc.Info,
		SelectionBg: tc.SelectionBg,
		SelectionFg: tc.SelectionFg,
		FrameBorder:       tc.FrameBorder,
		SurfaceAlt:        tc.SurfaceAlt,
		SearchHighlightBg: tc.SearchHighlightBg,
		SearchHighlightFg: tc.SearchHighlightFg,
		DeltaAdd:          tc.DeltaAdd,
		DeltaModify:       tc.DeltaModify,
		DeltaError:        tc.DeltaError,
		DeltaDelete:       tc.DeltaDelete,
	}
}
