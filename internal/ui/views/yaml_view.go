package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// YAMLLoadedMsg is sent when YAML content is loaded
type YAMLLoadedMsg struct {
	Content    string
	RawContent string
	Err        error
}

// GoBackMsg requests going back to the previous view
type GoBackMsg struct{}

// YAMLView displays raw YAML for a resource
type YAMLView struct {
	BaseView
	viewport   viewport.Model
	client     k8s.Client
	kind       string
	name       string
	content    string
	rawContent string
	loading    bool
	err        error
	spinner    *components.Spinner
}

// NewYAMLView creates a new YAML view
func NewYAMLView(client k8s.Client) *YAMLView {
	vp := viewport.New(80, 20)
	vp.Style = theme.Styles.Base

	return &YAMLView{
		viewport: vp,
		client:   client,
		spinner:  components.NewSpinner(),
	}
}

// SetResource sets the resource to display YAML for
func (v *YAMLView) SetResource(kind, namespace, name string) {
	v.kind = kind
	v.namespace = namespace
	v.name = name
}

// SetClient sets a new k8s client
func (v *YAMLView) SetClient(client k8s.Client) {
	v.client = client
}

// Init initializes the view
func (v *YAMLView) Init() tea.Cmd {
	if v.name == "" {
		return nil
	}
	return v.Refresh()
}

// Update handles messages
func (v *YAMLView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case YAMLLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.rawContent = msg.RawContent
			v.content = msg.Content
			v.viewport.SetContent(v.content)
			v.viewport.GotoTop()
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			return v, func() tea.Msg {
				return GoBackMsg{}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case msg.String() == "G":
			v.viewport.GotoBottom()

		case msg.String() == "g":
			v.viewport.GotoTop()

		default:
			// Let viewport handle scrolling keys (up/down/pgup/pgdn)
			var cmd tea.Cmd
			v.viewport, cmd = v.viewport.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return v, tea.Batch(cmds...)
		}
	}

	// Update spinner
	if v.loading {
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update viewport for non-key messages
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// View renders the view
func (v *YAMLView) View() string {
	if v.name == "" {
		return theme.Styles.StatusUnknown.Render("No resource selected. Press Escape to go back.")
	}

	if v.loading {
		return v.spinner.ViewCentered(v.width, v.height)
	}

	if v.err != nil {
		return theme.Styles.StatusError.Render("Error: " + v.err.Error())
	}

	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	// Header
	header := theme.Styles.PanelTitle.Render(fmt.Sprintf("YAML: %s/%s/%s", v.kind, v.namespace, v.name))

	// Pad header to full width
	headerWidth := lipgloss.Width(header)
	if headerWidth < v.width {
		header += bgStyle.Render(strings.Repeat(" ", v.width-headerWidth))
	}

	// Footer with help
	footer := theme.Styles.Help.Render("↑↓/pgup/pgdn scroll • g/G top/bottom • esc back")

	// Pad footer to full width
	footerWidth := lipgloss.Width(footer)
	if footerWidth < v.width {
		footer += bgStyle.Render(strings.Repeat(" ", v.width-footerWidth))
	}

	return header + "\n" + v.viewport.View() + "\n" + footer
}

// Name returns the view name
func (v *YAMLView) Name() string {
	return "YAML"
}

// ShortHelp returns keybindings for help
func (v *YAMLView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Escape,
	}
}

// SetSize sets the view dimensions
func (v *YAMLView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	v.viewport.Width = width
	v.viewport.Height = height - 3 // Account for header and footer
}

// IsLoading returns whether the view is currently loading data
func (v *YAMLView) IsLoading() bool {
	return v.loading
}

// Content returns the current YAML text (plain, without ANSI codes)
func (v *YAMLView) Content() string {
	return v.rawContent
}

// Refresh fetches the resource and marshals to YAML
func (v *YAMLView) Refresh() tea.Cmd {
	if v.name == "" {
		return nil
	}

	v.loading = true
	v.spinner.SetMessage("Loading YAML...")
	cmds := []tea.Cmd{v.spinner.Show()}

	cmds = append(cmds, func() tea.Msg {
		resource, err := v.client.Get(context.Background(), v.kind, v.namespace, v.name)
		if err != nil {
			return YAMLLoadedMsg{Err: err}
		}

		// Marshal the raw unstructured object to YAML
		var data interface{}
		if resource.Raw != nil {
			data = resource.Raw.Object
		} else {
			// Fallback: build a basic map from resource fields
			data = map[string]interface{}{
				"apiVersion": resource.APIVersion,
				"kind":       resource.Kind,
				"metadata": map[string]interface{}{
					"name":        resource.Name,
					"namespace":   resource.Namespace,
					"uid":         resource.UID,
					"labels":      resource.Labels,
					"annotations": resource.Annotations,
				},
			}
		}

		yamlBytes, err := yaml.Marshal(data)
		if err != nil {
			return YAMLLoadedMsg{Err: fmt.Errorf("failed to marshal YAML: %w", err)}
		}

		raw := string(yamlBytes)
		highlighted := HighlightYAML(raw)
		return YAMLLoadedMsg{Content: highlighted, RawContent: raw}
	})

	return tea.Batch(cmds...)
}
