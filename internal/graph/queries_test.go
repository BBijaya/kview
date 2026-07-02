package graph

import (
	"reflect"
	"sort"
	"testing"
)

// fixtureGraph builds a small graph resembling a typical workload:
//
//	dep-1 (Deployment) ─owns→ rs-1 (ReplicaSet) ─owns→ pod-1, pod-2 (Pods)
//	svc-1 (Service) ─selects→ pod-1
//	orphan-1 (ConfigMap) — disconnected
func fixtureGraph() *Graph {
	g := New()
	g.AddNode(&Node{UID: "dep-1", Kind: "Deployment", Name: "web", Namespace: "default", Status: StatusHealthy})
	g.AddNode(&Node{UID: "rs-1", Kind: "ReplicaSet", Name: "web-abc", Namespace: "default", Status: StatusHealthy})
	g.AddNode(&Node{UID: "pod-1", Kind: "Pod", Name: "web-abc-1", Namespace: "default", Status: StatusHealthy})
	g.AddNode(&Node{UID: "pod-2", Kind: "Pod", Name: "web-abc-2", Namespace: "default", Status: StatusError})
	g.AddNode(&Node{UID: "svc-1", Kind: "Service", Name: "web-svc", Namespace: "default", Status: StatusHealthy})
	g.AddNode(&Node{UID: "orphan-1", Kind: "ConfigMap", Name: "lonely", Namespace: "default", Status: StatusHealthy})

	g.AddEdge("dep-1", "rs-1", RelationOwns)
	g.AddEdge("rs-1", "pod-1", RelationOwns)
	g.AddEdge("rs-1", "pod-2", RelationOwns)
	g.AddEdge("svc-1", "pod-1", RelationSelects)
	return g
}

func sortedUIDs(nodes []*Node) []string {
	uids := make([]string, 0, len(nodes))
	for _, n := range nodes {
		uids = append(uids, n.UID)
	}
	sort.Strings(uids)
	return uids
}

func pathUIDs(nodes []*Node) []string {
	uids := make([]string, 0, len(nodes))
	for _, n := range nodes {
		uids = append(uids, n.UID)
	}
	return uids
}

func TestGetOwnerChain(t *testing.T) {
	tests := []struct {
		name string
		uid  string
		want []string // UIDs in order, root first
	}{
		{
			name: "pod walks up through replicaset to deployment",
			uid:  "pod-2",
			want: []string{"dep-1", "rs-1", "pod-2"},
		},
		{
			name: "root node returns just itself",
			uid:  "dep-1",
			want: []string{"dep-1"},
		},
		{
			name: "unknown uid returns empty chain",
			uid:  "nope",
			want: nil,
		},
	}

	q := NewQuery(fixtureGraph())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathUIDs(q.GetOwnerChain(tt.uid))
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetOwnerChain(%q) = %v, want %v", tt.uid, got, tt.want)
			}
		})
	}
}

func TestGetDescendants(t *testing.T) {
	tests := []struct {
		name string
		uid  string
		want []string // sorted UIDs
	}{
		{
			name: "deployment reaches replicaset and pods",
			uid:  "dep-1",
			want: []string{"pod-1", "pod-2", "rs-1"},
		},
		{
			name: "service reaches selected pod",
			uid:  "svc-1",
			want: []string{"pod-1"},
		},
		{
			name: "leaf pod has no descendants",
			uid:  "pod-1",
			want: nil,
		},
		{
			name: "unknown uid has no descendants",
			uid:  "nope",
			want: nil,
		},
	}

	q := NewQuery(fixtureGraph())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortedUIDs(q.GetDescendants(tt.uid))
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDescendants(%q) = %v, want %v", tt.uid, got, tt.want)
			}
		})
	}
}

func TestGetAncestors(t *testing.T) {
	tests := []struct {
		name string
		uid  string
		want []string // sorted UIDs
	}{
		{
			// Ancestors follow all incoming edges, not just ownership,
			// so the selecting Service is included alongside the owners.
			name: "pod ancestors include owners and selecting service",
			uid:  "pod-1",
			want: []string{"dep-1", "rs-1", "svc-1"},
		},
		{
			name: "root has no ancestors",
			uid:  "dep-1",
			want: nil,
		},
		{
			name: "unknown uid has no ancestors",
			uid:  "nope",
			want: nil,
		},
	}

	q := NewQuery(fixtureGraph())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortedUIDs(q.GetAncestors(tt.uid))
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAncestors(%q) = %v, want %v", tt.uid, got, tt.want)
			}
		})
	}
}

func TestFindPath(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		want []string // UIDs in path order, nil for no path
	}{
		{
			name: "downward path deployment to pod",
			from: "dep-1",
			to:   "pod-1",
			want: []string{"dep-1", "rs-1", "pod-1"},
		},
		{
			name: "upward path pod to deployment (search is undirected)",
			from: "pod-1",
			to:   "dep-1",
			want: []string{"pod-1", "rs-1", "dep-1"},
		},
		{
			name: "lateral path service to deployment via pod",
			from: "svc-1",
			to:   "dep-1",
			want: []string{"svc-1", "pod-1", "rs-1", "dep-1"},
		},
		{
			name: "same node returns single-node path",
			from: "dep-1",
			to:   "dep-1",
			want: []string{"dep-1"},
		},
		{
			name: "no path to disconnected node",
			from: "dep-1",
			to:   "orphan-1",
			want: nil,
		},
		{
			name: "unknown source returns nil",
			from: "nope",
			to:   "dep-1",
			want: nil,
		},
		{
			name: "unknown target returns nil",
			from: "dep-1",
			to:   "nope",
			want: nil,
		},
		{
			name: "unknown node to itself returns nil",
			from: "nope",
			to:   "nope",
			want: nil,
		},
	}

	q := NewQuery(fixtureGraph())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathUIDs(q.FindPath(tt.from, tt.to))
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindPath(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestGetSubgraph(t *testing.T) {
	q := NewQuery(fixtureGraph())
	sub := q.GetSubgraph("dep-1")

	wantNodes := []string{"dep-1", "pod-1", "pod-2", "rs-1"}
	gotNodes := make([]string, 0, len(sub.Nodes))
	for uid := range sub.Nodes {
		gotNodes = append(gotNodes, uid)
	}
	sort.Strings(gotNodes)
	if !reflect.DeepEqual(gotNodes, wantNodes) {
		t.Errorf("GetSubgraph(dep-1) nodes = %v, want %v", gotNodes, wantNodes)
	}

	// svc-1 is not a descendant, so its selects edge must be excluded.
	if len(sub.Edges) != 3 {
		t.Errorf("GetSubgraph(dep-1) has %d edges, want 3", len(sub.Edges))
	}
	if rel := sub.GetEdgeRelation("svc-1", "pod-1"); rel != "" {
		t.Errorf("GetSubgraph(dep-1) contains svc edge %q, want none", rel)
	}
}

func TestFindByKindAndStatus(t *testing.T) {
	q := NewQuery(fixtureGraph())

	if got := q.FindByKind("Pod"); len(got) != 2 {
		t.Errorf("FindByKind(Pod) returned %d nodes, want 2", len(got))
	}
	if got := q.FindByKind("Ingress"); len(got) != 0 {
		t.Errorf("FindByKind(Ingress) returned %d nodes, want 0", len(got))
	}
	if got := q.FindByStatus(StatusError); len(got) != 1 || got[0].UID != "pod-2" {
		t.Errorf("FindByStatus(error) = %v, want [pod-2]", sortedUIDs(got))
	}
	if got := q.FindUnhealthy(); len(got) != 1 || got[0].UID != "pod-2" {
		t.Errorf("FindUnhealthy() = %v, want [pod-2]", sortedUIDs(got))
	}
}
