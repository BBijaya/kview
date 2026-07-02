package graph

import (
	"testing"

	"github.com/bijaya/kview/internal/k8s"
)

func testDeployment(uid, name, ns string, replicas, ready, available int32) k8s.DeploymentInfo {
	return k8s.DeploymentInfo{
		Resource: k8s.Resource{
			UID:       uid,
			Kind:      "Deployment",
			Name:      name,
			Namespace: ns,
		},
		Replicas:          replicas,
		ReadyReplicas:     ready,
		AvailableReplicas: available,
	}
}

func testReplicaSet(uid, name, ns string, desired, ready int32, owners ...k8s.OwnerReference) k8s.ReplicaSetInfo {
	return k8s.ReplicaSetInfo{
		Resource: k8s.Resource{
			UID:       uid,
			Kind:      "ReplicaSet",
			Name:      name,
			Namespace: ns,
			OwnerRefs: owners,
		},
		DesiredReplicas: desired,
		ReadyReplicas:   ready,
	}
}

func testPod(uid, name, ns string, labels map[string]string, owners ...k8s.OwnerReference) k8s.PodInfo {
	return k8s.PodInfo{
		Resource: k8s.Resource{
			UID:       uid,
			Kind:      "Pod",
			Name:      name,
			Namespace: ns,
			Labels:    labels,
			OwnerRefs: owners,
		},
		Phase: "Running",
		Containers: []k8s.ContainerInfo{
			{Name: "app", Ready: true, State: "running"},
		},
	}
}

func testService(uid, name, ns string, selector map[string]string) k8s.ServiceInfo {
	return k8s.ServiceInfo{
		Resource: k8s.Resource{
			UID:       uid,
			Kind:      "Service",
			Name:      name,
			Namespace: ns,
		},
		Selector: selector,
	}
}

func TestBuilderOwnershipChain(t *testing.T) {
	depOwner := k8s.OwnerReference{Kind: "Deployment", Name: "web", UID: "dep-1"}
	rsOwner := k8s.OwnerReference{Kind: "ReplicaSet", Name: "web-abc", UID: "rs-1"}

	b := NewBuilder()
	// Parents must be added before children (see graph.go tier ordering).
	b.AddDeployments([]k8s.DeploymentInfo{testDeployment("dep-1", "web", "default", 2, 2, 2)})
	b.AddReplicaSets([]k8s.ReplicaSetInfo{testReplicaSet("rs-1", "web-abc", "default", 2, 2, depOwner)})
	b.AddPods([]k8s.PodInfo{
		testPod("pod-1", "web-abc-1", "default", map[string]string{"app": "web"}, rsOwner),
		testPod("pod-2", "web-abc-2", "default", map[string]string{"app": "web"}, rsOwner),
	})
	g := b.Build()

	nodes, edges := g.Size()
	if nodes != 4 {
		t.Errorf("Size() nodes = %d, want 4", nodes)
	}
	if edges != 3 {
		t.Errorf("Size() edges = %d, want 3", edges)
	}

	wantNodes := []struct {
		uid      string
		kind     string
		name     string
		status   NodeStatus
		wantInfo string
	}{
		{"dep-1", "Deployment", "web", StatusHealthy, "2/2/0"},
		{"rs-1", "ReplicaSet", "web-abc", StatusHealthy, "2/2"},
		{"pod-1", "Pod", "web-abc-1", StatusHealthy, "1/1"},
		{"pod-2", "Pod", "web-abc-2", StatusHealthy, "1/1"},
	}
	for _, want := range wantNodes {
		node := g.GetNode(want.uid)
		if node == nil {
			t.Errorf("GetNode(%q) = nil, want node", want.uid)
			continue
		}
		if node.Kind != want.kind {
			t.Errorf("node %s Kind = %q, want %q", want.uid, node.Kind, want.kind)
		}
		if node.Name != want.name {
			t.Errorf("node %s Name = %q, want %q", want.uid, node.Name, want.name)
		}
		if node.Status != want.status {
			t.Errorf("node %s Status = %q, want %q", want.uid, node.Status, want.status)
		}
		if got := node.Extras["info"]; got != want.wantInfo {
			t.Errorf("node %s Extras[info] = %q, want %q", want.uid, got, want.wantInfo)
		}
	}

	wantEdges := []struct {
		from, to string
	}{
		{"dep-1", "rs-1"},
		{"rs-1", "pod-1"},
		{"rs-1", "pod-2"},
	}
	for _, want := range wantEdges {
		if rel := g.GetEdgeRelation(want.from, want.to); rel != RelationOwns {
			t.Errorf("GetEdgeRelation(%q, %q) = %q, want %q", want.from, want.to, rel, RelationOwns)
		}
	}

	owned := g.GetOwnedChildren("rs-1")
	if len(owned) != 2 {
		t.Errorf("GetOwnedChildren(rs-1) returned %d nodes, want 2", len(owned))
	}
}

func TestBuilderEdgeRequiresBothNodes(t *testing.T) {
	// AddEdge silently drops edges when either endpoint is missing, which is
	// why graph.go adds parents before children. Adding children first means
	// the ownership edges are lost.
	depOwner := k8s.OwnerReference{Kind: "Deployment", Name: "web", UID: "dep-1"}
	rsOwner := k8s.OwnerReference{Kind: "ReplicaSet", Name: "web-abc", UID: "rs-1"}

	b := NewBuilder()
	b.AddPods([]k8s.PodInfo{testPod("pod-1", "web-abc-1", "default", nil, rsOwner)})
	b.AddReplicaSets([]k8s.ReplicaSetInfo{testReplicaSet("rs-1", "web-abc", "default", 1, 1, depOwner)})
	b.AddDeployments([]k8s.DeploymentInfo{testDeployment("dep-1", "web", "default", 1, 1, 1)})
	g := b.Build()

	nodes, edges := g.Size()
	if nodes != 3 {
		t.Errorf("Size() nodes = %d, want 3", nodes)
	}
	if edges != 0 {
		t.Errorf("Size() edges = %d, want 0 (owner edges dropped when parent added after child)", edges)
	}
}

func TestBuilderPodStatus(t *testing.T) {
	tests := []struct {
		name       string
		phase      string
		containers []k8s.ContainerInfo
		want       NodeStatus
	}{
		{
			name:       "running all ready",
			phase:      "Running",
			containers: []k8s.ContainerInfo{{Name: "app", Ready: true}},
			want:       StatusHealthy,
		},
		{
			name:       "running container not ready",
			phase:      "Running",
			containers: []k8s.ContainerInfo{{Name: "app", Ready: false}},
			want:       StatusWarning,
		},
		{
			name:  "running crashloop container not ready",
			phase: "Running",
			containers: []k8s.ContainerInfo{
				{Name: "app", Ready: false, StateReason: "CrashLoopBackOff"},
			},
			want: StatusError,
		},
		{
			name:  "running crashloop container ready",
			phase: "Running",
			containers: []k8s.ContainerInfo{
				{Name: "app", Ready: true, StateReason: "CrashLoopBackOff"},
			},
			want: StatusError,
		},
		{
			name:  "running oomkilled container not ready",
			phase: "Running",
			containers: []k8s.ContainerInfo{
				{Name: "app", Ready: false, StateReason: "OOMKilled"},
			},
			want: StatusError,
		},
		{
			name:  "error in later container wins over earlier not ready",
			phase: "Running",
			containers: []k8s.ContainerInfo{
				{Name: "sidecar", Ready: false},
				{Name: "app", Ready: false, StateReason: "CrashLoopBackOff"},
			},
			want: StatusError,
		},
		{
			name:  "pending pod",
			phase: "Pending",
			want:  StatusPending,
		},
		{
			name:  "failed pod",
			phase: "Failed",
			want:  StatusError,
		},
		{
			name:  "succeeded pod",
			phase: "Succeeded",
			want:  StatusHealthy,
		},
		{
			name:  "unknown phase",
			phase: "Evicted",
			want:  StatusUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := k8s.PodInfo{
				Resource:   k8s.Resource{UID: "pod-1", Name: "p", Namespace: "default"},
				Phase:      tt.phase,
				Containers: tt.containers,
			}
			b := NewBuilder()
			b.AddPods([]k8s.PodInfo{pod})
			node := b.Build().GetNode("pod-1")
			if node == nil {
				t.Fatal("GetNode(pod-1) = nil, want node")
			}
			if node.Status != tt.want {
				t.Errorf("pod status = %q, want %q", node.Status, tt.want)
			}
		})
	}
}

func TestBuilderDeploymentStatus(t *testing.T) {
	tests := []struct {
		name     string
		replicas int32
		ready    int32
		want     NodeStatus
	}{
		{name: "all ready", replicas: 3, ready: 3, want: StatusHealthy},
		{name: "partially ready", replicas: 3, ready: 1, want: StatusWarning},
		{name: "none ready", replicas: 3, ready: 0, want: StatusError},
		{name: "scaled to zero", replicas: 0, ready: 0, want: StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuilder()
			dep := testDeployment("dep-1", "web", "default", tt.replicas, tt.ready, tt.ready)
			b.AddDeployments([]k8s.DeploymentInfo{dep})
			node := b.Build().GetNode("dep-1")
			if node == nil {
				t.Fatal("GetNode(dep-1) = nil, want node")
			}
			if node.Status != tt.want {
				t.Errorf("deployment status = %q, want %q", node.Status, tt.want)
			}
		})
	}
}

func TestLinkServicesToPods(t *testing.T) {
	tests := []struct {
		name      string
		selector  map[string]string
		svcNS     string
		podLabels map[string]string
		podNS     string
		wantEdge  bool
	}{
		{
			name:      "exact label match",
			selector:  map[string]string{"app": "web"},
			svcNS:     "default",
			podLabels: map[string]string{"app": "web"},
			podNS:     "default",
			wantEdge:  true,
		},
		{
			name:      "selector subset of pod labels",
			selector:  map[string]string{"app": "web"},
			svcNS:     "default",
			podLabels: map[string]string{"app": "web", "tier": "frontend"},
			podNS:     "default",
			wantEdge:  true,
		},
		{
			name:      "multi-key selector all match",
			selector:  map[string]string{"app": "web", "tier": "frontend"},
			svcNS:     "default",
			podLabels: map[string]string{"app": "web", "tier": "frontend"},
			podNS:     "default",
			wantEdge:  true,
		},
		{
			name:      "label value mismatch",
			selector:  map[string]string{"app": "web"},
			svcNS:     "default",
			podLabels: map[string]string{"app": "api"},
			podNS:     "default",
			wantEdge:  false,
		},
		{
			name:      "selector key missing from pod labels",
			selector:  map[string]string{"app": "web", "tier": "frontend"},
			svcNS:     "default",
			podLabels: map[string]string{"app": "web"},
			podNS:     "default",
			wantEdge:  false,
		},
		{
			name:      "namespace mismatch",
			selector:  map[string]string{"app": "web"},
			svcNS:     "default",
			podLabels: map[string]string{"app": "web"},
			podNS:     "prod",
			wantEdge:  false,
		},
		{
			name:      "empty selector selects nothing",
			selector:  map[string]string{},
			svcNS:     "default",
			podLabels: map[string]string{"app": "web"},
			podNS:     "default",
			wantEdge:  false,
		},
		{
			name:      "pod with no labels",
			selector:  map[string]string{"app": "web"},
			svcNS:     "default",
			podLabels: nil,
			podNS:     "default",
			wantEdge:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := testService("svc-1", "web-svc", tt.svcNS, tt.selector)
			pod := testPod("pod-1", "web-1", tt.podNS, tt.podLabels)

			b := NewBuilder()
			b.AddServices([]k8s.ServiceInfo{svc})
			b.AddPods([]k8s.PodInfo{pod})
			b.LinkServicesToPods([]k8s.ServiceInfo{svc}, []k8s.PodInfo{pod})
			g := b.Build()

			rel := g.GetEdgeRelation("svc-1", "pod-1")
			if tt.wantEdge && rel != RelationSelects {
				t.Errorf("GetEdgeRelation(svc-1, pod-1) = %q, want %q", rel, RelationSelects)
			}
			if !tt.wantEdge && rel != "" {
				t.Errorf("GetEdgeRelation(svc-1, pod-1) = %q, want no edge", rel)
			}
		})
	}
}

func TestBuilderCalculateDepths(t *testing.T) {
	depOwner := k8s.OwnerReference{Kind: "Deployment", Name: "web", UID: "dep-1"}
	rsOwner := k8s.OwnerReference{Kind: "ReplicaSet", Name: "web-abc", UID: "rs-1"}

	b := NewBuilder()
	b.AddDeployments([]k8s.DeploymentInfo{testDeployment("dep-1", "web", "default", 1, 1, 1)})
	b.AddReplicaSets([]k8s.ReplicaSetInfo{testReplicaSet("rs-1", "web-abc", "default", 1, 1, depOwner)})
	b.AddPods([]k8s.PodInfo{testPod("pod-1", "web-abc-1", "default", nil, rsOwner)})
	b.CalculateDepths()
	g := b.Build()

	wantDepths := map[string]int{"dep-1": 0, "rs-1": 1, "pod-1": 2}
	for uid, want := range wantDepths {
		node := g.GetNode(uid)
		if node == nil {
			t.Fatalf("GetNode(%q) = nil, want node", uid)
		}
		if node.Depth != want {
			t.Errorf("node %s Depth = %d, want %d", uid, node.Depth, want)
		}
	}
}
