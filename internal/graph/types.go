package graph

import (
	"strings"

	"github.com/bijaya/kview/internal/k8s"
)

// Graph represents a resource relationship graph
type Graph struct {
	Nodes map[string]*Node
	Edges []*Edge
}

// Node represents a resource in the graph
type Node struct {
	UID       string
	Kind      string
	Name      string
	Namespace string
	Status    NodeStatus
	Resource  *k8s.Resource
	Depth     int               // Distance from root
	Extras    map[string]string // Additional display info (e.g., "info" for ready counts)
}

// NodeStatus represents the health status of a node
type NodeStatus string

const (
	StatusHealthy  NodeStatus = "healthy"
	StatusWarning  NodeStatus = "warning"
	StatusError    NodeStatus = "error"
	StatusUnknown  NodeStatus = "unknown"
	StatusPending  NodeStatus = "pending"
)

// Edge represents a relationship between nodes
type Edge struct {
	From     string   // UID of source node
	To       string   // UID of target node
	Relation Relation // Type of relationship
}

// Relation represents the type of relationship between resources
type Relation string

const (
	RelationOwns       Relation = "owns"       // Owner -> Owned (e.g., Deployment -> ReplicaSet)
	RelationSelects    Relation = "selects"     // Selector -> Selected (e.g., Service -> Pod)
	RelationRoutes     Relation = "routes"      // Router -> Target (e.g., Ingress -> Service)
	RelationMounts     Relation = "mounts"      // Consumer -> Volume (e.g., Pod -> ConfigMap)
	RelationUses       Relation = "uses"        // Generic usage relationship
	RelationBinds      Relation = "binds"       // PVC -> PV
	RelationTargets    Relation = "targets"     // HPA -> Deployment/STS
	RelationReferences Relation = "references"  // Ingress TLS -> Secret
)

// New creates a new empty graph
func New() *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
		Edges: make([]*Edge, 0),
	}
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(node *Node) {
	g.Nodes[node.UID] = node
}

// AddEdge adds an edge to the graph
func (g *Graph) AddEdge(from, to string, relation Relation) {
	// Verify both nodes exist
	if _, ok := g.Nodes[from]; !ok {
		return
	}
	if _, ok := g.Nodes[to]; !ok {
		return
	}

	edge := &Edge{
		From:     from,
		To:       to,
		Relation: relation,
	}
	g.Edges = append(g.Edges, edge)
}

// GetNode returns a node by UID
func (g *Graph) GetNode(uid string) *Node {
	return g.Nodes[uid]
}

// GetChildren returns all nodes that are targets of edges from the given node
func (g *Graph) GetChildren(uid string) []*Node {
	var children []*Node
	for _, edge := range g.Edges {
		if edge.From == uid {
			if child := g.Nodes[edge.To]; child != nil {
				children = append(children, child)
			}
		}
	}
	return children
}

// GetOwnedChildren returns only nodes connected via RelationOwns edges
func (g *Graph) GetOwnedChildren(uid string) []*Node {
	var children []*Node
	for _, edge := range g.Edges {
		if edge.From == uid && edge.Relation == RelationOwns {
			if child := g.Nodes[edge.To]; child != nil {
				children = append(children, child)
			}
		}
	}
	return children
}

// GetXrayChildren returns children for xray tree rendering: owned children first,
// then mounted/used resources as leaf nodes (matching k9s xray behavior).
func (g *Graph) GetXrayChildren(uid string) []*Node {
	var owned, refs []*Node
	seen := make(map[string]bool)
	for _, edge := range g.Edges {
		if edge.From != uid {
			continue
		}
		child := g.Nodes[edge.To]
		if child == nil || seen[child.UID] {
			continue
		}
		seen[child.UID] = true
		if edge.Relation == RelationOwns {
			owned = append(owned, child)
		} else {
			refs = append(refs, child)
		}
	}
	return append(owned, refs...)
}

// GetParents returns all nodes that have edges pointing to the given node
func (g *Graph) GetParents(uid string) []*Node {
	var parents []*Node
	for _, edge := range g.Edges {
		if edge.To == uid {
			if parent := g.Nodes[edge.From]; parent != nil {
				parents = append(parents, parent)
			}
		}
	}
	return parents
}

// GetRelatedNodes returns all nodes connected to the given node
func (g *Graph) GetRelatedNodes(uid string) []*Node {
	related := make(map[string]*Node)

	// Get children
	for _, child := range g.GetChildren(uid) {
		related[child.UID] = child
	}

	// Get parents
	for _, parent := range g.GetParents(uid) {
		related[parent.UID] = parent
	}

	var result []*Node
	for _, node := range related {
		result = append(result, node)
	}
	return result
}

// GetRoots returns nodes with no parents (top-level resources)
func (g *Graph) GetRoots() []*Node {
	hasParent := make(map[string]bool)
	for _, edge := range g.Edges {
		hasParent[edge.To] = true
	}

	var roots []*Node
	for uid, node := range g.Nodes {
		if !hasParent[uid] {
			roots = append(roots, node)
		}
	}
	return roots
}

// GetLeaves returns nodes with no children (bottom-level resources)
func (g *Graph) GetLeaves() []*Node {
	hasChildren := make(map[string]bool)
	for _, edge := range g.Edges {
		hasChildren[edge.From] = true
	}

	var leaves []*Node
	for uid, node := range g.Nodes {
		if !hasChildren[uid] {
			leaves = append(leaves, node)
		}
	}
	return leaves
}

// GetEdgesFrom returns all edges originating from a node
func (g *Graph) GetEdgesFrom(uid string) []*Edge {
	var edges []*Edge
	for _, edge := range g.Edges {
		if edge.From == uid {
			edges = append(edges, edge)
		}
	}
	return edges
}

// GetEdgesTo returns all edges pointing to a node
func (g *Graph) GetEdgesTo(uid string) []*Edge {
	var edges []*Edge
	for _, edge := range g.Edges {
		if edge.To == uid {
			edges = append(edges, edge)
		}
	}
	return edges
}

// Size returns the number of nodes and edges
func (g *Graph) Size() (nodes int, edges int) {
	return len(g.Nodes), len(g.Edges)
}

// FindNodeByName searches for nodes matching the given name.
// Returns exact matches first, then substring matches.
func (g *Graph) FindNodeByName(name string) []*Node {
	var exact, partial []*Node
	for _, node := range g.Nodes {
		if node.Name == name {
			exact = append(exact, node)
		} else if strings.Contains(node.Name, name) {
			partial = append(partial, node)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	return partial
}

// GetEdgeRelation returns the relation type of the edge from->to, if any.
func (g *Graph) GetEdgeRelation(from, to string) Relation {
	for _, edge := range g.Edges {
		if edge.From == from && edge.To == to {
			return edge.Relation
		}
	}
	return ""
}
