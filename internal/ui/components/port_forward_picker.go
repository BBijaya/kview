package components

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/theme"
)

// digitsOnly is a textinput.ValidateFunc that rejects non-digit characters
func digitsOnly(s string) error {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return fmt.Errorf("digits only")
		}
	}
	return nil
}

// PortForwardPickerConfirmMsg is sent when port forward is confirmed
type PortForwardPickerConfirmMsg struct {
	Namespace    string
	ResourceType string // "pods" or "services"
	ResourceName string
	Container    string
	LocalPort    int
	RemotePort   int
	Address      string
}

// PortForwardPickerCancelMsg is sent when port forward picker is cancelled
type PortForwardPickerCancelMsg struct{}

// portEntry represents a container port (used for hint display)
type portEntry struct {
	Container string
	Port      int32
	Protocol  string
	Name      string
}

// PortForwardPicker is a single-screen form for port forward parameters.
// Three editable fields: Container Port, Local Port, Address.
// Renders as an inline overlay on top of the existing view content.
type PortForwardPicker struct {
	visible      bool
	width        int
	height       int
	namespace    string
	resourceType string // "pods" or "services"
	resourceName string

	// Available container ports (for hint display)
	ports []portEntry

	// Three text input fields
	containerPortInput textinput.Model
	localPortInput     textinput.Model
	addressInput       textinput.Model
	focusedField       int  // 0=container port, 1=local port, 2=address
	localPortManual    bool // true once user manually edits local port
}

// newStyledInput creates a textinput with proper background styling to prevent color leak
func newStyledInput(placeholder string, charLimit, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = charLimit
	ti.Width = width
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.ColorText).Background(theme.ColorBackground)
	ti.PromptStyle = lipgloss.NewStyle().Background(theme.ColorBackground)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.ColorMuted).Background(theme.ColorBackground)
	ti.Cursor.Style = lipgloss.NewStyle().Background(theme.ColorHighlight)
	return ti
}

// NewPortForwardPicker creates a new port forward picker
func NewPortForwardPicker() *PortForwardPicker {
	containerPortInput := newStyledInput("port", 5, 10)
	containerPortInput.Validate = digitsOnly
	localPortInput := newStyledInput("port", 5, 10)
	localPortInput.Validate = digitsOnly
	return &PortForwardPicker{
		containerPortInput: containerPortInput,
		localPortInput:     localPortInput,
		addressInput:       newStyledInput("address", 45, 20),
	}
}

// Show shows the picker with the given pod and container info
func (p *PortForwardPicker) Show(namespace, pod string, containers []k8s.ContainerInfo) {
	p.visible = true
	p.namespace = namespace
	p.resourceType = "pods"
	p.resourceName = pod

	// Collect all defined container ports
	p.ports = nil
	for _, c := range containers {
		for _, port := range c.Ports {
			p.ports = append(p.ports, portEntry{
				Container: c.Name,
				Port:      port.ContainerPort,
				Protocol:  port.Protocol,
				Name:      port.Name,
			})
		}
	}

	// Set defaults based on available ports
	p.addressInput.SetValue("localhost")

	if len(p.ports) > 0 {
		portStr := fmt.Sprintf("%d", p.ports[0].Port)
		p.containerPortInput.SetValue(portStr)
		p.localPortInput.SetValue(portStr)
	} else {
		p.containerPortInput.SetValue("")
		p.localPortInput.SetValue("")
	}

	// Focus the first field
	p.focusedField = 0
	p.localPortManual = false
	p.containerPortInput.Focus()
	p.localPortInput.Blur()
	p.addressInput.Blur()
}

// ShowForService shows the picker for a service with its ports
func (p *PortForwardPicker) ShowForService(namespace, svcName string, servicePorts []k8s.ServicePort) {
	p.visible = true
	p.namespace = namespace
	p.resourceType = "services"
	p.resourceName = svcName

	// Collect service ports as port entries
	p.ports = nil
	for _, sp := range servicePorts {
		p.ports = append(p.ports, portEntry{
			Container: svcName,
			Port:      sp.Port,
			Protocol:  sp.Protocol,
			Name:      sp.Name,
		})
	}

	// Set defaults based on available ports
	p.addressInput.SetValue("localhost")

	if len(p.ports) > 0 {
		portStr := fmt.Sprintf("%d", p.ports[0].Port)
		p.containerPortInput.SetValue(portStr)
		p.localPortInput.SetValue(portStr)
	} else {
		p.containerPortInput.SetValue("")
		p.localPortInput.SetValue("")
	}

	// Focus the first field
	p.focusedField = 0
	p.localPortManual = false
	p.containerPortInput.Focus()
	p.localPortInput.Blur()
	p.addressInput.Blur()
}

// Hide hides the picker
func (p *PortForwardPicker) Hide() {
	p.visible = false
	p.containerPortInput.Blur()
	p.localPortInput.Blur()
	p.addressInput.Blur()
}

// IsVisible returns whether the picker is visible
func (p *PortForwardPicker) IsVisible() bool {
	return p.visible
}

// SetSize sets the overlay dimensions
func (p *PortForwardPicker) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Update handles messages for the picker
func (p *PortForwardPicker) Update(msg tea.Msg) (*PortForwardPicker, tea.Cmd) {
	if !p.visible {
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			p.Hide()
			return p, func() tea.Msg { return PortForwardPickerCancelMsg{} }

		case "tab", "down":
			p.blurAll()
			p.focusedField = (p.focusedField + 1) % 3
			p.focusCurrent()
			return p, textinput.Blink

		case "shift+tab", "up":
			p.blurAll()
			p.focusedField = (p.focusedField + 2) % 3
			p.focusCurrent()
			return p, textinput.Blink

		case "enter":
			containerPort, err := strconv.Atoi(p.containerPortInput.Value())
			if err != nil || containerPort < 1 || containerPort > 65535 {
				return p, nil
			}
			localPortStr := p.localPortInput.Value()
			if localPortStr == "" {
				localPortStr = p.containerPortInput.Value()
			}
			localPort, err := strconv.Atoi(localPortStr)
			if err != nil || localPort < 0 || localPort > 65535 {
				return p, nil
			}
			address := p.addressInput.Value()
			if address == "" {
				address = "localhost"
			}
			container := p.containerForPort(int32(containerPort))
			p.Hide()
			return p, func() tea.Msg {
				return PortForwardPickerConfirmMsg{
					Namespace:    p.namespace,
					ResourceType: p.resourceType,
					ResourceName: p.resourceName,
					Container:    container,
					LocalPort:    localPort,
					RemotePort:   containerPort,
					Address:      address,
				}
			}

		default:
			var cmd tea.Cmd
			switch p.focusedField {
			case 0:
				p.containerPortInput, cmd = p.containerPortInput.Update(msg)
				if !p.localPortManual {
					p.localPortInput.SetValue(p.containerPortInput.Value())
				}
			case 1:
				p.localPortInput, cmd = p.localPortInput.Update(msg)
				p.localPortManual = true
			case 2:
				p.addressInput, cmd = p.addressInput.Update(msg)
			}
			return p, cmd
		}
	}

	return p, nil
}

func (p *PortForwardPicker) blurAll() {
	p.containerPortInput.Blur()
	p.localPortInput.Blur()
	p.addressInput.Blur()
}

func (p *PortForwardPicker) focusCurrent() {
	switch p.focusedField {
	case 0:
		p.containerPortInput.Focus()
	case 1:
		p.localPortInput.Focus()
	case 2:
		p.addressInput.Focus()
	}
}

func (p *PortForwardPicker) containerForPort(port int32) string {
	for _, entry := range p.ports {
		if entry.Port == port {
			return entry.Container
		}
	}
	return ""
}

// renderBox builds the styled overlay box with "Port Forward" centered in the top border.
func (p *PortForwardPicker) renderBox() string {
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	labelStyle := lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Background(theme.ColorBackground)

	mutedStyle := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Background(theme.ColorBackground)

	focusLabelStyle := lipgloss.NewStyle().
		Foreground(theme.ColorHighlight).
		Background(theme.ColorBackground)

	overlayWidth := p.width * 2 / 5 // 40% of terminal width
	if overlayWidth < 40 {
		overlayWidth = 40
	}
	if overlayWidth > 70 {
		overlayWidth = 70
	}
	if p.width > 0 && overlayWidth > p.width-4 {
		overlayWidth = p.width - 4
	}
	innerWidth := overlayWidth - 2 // subtract border columns

	// Scale address input width to use available space
	// Budget: label(15) + space(1) + prompt(2) + input(addrWidth) + cursor(1) = pad = innerWidth - 2
	addrWidth := overlayWidth - 23
	if addrWidth < 15 {
		addrWidth = 15
	}
	p.addressInput.Width = addrWidth

	borderStyle := lipgloss.NewStyle().Foreground(theme.ColorPrimary).Background(theme.ColorBackground)
	borderChar := borderStyle.Render

	// Build top border with centered title: ╭──── Port Forward ────╮
	title := lipgloss.NewStyle().
		Foreground(theme.ColorHighlight).
		Background(theme.ColorBackground).
		Bold(true).
		Render("Port Forward")
	titleWidth := lipgloss.Width("Port Forward")
	dashSpace := innerWidth - titleWidth - 2 // 2 for spaces around title
	if dashSpace < 2 {
		dashSpace = 2
	}
	leftDashes := dashSpace / 2
	rightDashes := dashSpace - leftDashes
	topBorder := borderChar("╭") +
		borderChar(strings.Repeat("─", leftDashes)) +
		borderChar(" ") + title + borderChar(" ") +
		borderChar(strings.Repeat("─", rightDashes)) +
		borderChar("╮")

	// Build content lines (each normalized to exactly pad visible chars)
	padContent := func(line string) string {
		w := lipgloss.Width(line)
		pad := innerWidth - 2 // inner padding (1 each side)
		if w < pad {
			line += bgStyle.Render(strings.Repeat(" ", pad-w))
		} else if w > pad {
			line = ansiTruncateClean(line, pad)
		}
		return borderChar("│") + bgStyle.Render(" ") + line + bgStyle.Render(" ") + borderChar("│")
	}
	emptyLine := borderChar("│") + bgStyle.Render(strings.Repeat(" ", innerWidth)) + borderChar("│")

	var lines []string
	lines = append(lines, topBorder)

	// Namespace / resource name (truncate to fit content area)
	maxContentWidth := innerWidth - 2 // 1 char padding each side
	nsResource := p.namespace + "/" + p.resourceName
	if len(nsResource) > maxContentWidth {
		nsResource = nsResource[:maxContentWidth-1] + "…"
	}
	lines = append(lines, padContent(mutedStyle.Render(nsResource)))

	// Available ports (vertical, one per line)
	if len(p.ports) > 0 {
		for _, entry := range p.ports {
			hint := fmt.Sprintf("  %d/%s", entry.Port, entry.Protocol)
			if entry.Name != "" {
				hint += " (" + entry.Name + ")"
			}
			lines = append(lines, padContent(mutedStyle.Render(hint)))
		}
	}

	// Blank separator
	lines = append(lines, emptyLine)

	// Input fields
	portLabel := "Container Port:"
	if p.resourceType == "services" {
		portLabel = "Service Port:  "
	}

	type field struct {
		label string
		view  string
	}
	fields := []field{
		{portLabel, p.containerPortInput.View()},
		{"Local Port:    ", p.localPortInput.View()},
		{"Address:       ", p.addressInput.View()},
	}

	for i, f := range fields {
		style := labelStyle
		if i == p.focusedField {
			style = focusLabelStyle
		}
		line := style.Render(f.label) + " " + f.view
		lines = append(lines, padContent(line))
	}

	// Blank separator + footer shortcuts
	lines = append(lines, emptyLine)
	lines = append(lines, padContent(mutedStyle.Render("tab/↑↓:nav  enter:confirm  esc:cancel")))

	// Bottom border
	bottomBorder := borderChar("╰") + borderChar(strings.Repeat("─", innerWidth)) + borderChar("╯")
	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

// ViewOverlay composites the picker box on top of the background content.
// Only the exact box area is replaced; background content remains visible
// on both sides and above/below the box.
func (p *PortForwardPicker) ViewOverlay(background string) string {
	if !p.visible {
		return background
	}
	return OverlayCenter(p.renderBox(), background, p.width, p.height)
}
