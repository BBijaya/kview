package k8s

import (
	"context"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

// Watcher watches Kubernetes resources for changes
type Watcher struct {
	client    dynamic.Interface
	clusterID string
	watchers  map[string]watch.Interface
	mu        sync.RWMutex
	handlers  []WatchHandler
}

// WatchHandler handles watch events
type WatchHandler func(event WatchEvent)

// NewWatcher creates a new resource watcher
func NewWatcher(client dynamic.Interface, clusterID string) *Watcher {
	return &Watcher{
		client:    client,
		clusterID: clusterID,
		watchers:  make(map[string]watch.Interface),
	}
}

// AddHandler adds a handler for watch events
func (w *Watcher) AddHandler(handler WatchHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers = append(w.handlers, handler)
}

// Watch starts watching a resource type
func (w *Watcher) Watch(ctx context.Context, gvr schema.GroupVersionResource, namespace string) error {
	key := gvr.String() + "/" + namespace

	w.mu.Lock()
	if _, exists := w.watchers[key]; exists {
		w.mu.Unlock()
		return nil // Already watching
	}
	w.mu.Unlock()

	var watcher watch.Interface
	var err error

	if namespace == "" {
		watcher, err = w.client.Resource(gvr).Watch(ctx, metav1.ListOptions{})
	} else {
		watcher, err = w.client.Resource(gvr).Namespace(namespace).Watch(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return err
	}

	w.mu.Lock()
	w.watchers[key] = watcher
	w.mu.Unlock()

	// Process events in a goroutine
	go w.processEvents(ctx, key, watcher)

	return nil
}

// Stop stops watching a resource type
func (w *Watcher) Stop(gvr schema.GroupVersionResource, namespace string) {
	key := gvr.String() + "/" + namespace

	w.mu.Lock()
	defer w.mu.Unlock()

	if watcher, exists := w.watchers[key]; exists {
		watcher.Stop()
		delete(w.watchers, key)
	}
}

// StopAll stops all watchers
func (w *Watcher) StopAll() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for key, watcher := range w.watchers {
		watcher.Stop()
		delete(w.watchers, key)
	}
}

func (w *Watcher) processEvents(ctx context.Context, key string, watcher watch.Interface) {
	defer func() {
		w.mu.Lock()
		delete(w.watchers, key)
		w.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			watcher.Stop()
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return // Watcher closed
			}

			obj, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				continue
			}

			resource := w.unstructuredToResource(obj)
			watchEvent := WatchEvent{
				Type:     WatchEventType(event.Type),
				Resource: &resource,
			}

			// Call all handlers
			w.mu.RLock()
			handlers := make([]WatchHandler, len(w.handlers))
			copy(handlers, w.handlers)
			w.mu.RUnlock()

			for _, handler := range handlers {
				handler(watchEvent)
			}
		}
	}
}

func (w *Watcher) unstructuredToResource(obj *unstructured.Unstructured) Resource {
	resource := Resource{
		UID:         string(obj.GetUID()),
		APIVersion:  obj.GetAPIVersion(),
		Kind:        obj.GetKind(),
		Namespace:   obj.GetNamespace(),
		Name:        obj.GetName(),
		ClusterID:   w.clusterID,
		Labels:      obj.GetLabels(),
		Annotations: obj.GetAnnotations(),
		Raw:         obj,
		FetchedAt:   time.Now(),
	}

	for _, ref := range obj.GetOwnerReferences() {
		resource.OwnerRefs = append(resource.OwnerRefs, OwnerReference{
			Kind: ref.Kind,
			Name: ref.Name,
			UID:  string(ref.UID),
		})
	}

	return resource
}

// WatchPods starts watching pods
func (w *Watcher) WatchPods(ctx context.Context, namespace string) error {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	return w.Watch(ctx, gvr, namespace)
}

// WatchDeployments starts watching deployments
func (w *Watcher) WatchDeployments(ctx context.Context, namespace string) error {
	gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	return w.Watch(ctx, gvr, namespace)
}

// WatchServices starts watching services
func (w *Watcher) WatchServices(ctx context.Context, namespace string) error {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	return w.Watch(ctx, gvr, namespace)
}

// WatchEvents starts watching Kubernetes events
func (w *Watcher) WatchEvents(ctx context.Context, namespace string) error {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"}
	return w.Watch(ctx, gvr, namespace)
}
