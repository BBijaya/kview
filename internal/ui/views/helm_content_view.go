package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// HelmContentLoadedMsg is sent when Helm content (values or manifest) is loaded
type HelmContentLoadedMsg struct {
	Content    string
	RawContent string
	Err        error
}

// OpenHelmContentMsg requests opening a Helm content view (values or manifest)
type OpenHelmContentMsg struct {
	Mode        HelmContentMode
	ReleaseName string
	Namespace   string
	Revision    int
}

// HelmContentMode defines what kind of Helm content to display
type HelmContentMode int

const (
	HelmContentValues   HelmContentMode = iota
	HelmContentManifest
)

// HelmContentView displays Helm release values or manifest in a viewport
type HelmContentView struct {
	BaseView
	viewport    viewport.Model
	client      k8s.Client
	mode        HelmContentMode
	releaseName string
	revision    int
	content     string
	rawContent  string
	loading     bool
	err         error
	spinner     *components.Spinner
}

// NewHelmContentView creates a new Helm content view
func NewHelmContentView(client k8s.Client, mode HelmContentMode) *HelmContentView {
	vp := viewport.New(80, 20)
	vp.Style = theme.Styles.Base

	v := &HelmContentView{
		viewport: vp,
		client:   client,
		mode:     mode,
		spinner:  components.NewSpinner(),
	}

	switch mode {
	case HelmContentValues:
		v.spinner.SetMessage("Loading Helm values...")
	case HelmContentManifest:
		v.spinner.SetMessage("Loading Helm manifest...")
	}

	return v
}

// SetRelease configures the view for a specific release and revision.
func (v *HelmContentView) SetRelease(namespace, releaseName string, revision int) {
	v.namespace = namespace
	v.releaseName = releaseName
	v.revision = revision
}

// Init initializes the view
func (v *HelmContentView) Init() tea.Cmd {
	if v.releaseName == "" {
		return nil
	}
	return v.Refresh()
}

// Update handles messages
func (v *HelmContentView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case HelmContentLoadedMsg:
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
			var cmd tea.Cmd
			v.viewport, cmd = v.viewport.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return v, tea.Batch(cmds...)
		}
	}

	if v.loading {
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// View renders the view
func (v *HelmContentView) View() string {
	if v.releaseName == "" {
		return theme.Styles.StatusUnknown.Render("No release selected. Press Escape to go back.")
	}

	if v.loading {
		return v.spinner.ViewCentered(v.width, v.height)
	}

	if v.err != nil {
		return theme.Styles.StatusError.Render("Error: " + v.err.Error())
	}

	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	// Header
	var title string
	switch v.mode {
	case HelmContentValues:
		title = fmt.Sprintf("Values: %s (rev %d)", v.releaseName, v.revision)
	case HelmContentManifest:
		title = fmt.Sprintf("Manifest: %s (rev %d)", v.releaseName, v.revision)
	}
	header := theme.Styles.PanelTitle.Render(title)

	headerWidth := lipgloss.Width(header)
	if headerWidth < v.width {
		header += bgStyle.Render(strings.Repeat(" ", v.width-headerWidth))
	}

	// Footer
	footer := theme.Styles.Help.Render("↑↓/pgup/pgdn scroll • g/G top/bottom • esc back")
	footerWidth := lipgloss.Width(footer)
	if footerWidth < v.width {
		footer += bgStyle.Render(strings.Repeat(" ", v.width-footerWidth))
	}

	return header + "\n" + v.viewport.View() + "\n" + footer
}

// Name returns the view name
func (v *HelmContentView) Name() string {
	switch v.mode {
	case HelmContentValues:
		return "Helm Values"
	case HelmContentManifest:
		return "Helm Manifest"
	default:
		return "Helm Content"
	}
}

func (v *HelmContentView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Escape,
	}
}

func (v *HelmContentView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	v.viewport.Width = width
	v.viewport.Height = height - 3 // header + footer + divider
}

func (v *HelmContentView) IsLoading() bool { return v.loading }
func (v *HelmContentView) Content() string { return v.rawContent }

func (v *HelmContentView) Refresh() tea.Cmd {
	if v.releaseName == "" {
		return nil
	}

	v.loading = true
	ns := v.namespace
	name := v.releaseName
	rev := v.revision
	mode := v.mode

	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			var content string
			var err error
			switch mode {
			case HelmContentValues:
				content, err = v.client.GetHelmValues(context.Background(), ns, name, rev)
			case HelmContentManifest:
				content, err = v.client.GetHelmManifest(context.Background(), ns, name, rev)
			}
			if err != nil {
				return HelmContentLoadedMsg{Err: err}
			}
			highlighted := HighlightYAML(content)
			return HelmContentLoadedMsg{Content: highlighted, RawContent: content}
		},
	)
}

func (v *HelmContentView) SetClient(client k8s.Client) { v.client = client }
