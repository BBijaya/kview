package graph

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bijaya/kview/internal/k8s"
)

// buildStubClient embeds DisconnectedClient (every method errors → empty
// lists, which Build tolerates) and overrides a few List methods to return
// fixtures while recording how many list calls run concurrently.
type buildStubClient struct {
	*k8s.DisconnectedClient

	mu          sync.Mutex
	inflight    int
	maxInflight int
}

func (c *buildStubClient) enter() {
	c.mu.Lock()
	c.inflight++
	if c.inflight > c.maxInflight {
		c.maxInflight = c.inflight
	}
	c.mu.Unlock()
	time.Sleep(10 * time.Millisecond) // widen the overlap window
}

func (c *buildStubClient) exit() {
	c.mu.Lock()
	c.inflight--
	c.mu.Unlock()
}

func (c *buildStubClient) ListDeployments(ctx context.Context, ns string) ([]k8s.DeploymentInfo, error) {
	c.enter()
	defer c.exit()
	return []k8s.DeploymentInfo{{
		Resource: k8s.Resource{UID: "dep-1", Kind: "Deployment", Name: "web", Namespace: "default"},
	}}, nil
}

func (c *buildStubClient) ListReplicaSets(ctx context.Context, ns string) ([]k8s.ReplicaSetInfo, error) {
	c.enter()
	defer c.exit()
	return []k8s.ReplicaSetInfo{{
		Resource: k8s.Resource{
			UID: "rs-1", Kind: "ReplicaSet", Name: "web-abc", Namespace: "default",
			OwnerRefs: []k8s.OwnerReference{{UID: "dep-1", Kind: "Deployment", Name: "web"}},
		},
	}}, nil
}

func (c *buildStubClient) ListPods(ctx context.Context, ns string) ([]k8s.PodInfo, error) {
	c.enter()
	defer c.exit()
	return []k8s.PodInfo{{
		Resource: k8s.Resource{
			UID: "pod-1", Kind: "Pod", Name: "web-abc-1", Namespace: "default",
			OwnerRefs: []k8s.OwnerReference{{UID: "rs-1", Kind: "ReplicaSet", Name: "web-abc"}},
		},
		Phase: "Running",
	}}, nil
}

func (c *buildStubClient) ListServices(ctx context.Context, ns string) ([]k8s.ServiceInfo, error) {
	c.enter()
	defer c.exit()
	return []k8s.ServiceInfo{{
		Resource: k8s.Resource{UID: "svc-1", Kind: "Service", Name: "web-svc", Namespace: "default"},
	}}, nil
}

func (c *buildStubClient) ListConfigMaps(ctx context.Context, ns string) ([]k8s.ConfigMapInfo, error) {
	c.enter()
	defer c.exit()
	return nil, nil
}

func (c *buildStubClient) ListSecrets(ctx context.Context, ns string) ([]k8s.SecretInfo, error) {
	c.enter()
	defer c.exit()
	return nil, nil
}

func TestBuildProducesGraphFromConcurrentFetches(t *testing.T) {
	stub := &buildStubClient{DisconnectedClient: k8s.NewDisconnectedClient("stub")}
	rg := NewResourceGraph(stub)

	if err := rg.Build(context.Background(), "default"); err != nil {
		t.Fatalf("Build: %v", err)
	}

	g := rg.GetGraph()
	// Ownership chain intact despite concurrent fetching (adds are ordered)
	for _, uid := range []string{"dep-1", "rs-1", "pod-1", "svc-1"} {
		if g.GetNode(uid) == nil {
			t.Errorf("node %s missing from built graph", uid)
		}
	}
	if g.GetEdgeRelation("dep-1", "rs-1") != RelationOwns {
		t.Error("Deployment→ReplicaSet ownership edge missing")
	}
	if g.GetEdgeRelation("rs-1", "pod-1") != RelationOwns {
		t.Error("ReplicaSet→Pod ownership edge missing")
	}

	if stub.maxInflight < 2 {
		t.Errorf("max concurrent list calls = %d, want > 1 (fetches should overlap)", stub.maxInflight)
	}
}
