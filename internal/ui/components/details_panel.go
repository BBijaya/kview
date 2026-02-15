package components

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/theme"
)

// DetailsPanelLoadedMsg is sent when details are loaded
type DetailsPanelLoadedMsg struct {
	Content string
	Err     error
}

// DetailsPanel shows selected resource details in a side panel
type DetailsPanel struct {
	visible  bool
	loading  bool
	width    int
	height   int
	viewport viewport.Model
	content  string
	err      error

	// Current resource
	kind      string
	namespace string
	name      string

	// Spinner for loading state
	spinner *Spinner

	// Client for fetching resources
	client k8s.Client
}

// NewDetailsPanel creates a new details panel
func NewDetailsPanel(client k8s.Client) *DetailsPanel {
	vp := viewport.New(40, 20)
	vp.Style = theme.Styles.Base

	return &DetailsPanel{
		visible:  false,
		loading:  false,
		viewport: vp,
		spinner:  NewSpinner(),
		client:   client,
	}
}

// Show makes the panel visible
func (p *DetailsPanel) Show() {
	p.visible = true
}

// Hide hides the panel
func (p *DetailsPanel) Hide() {
	p.visible = false
}

// Toggle toggles the panel visibility
func (p *DetailsPanel) Toggle() {
	p.visible = !p.visible
}

// IsVisible returns whether the panel is visible
func (p *DetailsPanel) IsVisible() bool {
	return p.visible
}

// SetSize sets the panel dimensions
func (p *DetailsPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.viewport.Width = width - 4  // Account for border
	p.viewport.Height = height - 4 // Account for header and border
}

// SetClient sets the k8s client
func (p *DetailsPanel) SetClient(client k8s.Client) {
	p.client = client
}

// SetResource sets the resource to display and triggers a refresh
func (p *DetailsPanel) SetResource(kind, namespace, name string) tea.Cmd {
	// Skip if same resource
	if p.kind == kind && p.namespace == namespace && p.name == name {
		return nil
	}

	p.kind = kind
	p.namespace = namespace
	p.name = name

	if name == "" {
		p.content = ""
		p.viewport.SetContent("")
		return nil
	}

	return p.Refresh()
}

// Refresh reloads the resource details
func (p *DetailsPanel) Refresh() tea.Cmd {
	if p.name == "" || p.client == nil {
		return nil
	}

	p.loading = true
	kind := p.kind
	namespace := p.namespace
	name := p.name

	return tea.Batch(
		p.spinner.Show(),
		func() tea.Msg {
			resource, err := p.client.Get(context.Background(), kind, namespace, name)
			if err != nil {
				return DetailsPanelLoadedMsg{Err: err}
			}

			content := formatResourceDetails(resource)
			return DetailsPanelLoadedMsg{Content: content}
		},
	)
}

// Update handles messages
func (p *DetailsPanel) Update(msg tea.Msg) (*DetailsPanel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case DetailsPanelLoadedMsg:
		p.loading = false
		p.spinner.Hide()
		if msg.Err != nil {
			p.err = msg.Err
			p.content = ""
		} else {
			p.err = nil
			p.content = msg.Content
			p.viewport.SetContent(p.content)
			p.viewport.GotoTop()
		}
		return p, nil

	case tea.KeyMsg:
		if !p.visible {
			return p, nil
		}
		// Let viewport handle scrolling
		var cmd tea.Cmd
		p.viewport, cmd = p.viewport.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update spinner
	if p.loading {
		var cmd tea.Cmd
		p.spinner, cmd = p.spinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return p, tea.Batch(cmds...)
}

// View renders the panel
func (p *DetailsPanel) View() string {
	if !p.visible {
		return ""
	}

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorHighlight).
		Bold(true)

	var title string
	if p.name != "" {
		title = titleStyle.Render(fmt.Sprintf("Details: %s", p.name))
	} else {
		title = titleStyle.Render("Details")
	}

	// Content
	var content string
	if p.loading {
		content = p.spinner.ViewCentered(p.width-4, p.height-6)
	} else if p.err != nil {
		content = theme.Styles.StatusError.Render("Error: " + p.err.Error())
	} else if p.name == "" {
		content = theme.Styles.StatusUnknown.Render("No resource selected")
	} else {
		content = p.viewport.View()
	}

	// Build panel
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder).
		Width(p.width).
		Height(p.height)

	innerContent := title + "\n" + strings.Repeat("─", p.width-4) + "\n" + content

	return panelStyle.Render(innerContent)
}

// formatResourceDetails formats a resource for display in the details panel
func formatResourceDetails(r *k8s.Resource) string {
	var b strings.Builder

	// Header style
	headerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(theme.ColorHighlight)

	valueStyle := lipgloss.NewStyle().
		Foreground(theme.ColorText)

	mutedStyle := lipgloss.NewStyle().
		Foreground(theme.ColorMuted)

	// Basic info
	b.WriteString(headerStyle.Render("Resource Info"))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("Name:       ") + valueStyle.Render(r.Name) + "\n")
	b.WriteString(keyStyle.Render("Namespace:  ") + valueStyle.Render(r.Namespace) + "\n")
	b.WriteString(keyStyle.Render("Kind:       ") + valueStyle.Render(r.Kind) + "\n")
	b.WriteString(keyStyle.Render("API:        ") + valueStyle.Render(r.APIVersion) + "\n")
	b.WriteString(keyStyle.Render("UID:        ") + mutedStyle.Render(theme.TruncateString(r.UID, 30)) + "\n")
	b.WriteString("\n")

	// Labels
	b.WriteString(headerStyle.Render("Labels"))
	b.WriteString("\n")
	if len(r.Labels) == 0 {
		b.WriteString(mutedStyle.Render("  <none>") + "\n")
	} else {
		count := 0
		for k, v := range r.Labels {
			if count >= 5 {
				b.WriteString(mutedStyle.Render(fmt.Sprintf("  ... and %d more", len(r.Labels)-5)) + "\n")
				break
			}
			label := theme.TruncateString(k+"="+v, 35)
			b.WriteString("  " + valueStyle.Render(label) + "\n")
			count++
		}
	}
	b.WriteString("\n")

	// Conditions
	if len(r.Conditions) > 0 {
		b.WriteString(headerStyle.Render("Conditions"))
		b.WriteString("\n")
		for _, c := range r.Conditions {
			status := c.Status
			var statusStyle lipgloss.Style
			if status == "True" {
				statusStyle = theme.Styles.StatusHealthy
			} else if status == "False" {
				statusStyle = theme.Styles.StatusError
			} else {
				statusStyle = theme.Styles.StatusUnknown
			}
			b.WriteString("  " + keyStyle.Render(c.Type+": ") + statusStyle.Render(status) + "\n")
			if c.Reason != "" {
				b.WriteString("    " + mutedStyle.Render(c.Reason) + "\n")
			}
		}
		b.WriteString("\n")
	}

	// Owner References
	if len(r.OwnerRefs) > 0 {
		b.WriteString(headerStyle.Render("Owners"))
		b.WriteString("\n")
		for _, ref := range r.OwnerRefs {
			b.WriteString("  " + valueStyle.Render(ref.Kind+"/"+ref.Name) + "\n")
		}
		b.WriteString("\n")
	}

	// Spec summary (limited)
	if r.Raw != nil {
		if spec, ok := r.Raw.Object["spec"]; ok {
			b.WriteString(headerStyle.Render("Spec"))
			b.WriteString("\n")
			jsonBytes, err := json.MarshalIndent(spec, "  ", "  ")
			if err == nil {
				lines := strings.Split(string(jsonBytes), "\n")
				maxLines := 20
				for i, line := range lines {
					if i >= maxLines {
						b.WriteString(mutedStyle.Render("  ... (truncated)") + "\n")
						break
					}
					b.WriteString(valueStyle.Render(line) + "\n")
				}
			}
		}
	}

	return b.String()
}
