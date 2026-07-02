package components

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bijaya/kview/internal/k8s"
)

func TestSearchInputAcceptsPaste(t *testing.T) {
	s := NewSearchInput()
	s.SetWidth(60)
	s.Show()

	s, cmd := s.Update(tea.PasteMsg{Content: "app=nginx,tier!=cache"})
	if got := s.Value(); got != "app=nginx,tier!=cache" {
		t.Errorf("value after paste = %q, want pasted text", got)
	}
	// A value change must emit FilterChangedMsg so live filtering applies
	if cmd == nil {
		t.Fatal("expected a command carrying FilterChangedMsg, got nil")
	}
}

func TestCommandInputAcceptsPaste(t *testing.T) {
	c := NewCommandInput()
	c.SetWidth(60)
	c.Show()

	c, _ = c.Update(tea.PasteMsg{Content: "xray deploy"})
	if got := c.Value(); got != "xray deploy" {
		t.Errorf("value after paste = %q, want pasted text", got)
	}
}

func TestPortPickerAcceptsPasteInFocusedField(t *testing.T) {
	p := NewPortForwardPicker()
	p.SetSize(100, 40)
	p.Show("default", "web-abc", []k8s.ContainerInfo{{Name: "app"}})

	// Field 0 (container port) is focused on Show; clear the prefill first
	p.containerPortInput.SetValue("")
	p, _ = p.Update(tea.PasteMsg{Content: "8080"})
	if got := p.containerPortInput.Value(); got != "8080" {
		t.Errorf("container port after paste = %q, want 8080", got)
	}
	// Auto-sync to local port must apply on paste like on typing
	if got := p.localPortInput.Value(); got != "8080" {
		t.Errorf("local port after paste = %q, want auto-synced 8080", got)
	}
}

func TestScalePickerAcceptsPaste(t *testing.T) {
	p := NewScalePicker()
	p.SetSize(100, 40)
	p.Show("default", "web", "Deployment", 1)

	p.replicaInput.SetValue("")
	p, _ = p.Update(tea.PasteMsg{Content: "5"})
	if got := p.replicaInput.Value(); got != "5" {
		t.Errorf("replicas after paste = %q, want 5", got)
	}
}
