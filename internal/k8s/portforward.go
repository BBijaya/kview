package k8s

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// PortForwardSession represents an active port forward connection
type PortForwardSession struct {
	ID           int
	Namespace    string
	ResourceType string // "pods" or "services"
	ResourceName string
	Container    string // informational
	LocalPort    int
	RemotePort   int
	Address      string
	stopChan     chan struct{}
	readyChan    chan struct{}
	fw           *portforward.PortForwarder
}

// PortForwardManager manages port forward sessions
type PortForwardManager struct {
	mu        sync.RWMutex
	sessions  []*PortForwardSession
	nextID    int
	config    *rest.Config
	clientset *kubernetes.Clientset
}

// NewPortForwardManager creates a new port forward manager
func NewPortForwardManager(config *rest.Config, clientset *kubernetes.Clientset) *PortForwardManager {
	return &PortForwardManager{
		config:    config,
		clientset: clientset,
		nextID:    1,
	}
}

// SetClient updates the REST config and clientset (called on context switch)
func (m *PortForwardManager) SetClient(config *rest.Config, clientset *kubernetes.Clientset) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
	m.clientset = clientset
}

// StartForward creates and starts a new port forward session.
// localPort of 0 means the system will choose a free port.
// address specifies the local listen address (e.g. "localhost", "0.0.0.0").
// Returns the session (with resolved LocalPort) or an error.
func (m *PortForwardManager) StartForward(namespace, resourceType, resourceName, container string, localPort, remotePort int, address string) (*PortForwardSession, error) {
	m.mu.Lock()
	config := m.config
	clientset := m.clientset
	id := m.nextID
	m.nextID++

	// Check for port conflict with existing sessions
	if localPort != 0 {
		for _, s := range m.sessions {
			if s.LocalPort == localPort && s.Address == address {
				m.mu.Unlock()
				return nil, fmt.Errorf("local port %d is already in use by port forward to %s/%s", localPort, s.Namespace, s.ResourceName)
			}
		}
	}
	m.mu.Unlock()

	if config == nil || clientset == nil {
		return nil, fmt.Errorf("no cluster connection")
	}

	// Build the portforward URL
	reqURL := clientset.CoreV1().RESTClient().Post().
		Resource(resourceType).
		Namespace(namespace).
		Name(resourceName).
		SubResource("portforward").
		URL()

	// Create SPDY transport
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create round tripper: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{
		Scheme: reqURL.Scheme,
		Host:   reqURL.Host,
		Path:   reqURL.Path,
	})

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})

	// Port spec: "localPort:remotePort"
	portSpec := fmt.Sprintf("%d:%d", localPort, remotePort)

	// Default address to localhost
	if address == "" {
		address = "localhost"
	}

	// Create the port forwarder with custom listen address
	fw, err := portforward.NewOnAddresses(dialer, []string{address}, []string{portSpec}, stopChan, readyChan, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create port forwarder: %w", err)
	}

	session := &PortForwardSession{
		ID:           id,
		Namespace:    namespace,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Container:    container,
		LocalPort:    localPort,
		RemotePort:   remotePort,
		Address:      address,
		stopChan:     stopChan,
		readyChan:    readyChan,
		fw:           fw,
	}

	// Start forwarding in background goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- fw.ForwardPorts()
	}()

	// Wait for ready or error with timeout
	select {
	case <-readyChan:
		// Resolve actual local port (important when localPort was 0)
		ports, err := fw.GetPorts()
		if err == nil && len(ports) > 0 {
			session.LocalPort = int(ports[0].Local)
		}
	case err := <-errChan:
		return nil, fmt.Errorf("port forward failed: %w", err)
	case <-time.After(10 * time.Second):
		close(stopChan)
		return nil, fmt.Errorf("port forward timed out")
	}

	m.mu.Lock()
	m.sessions = append(m.sessions, session)
	m.mu.Unlock()

	return session, nil
}

// StopForward stops a specific port forward session by ID
func (m *PortForwardManager) StopForward(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, s := range m.sessions {
		if s.ID == id {
			close(s.stopChan)
			m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("port forward session %d not found", id)
}

// StopAll stops all active port forward sessions
func (m *PortForwardManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, s := range m.sessions {
		close(s.stopChan)
	}
	m.sessions = nil
}

// ActiveSessions returns a copy of all active sessions
func (m *PortForwardManager) ActiveSessions() []*PortForwardSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*PortForwardSession, len(m.sessions))
	copy(result, m.sessions)
	return result
}

// ActiveForPod returns active port forward sessions for a specific pod
func (m *PortForwardManager) ActiveForPod(namespace, name string) []*PortForwardSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*PortForwardSession
	for _, s := range m.sessions {
		if s.Namespace == namespace && s.ResourceName == name {
			result = append(result, s)
		}
	}
	return result
}
