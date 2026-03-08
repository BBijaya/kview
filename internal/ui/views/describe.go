package views

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// DescribeLoadedMsg is sent when description is loaded
type DescribeLoadedMsg struct {
	Description    string
	RawDescription string
	Err            error
}

// DescribeView displays detailed information about a resource
type DescribeView struct {
	BaseView
	viewport       viewport.Model
	client         k8s.Client
	kind           string
	namespace      string
	name           string
	description    string
	rawDescription string
	loading        bool
	err            error
	spinner        *components.Spinner
	search         ViewportSearch
}

// NewDescribeView creates a new describe view
func NewDescribeView(client k8s.Client) *DescribeView {
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))
	vp.Style = theme.Styles.Base
	ConfigureHighlightStyles(&vp)

	return &DescribeView{
		viewport: vp,
		client:   client,
		spinner:  components.NewSpinner(),
	}
}

// SetClient sets a new k8s client
func (v *DescribeView) SetClient(client k8s.Client) {
	v.client = client
}

// SetResource sets the resource to describe
func (v *DescribeView) SetResource(kind, namespace, name string) {
	v.kind = kind
	v.namespace = namespace
	v.name = name
}

// Init initializes the view
func (v *DescribeView) Init() tea.Cmd {
	if v.name == "" {
		return nil
	}
	return v.Refresh()
}

// Update handles messages
func (v *DescribeView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case DescribeLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.rawDescription = msg.RawDescription
			v.description = msg.Description
			if v.search.HasSearch() {
				matches := v.search.RecomputeMatches(v.rawDescription)
				v.viewport.SetContent(v.rawDescription)
				v.viewport.SetHighlights(matches)
			} else {
				v.viewport.SetContent(v.description)
			}
			v.viewport.GotoTop()
		}

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			return v, func() tea.Msg {
				return GoBackMsg{}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().LogSearchNext):
			if v.search.HasSearch() {
				v.viewport.HighlightNext()
			}
			return v, nil

		case key.Matches(msg, theme.DefaultKeyMap().LogSearchPrev):
			if v.search.HasSearch() {
				v.viewport.HighlightPrevious()
			}
			return v, nil

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
func (v *DescribeView) View() string {
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
	header := theme.Styles.PanelTitle.Render(fmt.Sprintf("Describe: %s/%s/%s", v.kind, v.namespace, v.name))

	// Search status
	if status := v.search.StatusText(); status != "" {
		header += bgStyle.Render(" ") + theme.Styles.StatusPending.Render(status)
	}

	// Pad header to full width
	headerWidth := lipgloss.Width(header)
	if headerWidth < v.width {
		header += bgStyle.Render(strings.Repeat(" ", v.width-headerWidth))
	}

	// Footer with help
	footer := theme.Styles.Help.Render("↑↓/←→ scroll • g/G top/bottom • / search • n/N next/prev • esc back")

	// Pad footer to full width
	footerWidth := lipgloss.Width(footer)
	if footerWidth < v.width {
		footer += bgStyle.Render(strings.Repeat(" ", v.width-footerWidth))
	}

	return header + "\n" + v.viewport.View() + "\n" + footer
}

// Name returns the view name
func (v *DescribeView) Name() string {
	return "Describe"
}

// ShortHelp returns keybindings for help
func (v *DescribeView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Escape,
	}
}

// SetSize sets the view dimensions
func (v *DescribeView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	v.viewport.SetWidth(width)
	v.viewport.SetHeight(height - 3) // Account for header and footer
}

// IsLoading returns whether the view is currently loading data
func (v *DescribeView) IsLoading() bool {
	return v.loading
}

// Content returns the current describe text (plain, without ANSI codes)
func (v *DescribeView) Content() string {
	return v.rawDescription
}

// ApplySearch implements ViewportSearcher.
func (v *DescribeView) ApplySearch(pattern string) {
	if pattern == "" {
		v.ClearSearch()
		return
	}
	matches := v.search.ApplySearch(pattern, v.rawDescription)
	v.viewport.SetContent(v.rawDescription)
	v.viewport.SetHighlights(matches)
}

// ActiveSearchPattern implements ViewportSearcher.
func (v *DescribeView) ActiveSearchPattern() string {
	return v.search.ActivePattern()
}

// ClearSearch implements ViewportSearcher.
func (v *DescribeView) ClearSearch() {
	v.search.Clear()
	v.viewport.ClearHighlights()
	v.viewport.SetContent(v.description)
}

// Refresh refreshes the resource description
func (v *DescribeView) Refresh() tea.Cmd {
	if v.name == "" {
		return nil
	}

	v.loading = true
	v.spinner.SetMessage("Loading resource details...")
	cmds := []tea.Cmd{v.spinner.Show()}

	cmds = append(cmds, func() tea.Msg {
		resource, err := v.client.Get(context.Background(), v.kind, v.namespace, v.name)
		if err != nil {
			return DescribeLoadedMsg{Err: err}
		}

		raw := formatResource(resource, false)
		highlighted := formatResource(resource, true)
		return DescribeLoadedMsg{Description: highlighted, RawDescription: raw}
	})

	return tea.Batch(cmds...)
}

// formatResource formats a resource for display.
// When highlight is true, lipgloss styles are applied to metadata and
// chroma syntax highlighting is applied to JSON spec/status blocks.
func formatResource(r *k8s.Resource, highlight bool) string {
	var b strings.Builder

	// Style closures — identity when not highlighting
	label := func(s string) string { return s }
	value := func(s string) string { return s }
	section := func(s string) string { return s }
	if highlight {
		label = func(s string) string { return theme.Styles.InfoLabel.Render(s) }
		value = func(s string) string { return theme.Styles.InfoValue.Render(s) }
		section = func(s string) string { return theme.Styles.PanelTitle.Render(s) }
	}

	// Basic info
	b.WriteString(label("Name:         ") + value(r.Name) + "\n")
	b.WriteString(label("Namespace:    ") + value(r.Namespace) + "\n")
	b.WriteString(label("Kind:         ") + value(r.Kind) + "\n")
	b.WriteString(label("API Version:  ") + value(r.APIVersion) + "\n")
	b.WriteString(label("UID:          ") + value(r.UID) + "\n")
	b.WriteString("\n")

	// Labels
	b.WriteString(section("Labels:") + "\n")
	if len(r.Labels) == 0 {
		b.WriteString("  <none>\n")
	} else {
		for k, v := range r.Labels {
			b.WriteString("  " + label(k) + "=" + value(v) + "\n")
		}
	}
	b.WriteString("\n")

	// Annotations
	b.WriteString(section("Annotations:") + "\n")
	if len(r.Annotations) == 0 {
		b.WriteString("  <none>\n")
	} else {
		for k, v := range r.Annotations {
			// Truncate long annotations
			if len(v) > 100 {
				v = v[:100] + "..."
			}
			b.WriteString("  " + label(k) + "=" + value(v) + "\n")
		}
	}
	b.WriteString("\n")

	// Owner References
	if len(r.OwnerRefs) > 0 {
		b.WriteString(section("Owner References:") + "\n")
		for _, ref := range r.OwnerRefs {
			b.WriteString("  " + label(ref.Kind+": ") + value(ref.Name) + " (UID: " + value(ref.UID) + ")\n")
		}
		b.WriteString("\n")
	}

	// Conditions
	if len(r.Conditions) > 0 {
		b.WriteString(section("Conditions:") + "\n")
		for _, c := range r.Conditions {
			b.WriteString("  " + label("Type: ") + value(c.Type) + "\n")
			b.WriteString("    " + label("Status: ") + value(c.Status) + "\n")
			if c.Reason != "" {
				b.WriteString("    " + label("Reason: ") + value(c.Reason) + "\n")
			}
			if c.Message != "" {
				b.WriteString("    " + label("Message: ") + value(c.Message) + "\n")
			}
		}
		b.WriteString("\n")
	}

	// Raw JSON (formatted)
	if r.Raw != nil {
		b.WriteString(section("Spec:") + "\n")
		spec, ok := r.Raw.Object["spec"]
		if ok {
			jsonBytes, err := json.MarshalIndent(spec, "  ", "  ")
			if err == nil {
				rendered := "  " + string(jsonBytes)
				if highlight {
					rendered = HighlightJSON(rendered)
				}
				b.WriteString(rendered)
			} else {
				b.WriteString("  (failed to render spec)")
			}
		}
		b.WriteString("\n\n")

		b.WriteString(section("Status:") + "\n")
		status, ok := r.Raw.Object["status"]
		if ok {
			jsonBytes, err := json.MarshalIndent(status, "  ", "  ")
			if err == nil {
				rendered := "  " + string(jsonBytes)
				if highlight {
					rendered = HighlightJSON(rendered)
				}
				b.WriteString(rendered)
			} else {
				b.WriteString("  (failed to render status)")
			}
		}
	}

	return b.String()
}
