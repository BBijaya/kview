package k8s

import (
	"context"
	"fmt"
	"sync"
)

// MultiClusterManager manages connections to multiple Kubernetes clusters
type MultiClusterManager struct {
	mu      sync.RWMutex
	clients map[string]*K8sClient
	active  string // Active context name
}

// NewMultiClusterManager creates a new multi-cluster manager
func NewMultiClusterManager() *MultiClusterManager {
	return &MultiClusterManager{
		clients: make(map[string]*K8sClient),
	}
}

// Connect connects to a cluster by context name
func (m *MultiClusterManager) Connect(contextName string) (*K8sClient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already connected
	if client, ok := m.clients[contextName]; ok {
		m.active = contextName
		return client, nil
	}

	// Create new client
	client, err := NewClientForContext(contextName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", contextName, err)
	}

	m.clients[contextName] = client
	m.active = contextName
	return client, nil
}

// Disconnect disconnects from a cluster
func (m *MultiClusterManager) Disconnect(contextName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.clients, contextName)

	if m.active == contextName {
		m.active = ""
		// Set active to first remaining client
		for name := range m.clients {
			m.active = name
			break
		}
	}
}

// DisconnectAll disconnects from all clusters
func (m *MultiClusterManager) DisconnectAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients = make(map[string]*K8sClient)
	m.active = ""
}

// GetClient returns the client for a specific context
func (m *MultiClusterManager) GetClient(contextName string) (*K8sClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[contextName]
	return client, ok
}

// GetActiveClient returns the currently active client
func (m *MultiClusterManager) GetActiveClient() (*K8sClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.active == "" {
		return nil, false
	}
	client, ok := m.clients[m.active]
	return client, ok
}

// SetActive sets the active context
func (m *MultiClusterManager) SetActive(contextName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.clients[contextName]; !ok {
		return fmt.Errorf("not connected to %s", contextName)
	}
	m.active = contextName
	return nil
}

// ActiveContext returns the name of the active context
func (m *MultiClusterManager) ActiveContext() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}

// ConnectedContexts returns a list of all connected context names
func (m *MultiClusterManager) ConnectedContexts() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	contexts := make([]string, 0, len(m.clients))
	for name := range m.clients {
		contexts = append(contexts, name)
	}
	return contexts
}

// ConnectedCount returns the number of connected clusters
func (m *MultiClusterManager) ConnectedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// ClusterInfo contains information about a connected cluster
type ClusterInfo struct {
	ContextName   string
	ServerURL     string
	ServerVersion string
	IsActive      bool
}

// GetClusterInfo returns information about all connected clusters
func (m *MultiClusterManager) GetClusterInfo() []ClusterInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := make([]ClusterInfo, 0, len(m.clients))
	for name, client := range m.clients {
		info = append(info, ClusterInfo{
			ContextName:   name,
			ServerURL:     client.ClusterID(),
			ServerVersion: client.ServerVersion(),
			IsActive:      name == m.active,
		})
	}
	return info
}

// ForEach executes a function for each connected client
func (m *MultiClusterManager) ForEach(fn func(contextName string, client *K8sClient) error) error {
	m.mu.RLock()
	clients := make(map[string]*K8sClient)
	for k, v := range m.clients {
		clients[k] = v
	}
	m.mu.RUnlock()

	for name, client := range clients {
		if err := fn(name, client); err != nil {
			return err
		}
	}
	return nil
}

// ForEachParallel executes a function for each connected client in parallel
func (m *MultiClusterManager) ForEachParallel(ctx context.Context, fn func(contextName string, client *K8sClient) error) []error {
	m.mu.RLock()
	clients := make(map[string]*K8sClient)
	for k, v := range m.clients {
		clients[k] = v
	}
	m.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(clients))

	for name, client := range clients {
		wg.Add(1)
		go func(n string, c *K8sClient) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				if err := fn(n, c); err != nil {
					errChan <- fmt.Errorf("%s: %w", n, err)
				}
			}
		}(name, client)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	return errors
}

// ComparePodsAcrossClusters compares pods across all connected clusters
func (m *MultiClusterManager) ComparePodsAcrossClusters(ctx context.Context, namespace string) (map[string][]PodInfo, error) {
	m.mu.RLock()
	clients := make(map[string]*K8sClient)
	for k, v := range m.clients {
		clients[k] = v
	}
	m.mu.RUnlock()

	result := make(map[string][]PodInfo)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for name, client := range clients {
		wg.Add(1)
		go func(n string, c *K8sClient) {
			defer wg.Done()

			pods, err := c.ListPods(ctx, namespace)
			if err != nil {
				return // Skip errors for comparison
			}

			mu.Lock()
			result[n] = pods
			mu.Unlock()
		}(name, client)
	}

	wg.Wait()
	return result, nil
}
