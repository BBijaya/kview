package cache

import (
	"strings"
	"sync"

	"github.com/bijaya/kview/internal/k8s"
)

// Index provides fast lookups for cached resources
type Index struct {
	mu sync.RWMutex

	// byName maps "kind/namespace/name" to UID
	byName map[string]string

	// byLabel maps "label=value" to list of UIDs
	byLabel map[string][]string

	// byOwner maps owner UID to list of owned UIDs
	byOwner map[string][]string

	// byNamespace maps namespace to list of UIDs
	byNamespace map[string][]string
}

// NewIndex creates a new index
func NewIndex() *Index {
	return &Index{
		byName:      make(map[string]string),
		byLabel:     make(map[string][]string),
		byOwner:     make(map[string][]string),
		byNamespace: make(map[string][]string),
	}
}

// Add adds a resource to the index
func (i *Index) Add(r k8s.Resource) {
	i.mu.Lock()
	defer i.mu.Unlock()

	uid := r.UID

	// Index by name
	key := nameKey(r.Kind, r.Namespace, r.Name)
	i.byName[key] = uid

	// Index by labels
	for k, v := range r.Labels {
		labelKey := k + "=" + v
		i.byLabel[labelKey] = appendUnique(i.byLabel[labelKey], uid)
	}

	// Index by owner
	for _, owner := range r.OwnerRefs {
		i.byOwner[owner.UID] = appendUnique(i.byOwner[owner.UID], uid)
	}

	// Index by namespace
	if r.Namespace != "" {
		i.byNamespace[r.Namespace] = appendUnique(i.byNamespace[r.Namespace], uid)
	}
}

// Remove removes a resource from the index
func (i *Index) Remove(r k8s.Resource) {
	i.mu.Lock()
	defer i.mu.Unlock()

	uid := r.UID

	// Remove from name index
	key := nameKey(r.Kind, r.Namespace, r.Name)
	delete(i.byName, key)

	// Remove from label index
	for k, v := range r.Labels {
		labelKey := k + "=" + v
		i.byLabel[labelKey] = removeFromSlice(i.byLabel[labelKey], uid)
	}

	// Remove from owner index
	for _, owner := range r.OwnerRefs {
		i.byOwner[owner.UID] = removeFromSlice(i.byOwner[owner.UID], uid)
	}

	// Remove from namespace index
	if r.Namespace != "" {
		i.byNamespace[r.Namespace] = removeFromSlice(i.byNamespace[r.Namespace], uid)
	}
}

// LookupByName finds a resource UID by kind, namespace, and name
func (i *Index) LookupByName(kind, namespace, name string) (string, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	key := nameKey(kind, namespace, name)
	uid, ok := i.byName[key]
	return uid, ok
}

// LookupByLabel finds resource UIDs by label
func (i *Index) LookupByLabel(label, value string) []string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	key := label + "=" + value
	return i.byLabel[key]
}

// LookupByOwner finds resource UIDs owned by the given owner UID
func (i *Index) LookupByOwner(ownerUID string) []string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.byOwner[ownerUID]
}

// LookupByNamespace finds resource UIDs in the given namespace
func (i *Index) LookupByNamespace(namespace string) []string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.byNamespace[namespace]
}

// Clear clears the entire index
func (i *Index) Clear() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.byName = make(map[string]string)
	i.byLabel = make(map[string][]string)
	i.byOwner = make(map[string][]string)
	i.byNamespace = make(map[string][]string)
}

func nameKey(kind, namespace, name string) string {
	return strings.ToLower(kind) + "/" + namespace + "/" + name
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
