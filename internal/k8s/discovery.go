package k8s

import (
	"sort"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
)

// APIResourceInfo describes a single API resource discovered from the cluster.
type APIResourceInfo struct {
	Group      string // "apps", "" for core, "cert-manager.io", etc.
	Version    string // "v1", "v1beta1"
	Resource   string // plural: "deployments", "certificates"
	Kind       string // PascalCase: "Deployment", "Certificate"
	ShortNames []string
	Namespaced bool
	Verbs      []string
	GVR        schema.GroupVersionResource
}

// APIResourceRegistry holds all discovered API resources with lookup indexes.
type APIResourceRegistry struct {
	mu          sync.RWMutex
	resources   []APIResourceInfo
	byResource  map[string]int // plural name -> index
	byKind      map[string]int // lowercase kind -> index
	byShortName map[string]int // short name -> index
}

// NewAPIResourceRegistry creates an empty registry.
func NewAPIResourceRegistry() *APIResourceRegistry {
	return &APIResourceRegistry{
		byResource:  make(map[string]int),
		byKind:      make(map[string]int),
		byShortName: make(map[string]int),
	}
}

// Discover queries the cluster for all available API resources and populates
// the registry. It handles partial discovery errors gracefully (some API groups
// may be unavailable).
func (r *APIResourceRegistry) Discover(clientset *kubernetes.Clientset) error {
	lists, err := clientset.Discovery().ServerPreferredResources()
	if err != nil {
		// Partial results are still usable when some groups fail
		if !discovery.IsGroupDiscoveryFailedError(err) {
			return err
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.resources = nil
	r.byResource = make(map[string]int)
	r.byKind = make(map[string]int)
	r.byShortName = make(map[string]int)

	for _, list := range lists {
		if list == nil {
			continue
		}
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			continue
		}

		for _, res := range list.APIResources {
			// Skip subresources (e.g. "pods/log", "deployments/scale")
			if strings.Contains(res.Name, "/") {
				continue
			}

			// Skip resources that can't be listed
			if !hasVerb(res.Verbs, "list") {
				continue
			}

			info := APIResourceInfo{
				Group:      gv.Group,
				Version:    gv.Version,
				Resource:   res.Name,
				Kind:       res.Kind,
				ShortNames: res.ShortNames,
				Namespaced: res.Namespaced,
				Verbs:      verbsToStrings(res.Verbs),
				GVR: schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: res.Name,
				},
			}

			idx := len(r.resources)
			r.resources = append(r.resources, info)

			// Build lookup indexes (first-registered wins for duplicates)
			key := strings.ToLower(res.Name)
			if _, exists := r.byResource[key]; !exists {
				r.byResource[key] = idx
			}

			kindKey := strings.ToLower(res.Kind)
			if _, exists := r.byKind[kindKey]; !exists {
				r.byKind[kindKey] = idx
			}

			for _, sn := range res.ShortNames {
				snKey := strings.ToLower(sn)
				if _, exists := r.byShortName[snKey]; !exists {
					r.byShortName[snKey] = idx
				}
			}
		}
	}

	return nil
}

// Lookup finds a resource by plural name, kind, or short name (case-insensitive).
// It checks byResource -> byKind -> byShortName in order.
func (r *APIResourceRegistry) Lookup(name string) (*APIResourceInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := strings.ToLower(name)

	if idx, ok := r.byResource[key]; ok {
		return &r.resources[idx], true
	}
	if idx, ok := r.byKind[key]; ok {
		return &r.resources[idx], true
	}
	if idx, ok := r.byShortName[key]; ok {
		return &r.resources[idx], true
	}

	return nil, false
}

// All returns all discovered resources sorted by group then resource name.
func (r *APIResourceRegistry) All() []APIResourceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]APIResourceInfo, len(r.resources))
	copy(result, r.resources)

	sort.Slice(result, func(i, j int) bool {
		if result[i].Group != result[j].Group {
			return result[i].Group < result[j].Group
		}
		return result[i].Resource < result[j].Resource
	})

	return result
}

func hasVerb(verbs metav1.Verbs, verb string) bool {
	for _, v := range verbs {
		if v == verb {
			return true
		}
	}
	return false
}

func verbsToStrings(verbs metav1.Verbs) []string {
	result := make([]string, len(verbs))
	for i, v := range verbs {
		result[i] = string(v)
	}
	return result
}
