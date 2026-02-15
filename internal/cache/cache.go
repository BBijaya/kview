package cache

import (
	"sync"
	"time"

	"github.com/bijaya/kview/internal/k8s"
)

// Cache provides in-memory caching for Kubernetes resources
type Cache struct {
	mu        sync.RWMutex
	pods      map[string][]k8s.PodInfo        // namespace -> pods
	deploys   map[string][]k8s.DeploymentInfo // namespace -> deployments
	services  map[string][]k8s.ServiceInfo    // namespace -> services
	resources map[string]map[string]k8s.Resource // kind -> uid -> resource
	ttl       time.Duration
	lastFetch map[string]time.Time // cache key -> last fetch time
}

// New creates a new cache with the specified TTL
func New(ttl time.Duration) *Cache {
	return &Cache{
		pods:      make(map[string][]k8s.PodInfo),
		deploys:   make(map[string][]k8s.DeploymentInfo),
		services:  make(map[string][]k8s.ServiceInfo),
		resources: make(map[string]map[string]k8s.Resource),
		ttl:       ttl,
		lastFetch: make(map[string]time.Time),
	}
}

// SetPods caches pods for a namespace
func (c *Cache) SetPods(namespace string, pods []k8s.PodInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pods[namespace] = pods
	c.lastFetch["pods:"+namespace] = time.Now()
}

// GetPods returns cached pods for a namespace
func (c *Cache) GetPods(namespace string) ([]k8s.PodInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.isExpired("pods:" + namespace) {
		return nil, false
	}

	pods, ok := c.pods[namespace]
	return pods, ok
}

// SetDeployments caches deployments for a namespace
func (c *Cache) SetDeployments(namespace string, deploys []k8s.DeploymentInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deploys[namespace] = deploys
	c.lastFetch["deploys:"+namespace] = time.Now()
}

// GetDeployments returns cached deployments for a namespace
func (c *Cache) GetDeployments(namespace string) ([]k8s.DeploymentInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.isExpired("deploys:" + namespace) {
		return nil, false
	}

	deploys, ok := c.deploys[namespace]
	return deploys, ok
}

// SetServices caches services for a namespace
func (c *Cache) SetServices(namespace string, services []k8s.ServiceInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[namespace] = services
	c.lastFetch["services:"+namespace] = time.Now()
}

// GetServices returns cached services for a namespace
func (c *Cache) GetServices(namespace string) ([]k8s.ServiceInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.isExpired("services:" + namespace) {
		return nil, false
	}

	services, ok := c.services[namespace]
	return services, ok
}

// SetResource caches a single resource
func (c *Cache) SetResource(kind string, resource k8s.Resource) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.resources[kind] == nil {
		c.resources[kind] = make(map[string]k8s.Resource)
	}
	c.resources[kind][resource.UID] = resource
}

// GetResource returns a cached resource by kind and UID
func (c *Cache) GetResource(kind, uid string) (k8s.Resource, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if kindMap, ok := c.resources[kind]; ok {
		resource, ok := kindMap[uid]
		return resource, ok
	}
	return k8s.Resource{}, false
}

// InvalidatePods invalidates the pods cache for a namespace
func (c *Cache) InvalidatePods(namespace string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pods, namespace)
	delete(c.lastFetch, "pods:"+namespace)
}

// InvalidateDeployments invalidates the deployments cache for a namespace
func (c *Cache) InvalidateDeployments(namespace string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.deploys, namespace)
	delete(c.lastFetch, "deploys:"+namespace)
}

// InvalidateServices invalidates the services cache for a namespace
func (c *Cache) InvalidateServices(namespace string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.services, namespace)
	delete(c.lastFetch, "services:"+namespace)
}

// InvalidateAll clears the entire cache
func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pods = make(map[string][]k8s.PodInfo)
	c.deploys = make(map[string][]k8s.DeploymentInfo)
	c.services = make(map[string][]k8s.ServiceInfo)
	c.resources = make(map[string]map[string]k8s.Resource)
	c.lastFetch = make(map[string]time.Time)
}

func (c *Cache) isExpired(key string) bool {
	if c.ttl == 0 {
		return false
	}
	lastFetch, ok := c.lastFetch[key]
	if !ok {
		return true
	}
	return time.Since(lastFetch) > c.ttl
}

// AllPods returns all cached pods across all namespaces
func (c *Cache) AllPods() []k8s.PodInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var allPods []k8s.PodInfo
	for _, pods := range c.pods {
		allPods = append(allPods, pods...)
	}
	return allPods
}

// AllDeployments returns all cached deployments across all namespaces
func (c *Cache) AllDeployments() []k8s.DeploymentInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var allDeploys []k8s.DeploymentInfo
	for _, deploys := range c.deploys {
		allDeploys = append(allDeploys, deploys...)
	}
	return allDeploys
}

// AllServices returns all cached services across all namespaces
func (c *Cache) AllServices() []k8s.ServiceInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var allServices []k8s.ServiceInfo
	for _, services := range c.services {
		allServices = append(allServices, services...)
	}
	return allServices
}
