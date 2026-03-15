package components

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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
	errorMsg           string // validation error shown below fields
}

// newStyledInput creates a textinput with proper background styling to prevent color leak
func newStyledInput(placeholder string, charLimit, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = charLimit
	ti.SetWidth(width)
	styles := textinput.DefaultDarkStyles()
	styles.Focused.Text = lipgloss.NewStyle().Foreground(theme.ColorText).Background(theme.ColorBackground)
	styles.Focused.Prompt = lipgloss.NewStyle().Background(theme.ColorBackground)
	styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(theme.ColorMuted).Background(theme.ColorBackground)
	ti.SetStyles(styles)
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
	var ports []portEntry
	for _, c := range containers {
		for _, port := range c.Ports {
			ports = append(ports, portEntry{c.Name, port.ContainerPort, port.Protocol, port.Name})
		}
	}
	p.show(namespace, "pods", pod, ports)
}

// ShowForService shows the picker for a service with its ports
func (p *PortForwardPicker) ShowForService(namespace, svcName string, servicePorts []k8s.ServicePort) {
	var ports []portEntry
	for _, sp := range servicePorts {
		ports = append(ports, portEntry{svcName, sp.Port, sp.Protocol, sp.Name})
	}
	p.show(namespace, "services", svcName, ports)
}

// show is the shared setup for both pod and service port forwarding
func (p *PortForwardPicker) show(namespace, resourceType, resourceName string, ports []portEntry) {
	p.visible = true
	p.namespace = namespace
	p.resourceType = resourceType
	p.resourceName = resourceName
	p.ports = ports
	p.errorMsg = ""

	p.addressInput.SetValue("localhost")
	if len(p.ports) > 0 {
		portStr := fmt.Sprintf("%d", p.ports[0].Port)
		p.containerPortInput.SetValue(portStr)
		p.localPortInput.SetValue(portStr)
	} else {
		p.containerPortInput.SetValue("")
		p.localPortInput.SetValue("")
	}

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
	case tea.KeyPressMsg:
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
				p.errorMsg = "Container port must be 1-65535"
				return p, nil
			}
			localPortStr := p.localPortInput.Value()
			if localPortStr == "" {
				localPortStr = p.containerPortInput.Value()
			}
			localPort, err := strconv.Atoi(localPortStr)
			if err != nil || localPort < 0 || localPort > 65535 {
				p.errorMsg = "Local port must be 0-65535"
				return p, nil
			}
			if localPort > 0 && localPort < 1024 {
				p.errorMsg = fmt.Sprintf("Port %d requires root (use 1024+ or 0=auto)", localPort)
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
			p.errorMsg = ""
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

func (p *PortForwardPicker) containersForPort(port int32) []string {
	var result []string
	for _, entry := range p.ports {
		if entry.Port == port {
			result = append(result, entry.Container)
		}
	}
	return result
}

func (p *PortForwardPicker) hasMultipleContainers() bool {
	if len(p.ports) < 2 {
		return false
	}
	first := p.ports[0].Container
	for _, entry := range p.ports[1:] {
		if entry.Container != first {
			return true
		}
	}
	return false
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
	p.addressInput.SetWidth(addrWidth)

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
	multiContainer := p.hasMultipleContainers()
	if len(p.ports) > 0 {
		for _, entry := range p.ports {
			hint := fmt.Sprintf("  %d/%s", entry.Port, entry.Protocol)
			if entry.Name != "" {
				hint += " (" + entry.Name + ")"
			}
			if multiContainer {
				hint += " [" + entry.Container + "]"
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

	errorStyle := lipgloss.NewStyle().
		Foreground(theme.ColorError).
		Background(theme.ColorBackground)

	type field struct {
		label  string
		view   string
		suffix string
	}
	fields := []field{
		{portLabel, p.containerPortInput.View(), ""},
		{"Local Port:    ", p.localPortInput.View(), mutedStyle.Render(" (0=auto)")},
		{"Address:       ", p.addressInput.View(), ""},
	}

	for i, f := range fields {
		style := labelStyle
		if i == p.focusedField {
			style = focusLabelStyle
		}
		line := style.Render(f.label) + " " + f.view + f.suffix
		lines = append(lines, padContent(line))
	}

	// Container disambiguation note when multiple containers match the typed port
	if multiContainer {
		if containerPort, err := strconv.Atoi(p.containerPortInput.Value()); err == nil {
			matches := p.containersForPort(int32(containerPort))
			if len(matches) > 1 {
				lines = append(lines, padContent(mutedStyle.Render("→ using container: "+matches[0])))
			}
		}
	}

	// Validation error message
	if p.errorMsg != "" {
		lines = append(lines, padContent(errorStyle.Render(p.errorMsg)))
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
