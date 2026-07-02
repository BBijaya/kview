package components

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func workloadsHeader(width int) *Header {
	h := NewHeader()
	h.SetWidth(width)
	h.SetContext("minikube")
	h.SetClusterName("kubernetes")
	h.SetUser("minikube-user")
	h.SetServerVersion("v1.28.0")
	h.SetCategories([]string{"Workloads", "Network", "Config", "Cluster", "Helm"})
	h.SetResourceTabs([][]string{
		{"Pods", "Deploy", "RS", "DS", "STS", "Jobs", "CJ", "HPA"},
		{"Svc", "Ing"},
		{"CM", "Sec", "PVC"},
		{"Nodes", "Events"},
		{"Rel"},
	})
	h.SetActiveCategory(0)
	h.SetCurrentViewType("pods")
	return h
}

func TestHeaderTabRowNeverExceedsWidth(t *testing.T) {
	for _, width := range []int{40, 60, 80, 100, 120, 200} {
		h := workloadsHeader(width)
		row := h.renderCategoryWithResources()
		if w := lipgloss.Width(row); w > width {
			t.Errorf("width %d: tab row is %d cells wide (overflow breaks 7-line header)", width, w)
		}
	}
}

func TestHeaderInfoPaneAlwaysSevenLines(t *testing.T) {
	for _, width := range []int{60, 80, 100, 160} {
		h := workloadsHeader(width)
		view := h.InfoPaneView()
		lines := strings.Split(view, "\n")
		if len(lines) != 7 {
			t.Fatalf("width %d: InfoPaneView has %d lines, want 7", width, len(lines))
		}
		for i, line := range lines {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: header line %d is %d cells wide", width, i+1, w)
			}
		}
	}
}

func TestHeaderTabRowFullContentOnWideTerminal(t *testing.T) {
	h := workloadsHeader(200)
	plain := ansi.Strip(h.renderCategoryWithResources())
	for _, want := range []string{"Workloads", "Helm", "[1] Pods", "[8] HPA"} {
		if !strings.Contains(plain, want) {
			t.Errorf("wide tab row missing %q\ngot: %s", want, plain)
		}
	}
	if strings.Contains(plain, "…") {
		t.Errorf("wide tab row should not be truncated, got: %s", plain)
	}
}

func TestHeaderTabRowTruncatesWithEllipsis(t *testing.T) {
	h := workloadsHeader(60)
	plain := ansi.Strip(h.renderCategoryWithResources())
	if !strings.Contains(plain, "…") {
		t.Errorf("narrow tab row (60 cols) should end with ellipsis, got: %s", plain)
	}
	// Categories come before tabs, so the first category must survive
	if !strings.Contains(plain, "Workloads") {
		t.Errorf("narrow tab row lost leading category, got: %s", plain)
	}
}
