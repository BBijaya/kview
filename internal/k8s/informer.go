package k8s

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ListFunc is a function that lists resources from the API server.
// It returns the typed data (e.g. []PodInfo) and an error.
type ListFunc func(ctx context.Context, client Client, namespace string) (any, error)

// ResourceInformer maintains a background cache for a single resource kind.
// It runs List+Watch in a background goroutine and exposes a thread-safe
// Snapshot for the UI to poll on a timer. Watch events never enter the
// Bubble Tea Update loop — they only set a dirty flag that triggers a re-List.
type ResourceInformer struct {
	mu          sync.RWMutex
	client      Client
	kind        string // "pods", "deployments", etc.
	namespace   string
	data        any    // []PodInfo, []DeploymentInfo, etc.
	generation  uint64
	lastError   error
	dirty       int32 // atomic: 1=needs re-list
	listFunc    ListFunc
	stopCh      chan struct{}
	watchCancel context.CancelFunc
	started     bool
}

// NewResourceInformer creates a new informer for the given resource kind.
func NewResourceInformer(client Client, kind string, listFunc ListFunc) *ResourceInformer {
	return &ResourceInformer{
		client:   client,
		kind:     kind,
		listFunc: listFunc,
	}
}

// Start begins the informer's background goroutines for the given namespace.
// It performs an initial List, then starts the watch and refresh loops.
func (ri *ResourceInformer) Start(namespace string) {
	ri.mu.Lock()
	if ri.started {
		ri.mu.Unlock()
		return
	}
	ri.namespace = namespace
	ri.started = true
	ri.stopCh = make(chan struct{})
	ri.mu.Unlock()

	// Initial list (synchronous-ish in a goroutine so Start doesn't block)
	go func() {
		ri.listAndStore()

		// Start background loops
		ctx, cancel := context.WithCancel(context.Background())
		ri.mu.Lock()
		ri.watchCancel = cancel
		ri.mu.Unlock()

		go ri.watchLoop(ctx)
		go ri.refreshLoop(ctx)
	}()
}

// Stop shuts down the informer's background goroutines.
func (ri *ResourceInformer) Stop() {
	ri.mu.Lock()
	defer ri.mu.Unlock()

	if !ri.started {
		return
	}
	ri.started = false

	if ri.watchCancel != nil {
		ri.watchCancel()
		ri.watchCancel = nil
	}
	close(ri.stopCh)
}

// SetNamespace stops the current informer and restarts with a new namespace.
func (ri *ResourceInformer) SetNamespace(ns string) {
	ri.Stop()
	// Reset generation so the UI knows data is stale
	ri.mu.Lock()
	ri.data = nil
	ri.generation = 0
	ri.lastError = nil
	ri.mu.Unlock()
	atomic.StoreInt32(&ri.dirty, 0)
	ri.Start(ns)
}

// SetClient updates the client reference (used after context switch).
func (ri *ResourceInformer) SetClient(client Client) {
	ri.mu.Lock()
	ri.client = client
	ri.mu.Unlock()
}

// Invalidate marks the informer's data as dirty so the next refresh cycle
// will re-list from the API server.
func (ri *ResourceInformer) Invalidate() {
	atomic.StoreInt32(&ri.dirty, 1)
}

// Snapshot returns a thread-safe copy of the current data, generation counter,
// and last error.
func (ri *ResourceInformer) Snapshot() (data any, gen uint64, err error) {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	return ri.data, ri.generation, ri.lastError
}

// Generation returns the current generation counter.
func (ri *ResourceInformer) Generation() uint64 {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	return ri.generation
}

// Started returns whether the informer is currently running.
func (ri *ResourceInformer) Started() bool {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	return ri.started
}

// listAndStore calls the listFunc, stores the result, and increments generation.
func (ri *ResourceInformer) listAndStore() {
	ri.mu.RLock()
	client := ri.client
	ns := ri.namespace
	ri.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	data, err := ri.listFunc(ctx, client, ns)

	ri.mu.Lock()
	if err != nil {
		ri.lastError = err
	} else {
		ri.data = data
		ri.lastError = nil
		ri.generation++
	}
	ri.mu.Unlock()

	atomic.StoreInt32(&ri.dirty, 0)
}

// watchLoop watches for resource changes and sets the dirty flag.
// It reconnects with exponential backoff on failure.
func (ri *ResourceInformer) watchLoop(ctx context.Context) {
	backoff := time.Second

	for {
		select {
		case <-ctx.Done():
			return
		case <-ri.stopCh:
			return
		default:
		}

		ri.mu.RLock()
		client := ri.client
		ns := ri.namespace
		ri.mu.RUnlock()

		ch, err := client.Watch(ctx, ri.kind, ns)
		if err != nil {
			// Watch not supported or failed — wait and retry
			select {
			case <-ctx.Done():
				return
			case <-ri.stopCh:
				return
			case <-time.After(backoff):
				if backoff < 60*time.Second {
					backoff *= 2
				}
				continue
			}
		}

		// Reset backoff on successful connection
		backoff = time.Second

		// Drain events, just set dirty flag
		for {
			select {
			case <-ctx.Done():
				return
			case <-ri.stopCh:
				return
			case _, ok := <-ch:
				if !ok {
					// Watch channel closed, reconnect
					goto reconnect
				}
				atomic.StoreInt32(&ri.dirty, 1)
			}
		}

	reconnect:
		// Small delay before reconnecting
		select {
		case <-ctx.Done():
			return
		case <-ri.stopCh:
			return
		case <-time.After(time.Second):
		}
	}
}

// refreshLoop checks the dirty flag every 2 seconds and re-lists if needed.
func (ri *ResourceInformer) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ri.stopCh:
			return
		case <-ticker.C:
			if atomic.LoadInt32(&ri.dirty) == 1 {
				ri.listAndStore()
			}
		}
	}
}
