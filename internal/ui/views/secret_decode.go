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

// DecodeSecretMsg requests opening the decode view for a secret
type DecodeSecretMsg struct {
	Namespace string
	Name      string
}

// SecretDecodedMsg is sent when decoded secret data is loaded
type SecretDecodedMsg struct {
	Content    string
	RawContent string
	Err        error
}

// SecretDecodeView displays base64-decoded secret data
type SecretDecodeView struct {
	BaseView
	viewport   viewport.Model
	client     k8s.Client
	name       string
	content    string
	rawContent string
	loading    bool
	err        error
	spinner    *components.Spinner
}

// NewSecretDecodeView creates a new secret decode view
func NewSecretDecodeView(client k8s.Client) *SecretDecodeView {
	vp := viewport.New(80, 20)
	vp.Style = theme.Styles.Base

	return &SecretDecodeView{
		viewport: vp,
		client:   client,
		spinner:  components.NewSpinner(),
	}
}

// SetResource sets the secret to decode
func (v *SecretDecodeView) SetResource(namespace, name string) {
	v.namespace = namespace
	v.name = name
}

// Init initializes the view
func (v *SecretDecodeView) Init() tea.Cmd {
	if v.name == "" {
		return nil
	}
	return v.Refresh()
}

// Update handles messages
func (v *SecretDecodeView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case SecretDecodedMsg:
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
func (v *SecretDecodeView) View() string {
	if v.name == "" {
		return theme.Styles.StatusUnknown.Render("No secret selected. Press Escape to go back.")
	}

	if v.loading {
		return v.spinner.ViewCentered(v.width, v.height)
	}

	if v.err != nil {
		return theme.Styles.StatusError.Render("Error: " + v.err.Error())
	}

	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	header := theme.Styles.PanelTitle.Render(fmt.Sprintf("Decode: %s/%s", v.namespace, v.name))
	headerWidth := lipgloss.Width(header)
	if headerWidth < v.width {
		header += bgStyle.Render(strings.Repeat(" ", v.width-headerWidth))
	}

	footer := theme.Styles.Help.Render("↑↓/pgup/pgdn scroll • g/G top/bottom • esc back")
	footerWidth := lipgloss.Width(footer)
	if footerWidth < v.width {
		footer += bgStyle.Render(strings.Repeat(" ", v.width-footerWidth))
	}

	return header + "\n" + v.viewport.View() + "\n" + footer
}

func (v *SecretDecodeView) Name() string    { return "Secret Decoded" }
func (v *SecretDecodeView) Content() string { return v.rawContent }

func (v *SecretDecodeView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Escape,
	}
}

func (v *SecretDecodeView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	v.viewport.Width = width
	v.viewport.Height = height - 3
}

func (v *SecretDecodeView) IsLoading() bool { return v.loading }

func (v *SecretDecodeView) Refresh() tea.Cmd {
	if v.name == "" {
		return nil
	}

	v.loading = true
	v.spinner.SetMessage("Decoding secret...")
	ns := v.namespace
	name := v.name

	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			content, err := v.client.GetSecretDecoded(context.Background(), ns, name)
			if err != nil {
				return SecretDecodedMsg{Err: err}
			}
			highlighted := highlightSecretContent(content)
			return SecretDecodedMsg{Content: highlighted, RawContent: content}
		},
	)
}

func (v *SecretDecodeView) SetClient(client k8s.Client) { v.client = client }
