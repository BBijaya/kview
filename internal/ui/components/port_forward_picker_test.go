package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/bijaya/kview/internal/k8s"
)

func manyPortsPicker(portCount, width, height int) *PortForwardPicker {
	p := NewPortForwardPicker()
	p.SetSize(width, height)
	var ports []k8s.ContainerPort
	for i := 0; i < portCount; i++ {
		ports = append(ports, k8s.ContainerPort{
			Name:          "",
			ContainerPort: int32(8000 + i),
			Protocol:      "TCP",
		})
	}
	p.Show("default", "web-abc", []k8s.ContainerInfo{{Name: "app", Ports: ports}})
	return p
}

func TestPortPickerBoxFitsTerminalHeight(t *testing.T) {
	// A pod with many ports on a short terminal: the box previously grew
	// one line per port and Overlay clipped the bottom, hiding the input
	// fields, validation errors, and the enter/esc footer.
	p := manyPortsPicker(30, 100, 20)
	box := p.renderBox()
	lines := strings.Split(box, "\n")
	if len(lines) > 18 { // height - 2 margin
		t.Errorf("box is %d lines tall on a 20-line terminal (bottom would be clipped)", len(lines))
	}

	plain := ansi.Strip(box)
	if !strings.Contains(plain, "enter:confirm") {
		t.Error("footer shortcuts missing from budgeted box")
	}
	if !strings.Contains(plain, "more port(s)") {
		t.Error("overflow summary line missing when ports are collapsed")
	}
}

func TestPortPickerShowsAllPortsWhenTheyFit(t *testing.T) {
	p := manyPortsPicker(3, 100, 40)
	plain := ansi.Strip(p.renderBox())
	for _, port := range []string{"8000/TCP", "8001/TCP", "8002/TCP"} {
		if !strings.Contains(plain, port) {
			t.Errorf("port hint %q missing on tall terminal", port)
		}
	}
	if strings.Contains(plain, "more port(s)") {
		t.Error("unexpected overflow summary when everything fits")
	}
}

func TestPortPickerSurvivesTinyTerminal(t *testing.T) {
	p := manyPortsPicker(10, 50, 8)
	box := p.renderBox() // must not panic
	plain := ansi.Strip(box)
	if !strings.Contains(plain, "more port(s)") {
		t.Error("expected all hints collapsed into summary on tiny terminal")
	}
}
