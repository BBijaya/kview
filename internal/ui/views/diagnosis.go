package views

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/bijaya/kview/internal/analyzer"
	"github.com/bijaya/kview/internal/analyzer/rules"
	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// DiagnosisLoadedMsg is sent when diagnoses are loaded
type DiagnosisLoadedMsg struct {
	Diagnoses []analyzer.Diagnosis
	Err       error
}

// DiagnosisView displays problem diagnoses
type DiagnosisView struct {
	BaseView
	table     *components.Table
	client    k8s.Client
	ruleSet   *rules.RuleSet
	diagnoses []analyzer.Diagnosis
	loading   bool
	err       error
	selected  int
}

// NewDiagnosisView creates a new diagnosis view
func NewDiagnosisView(client k8s.Client) *DiagnosisView {
	columns := []components.Column{
		{Title: "SEVERITY", Width: 10},
		{Title: "NAMESPACE", Width: 15},
		{Title: "RESOURCE", Width: 30, MinWidth: 20, Flexible: true},
		{Title: "PROBLEM", Width: 40, MinWidth: 20, Flexible: true},
	}

	return &DiagnosisView{
		table:   components.NewTable(columns),
		client:  client,
		ruleSet: rules.NewRuleSet(),
	}
}

// Init initializes the view
func (v *DiagnosisView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *DiagnosisView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case DiagnosisLoadedMsg:
		v.loading = false
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.diagnoses = msg.Diagnoses
			v.updateTable()
		}

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			return v, func() tea.Msg {
				return theme.SwitchViewMsg{View: theme.ViewPods}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Enter):
			// Navigate to the affected resource's describe view
			if v.selected < len(v.diagnoses) {
				d := v.diagnoses[v.selected]
				return v, func() tea.Msg {
					return OpenViewMsg{
						TargetView: theme.ViewDescribe,
						Kind:       d.ResourceKind,
						Resource:   "pods",
						Namespace:  d.Namespace,
						Name:       d.ResourceName,
						UID:        d.ResourceUID,
					}
				}
			}
		}
	}

	// Update table
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Track selection
	if row := v.table.SelectedRow(); row != nil {
		for i, d := range v.diagnoses {
			if d.ID == row.ID {
				v.selected = i
				break
			}
		}
	}

	return v, tea.Batch(cmds...)
}

// View renders the view
func (v *DiagnosisView) View() string {
	var b strings.Builder

	// Summary header
	summary := analyzer.Summarize(v.diagnoses)
	b.WriteString(v.renderSummary(summary))
	b.WriteString("\n\n")

	if v.loading {
		b.WriteString("Analyzing cluster...")
		return b.String()
	}

	if v.err != nil {
		b.WriteString(theme.Styles.StatusError.Render("Error: " + v.err.Error()))
		return b.String()
	}

	if len(v.diagnoses) == 0 {
		b.WriteString(theme.Styles.StatusHealthy.Render("No problems detected"))
		return b.String()
	}

	// Table of diagnoses
	b.WriteString(v.table.View())

	// Show details if a diagnosis is selected
	if v.selected < len(v.diagnoses) {
		b.WriteString("\n\n")
		b.WriteString(v.renderDiagnosisDetail(v.diagnoses[v.selected]))
	}

	return b.String()
}

func (v *DiagnosisView) renderSummary(s analyzer.DiagnosisSummary) string {
	parts := []string{
		theme.Styles.PanelTitle.Render("Cluster Health"),
		" │ ",
	}

	if s.Critical > 0 {
		parts = append(parts, theme.Styles.StatusError.Render(fmt.Sprintf("Critical: %d", s.Critical)))
		parts = append(parts, " ")
	}
	if s.Warning > 0 {
		parts = append(parts, theme.Styles.StatusWarning.Render(fmt.Sprintf("Warning: %d", s.Warning)))
		parts = append(parts, " ")
	}
	if s.Info > 0 {
		parts = append(parts, theme.Styles.StatusPending.Render(fmt.Sprintf("Info: %d", s.Info)))
	}
	if s.Total == 0 {
		parts = append(parts, theme.Styles.StatusHealthy.Render("All healthy"))
	}

	return strings.Join(parts, "")
}

func (v *DiagnosisView) renderDiagnosisDetail(d analyzer.Diagnosis) string {
	var b strings.Builder

	// Severity indicator
	severityStyle := theme.Styles.StatusUnknown
	switch d.Severity {
	case analyzer.SeverityCritical:
		severityStyle = theme.Styles.StatusError
	case analyzer.SeverityWarning:
		severityStyle = theme.Styles.StatusWarning
	case analyzer.SeverityInfo:
		severityStyle = theme.Styles.StatusPending
	}

	b.WriteString(severityStyle.Render(strings.ToUpper(string(d.Severity))))
	b.WriteString(" ")
	b.WriteString(theme.Styles.PanelTitle.Render(d.Problem))
	b.WriteString("\n\n")

	// Root cause
	b.WriteString(theme.Styles.Focused.Render("Root Cause:"))
	b.WriteString("\n")
	b.WriteString(d.RootCause)
	b.WriteString("\n\n")

	// Suggestions
	if len(d.Suggestions) > 0 {
		b.WriteString(theme.Styles.Focused.Render("Suggested Actions:"))
		b.WriteString("\n")
		for i, s := range d.Suggestions {
			risk := ""
			switch s.Risk {
			case "high":
				risk = theme.Styles.StatusError.Render("[high risk]")
			case "medium":
				risk = theme.Styles.StatusWarning.Render("[medium risk]")
			default:
				risk = theme.Styles.StatusHealthy.Render("[low risk]")
			}

			b.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, s.Title, risk))
			b.WriteString(fmt.Sprintf("   %s\n", s.Description))
			if s.Command != "" {
				b.WriteString(theme.Styles.Help.Render(fmt.Sprintf("   $ %s\n", s.Command)))
			}
		}
	}

	return b.String()
}

// Name returns the view name
func (v *DiagnosisView) Name() string {
	return "Diagnosis"
}

// ShortHelp returns keybindings for help
func (v *DiagnosisView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Refresh,
		theme.DefaultKeyMap().Escape,
	}
}

// SetSize sets the view dimensions
func (v *DiagnosisView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	// Reserve space for summary and details
	tableHeight := height / 2
	v.table.SetSize(width, tableHeight)
}

// ResetSelection resets the table cursor to the top
func (v *DiagnosisView) ResetSelection() {
	v.table.GotoTop()
}

// SetClient updates the Kubernetes client
func (v *DiagnosisView) SetClient(client k8s.Client) {
	v.client = client
}

// IsLoading returns whether the view is currently loading data
func (v *DiagnosisView) IsLoading() bool {
	return v.loading
}

// SelectedName returns the name of the currently selected resource
func (v *DiagnosisView) SelectedName() string {
	return v.table.SelectedValue(2)
}

// Refresh runs the analysis
func (v *DiagnosisView) Refresh() tea.Cmd {
	v.loading = true
	return func() tea.Msg {
		// Get pods
		pods, err := v.client.ListPods(context.Background(), v.namespace)
		if err != nil {
			return DiagnosisLoadedMsg{Err: err}
		}

		// Run analysis
		diagnoses := v.ruleSet.Analyze(nil, pods, nil)

		return DiagnosisLoadedMsg{Diagnoses: diagnoses}
	}
}

func (v *DiagnosisView) updateTable() {
	rows := make([]components.Row, len(v.diagnoses))
	for i, d := range v.diagnoses {
		severity := string(d.Severity)
		rows[i] = components.Row{
			ID: d.ID,
			Values: []string{
				strings.ToUpper(severity),
				d.Namespace,
				fmt.Sprintf("%s/%s", d.ResourceKind, d.ResourceName),
				d.Problem,
			},
			Status: severity,
		}
	}
	v.table.SetRows(rows)
}

// GetTable returns the underlying table component.
func (v *DiagnosisView) GetTable() *components.Table {
	return v.table
}
