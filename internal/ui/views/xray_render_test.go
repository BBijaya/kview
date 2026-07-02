package views

import (
	"testing"

	"github.com/bijaya/kview/internal/graph"
)

func xrayServiceFixture() *graph.Graph {
	g := graph.New()
	g.AddNode(&graph.Node{UID: "svc-1", Kind: "Service", Name: "web-svc", Namespace: "default", Status: graph.StatusHealthy})
	g.AddNode(&graph.Node{UID: "svc-2", Kind: "Service", Name: "web-svc-headless", Namespace: "default", Status: graph.StatusHealthy})
	g.AddNode(&graph.Node{UID: "pod-1", Kind: "Pod", Name: "web-1", Namespace: "default", Status: graph.StatusHealthy})
	g.AddNode(&graph.Node{UID: "pod-2", Kind: "Pod", Name: "web-2", Namespace: "default", Status: graph.StatusHealthy})
	g.AddEdge("svc-1", "pod-1", graph.RelationSelects)
	g.AddEdge("svc-1", "pod-2", graph.RelationSelects)
	g.AddEdge("svc-2", "pod-1", graph.RelationSelects) // pod shared by both services
	return g
}

func xrayTypeView(g *graph.Graph, rootKind string, expanded map[string]bool) *XrayView {
	return &XrayView{
		graph:    g,
		mode:     xrayModeType,
		rootKind: rootKind,
		expanded: expanded,
	}
}

// childrenUnder returns the names of nodes rendered directly under the node
// with the given name (depth = parent depth + 1, until depth returns to it).
func childrenUnder(nodes []*xrayNode, parentName string) []string {
	var out []string
	parentDepth := -1
	for _, n := range nodes {
		if parentDepth >= 0 {
			if n.depth <= parentDepth {
				break
			}
			if n.depth == parentDepth+1 {
				out = append(out, n.name)
			}
			continue
		}
		if n.name == parentName {
			parentDepth = n.depth
		}
	}
	return out
}

func TestXraySvcShowsPodsUnderEveryService(t *testing.T) {
	v := xrayTypeView(xrayServiceFixture(), "Service", map[string]bool{
		"ns/default": true, "svc-1": true, "svc-2": true,
	})
	nodes := v.flattenTree()

	if got := childrenUnder(nodes, "web-svc"); len(got) != 2 {
		t.Errorf("web-svc children = %v, want its 2 selected pods", got)
	}
	// The regression: pod-1 is shared, and previously rendered only under
	// the first service while the second showed a phantom child count.
	if got := childrenUnder(nodes, "web-svc-headless"); len(got) != 1 || got[0] != "web-1" {
		t.Errorf("web-svc-headless children = %v, want [web-1] (shared pod must render under both services)", got)
	}

	// Child counts must match what actually renders
	for _, n := range nodes {
		if n.kind == "Service" && n.isExpanded {
			rendered := len(childrenUnder(nodes, n.name))
			if n.childCount != rendered {
				t.Errorf("%s childCount = %d but %d children rendered", n.name, n.childCount, rendered)
			}
		}
	}
}

func TestXrayFlattenCycleSafeWithinSubtree(t *testing.T) {
	g := graph.New()
	g.AddNode(&graph.Node{UID: "a", Kind: "Service", Name: "a", Namespace: "default"})
	g.AddNode(&graph.Node{UID: "b", Kind: "Thing", Name: "b", Namespace: "default"})
	g.AddEdge("a", "b", graph.RelationSelects)
	g.AddEdge("b", "a", graph.RelationSelects) // cycle

	v := xrayTypeView(g, "Service", map[string]bool{"ns/default": true, "a": true, "b": true})
	nodes := v.flattenTree() // must terminate
	if len(nodes) == 0 {
		t.Fatal("expected nodes from cyclic graph flatten")
	}
}
