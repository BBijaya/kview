package views

import (
	"testing"

	"github.com/bijaya/kview/internal/graph"
)

// xrayNavFixture: dep-1 owns rs-1 owns pod-1; pod-1 mounts cm-1.
func xrayNavFixture() *graph.Graph {
	g := graph.New()
	g.AddNode(&graph.Node{UID: "dep-1", Kind: "Deployment", Name: "web", Namespace: "default", Status: graph.StatusHealthy})
	g.AddNode(&graph.Node{UID: "rs-1", Kind: "ReplicaSet", Name: "web-abc", Namespace: "default", Status: graph.StatusHealthy})
	g.AddNode(&graph.Node{UID: "pod-1", Kind: "Pod", Name: "web-abc-1", Namespace: "default", Status: graph.StatusHealthy})
	g.AddNode(&graph.Node{UID: "cm-1", Kind: "ConfigMap", Name: "web-config", Namespace: "default", Status: graph.StatusHealthy})
	g.AddEdge("dep-1", "rs-1", graph.RelationOwns)
	g.AddEdge("rs-1", "pod-1", graph.RelationOwns)
	g.AddEdge("pod-1", "cm-1", graph.RelationMounts)
	return g
}

func findFlatNode(nodes []*xrayNode, name string) *xrayNode {
	for _, n := range nodes {
		if n.name == name {
			return n
		}
	}
	return nil
}

func TestXrayDrillIntoLeafAndPopBack(t *testing.T) {
	v := NewXrayView(nil)
	v.graph = xrayNavFixture()
	v.mode = xrayModeType
	v.rootKind = "Deployment"
	v.initExpanded()
	v.rebuildTable()

	originalExpanded := v.expanded
	cm := findFlatNode(v.flatNodes, "web-config")
	if cm == nil {
		t.Fatalf("fixture: ConfigMap not rendered in deployment tree; nodes: %v", nodeNames(v.flatNodes))
	}
	if !v.isDrillable(cm) {
		t.Fatal("ConfigMap leaf should be drillable")
	}

	v.drillInto(cm)

	if v.mode != xrayModeResource {
		t.Errorf("mode after drill = %v, want resource mode", v.mode)
	}
	if v.focusUID != "cm-1" {
		t.Errorf("focusUID after drill = %q, want cm-1", v.focusUID)
	}
	if len(v.navStack) != 1 {
		t.Fatalf("navStack length = %d, want 1", len(v.navStack))
	}
	if findFlatNode(v.flatNodes, "web-config") == nil {
		t.Error("drilled view should render the focused ConfigMap")
	}

	if !v.popNav() {
		t.Fatal("popNav returned false with a non-empty trail")
	}
	if v.mode != xrayModeType || v.rootKind != "Deployment" {
		t.Errorf("after pop: mode=%v rootKind=%q, want type mode / Deployment", v.mode, v.rootKind)
	}
	if len(v.navStack) != 0 {
		t.Errorf("navStack length after pop = %d, want 0", len(v.navStack))
	}
	// Expansion state must be the exact pre-drill map (identity restore)
	if &originalExpanded == nil || len(v.expanded) != len(originalExpanded) {
		t.Errorf("expanded state not restored: %d entries, want %d", len(v.expanded), len(originalExpanded))
	}
	if v.popNav() {
		t.Error("popNav on empty trail should return false (Escape then leaves the view)")
	}
}

func TestXrayDrillableRules(t *testing.T) {
	v := NewXrayView(nil)
	v.focusUID = "dep-1"
	cases := []struct {
		name string
		node *xrayNode
		want bool
	}{
		{"namespace header", &xrayNode{uid: "ns/default", kind: "Namespace", isNsHeader: true}, false},
		{"section header", &xrayNode{uid: "section/x", isSectionHeader: true}, false},
		{"owner hint", &xrayNode{uid: "hint/x", isOwnerHint: true}, false},
		{"container", &xrayNode{uid: "pod-1/co/app", kind: "Container", name: "app"}, false},
		{"current focus", &xrayNode{uid: "dep-1", kind: "Deployment", name: "web"}, false},
		{"pod", &xrayNode{uid: "pod-1", kind: "Pod", name: "web-abc-1"}, true},
		{"configmap", &xrayNode{uid: "cm-1", kind: "ConfigMap", name: "cfg"}, true},
		{"node", &xrayNode{uid: "node-1", kind: "Node", name: "worker-1"}, true},
	}
	for _, tc := range cases {
		if got := v.isDrillable(tc.node); got != tc.want {
			t.Errorf("isDrillable(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestXraySetModeClearsDrillTrail(t *testing.T) {
	v := NewXrayView(nil)
	v.graph = xrayNavFixture()
	v.mode = xrayModeType
	v.rootKind = "Deployment"
	v.initExpanded()
	v.rebuildTable()

	if cm := findFlatNode(v.flatNodes, "web-config"); cm != nil {
		v.drillInto(cm)
	}
	if len(v.navStack) == 0 {
		t.Fatal("fixture: expected a drill to have happened")
	}

	if err := v.SetMode("deploy"); err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	if len(v.navStack) != 0 {
		t.Errorf("navStack after SetMode = %d entries, want 0 (fresh entry point)", len(v.navStack))
	}
}

func nodeNames(nodes []*xrayNode) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.kind+"/"+n.name)
	}
	return out
}
