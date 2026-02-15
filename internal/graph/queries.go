package graph

// Query provides traversal and query operations on the graph
type Query struct {
	graph *Graph
}

// NewQuery creates a new query helper
func NewQuery(g *Graph) *Query {
	return &Query{graph: g}
}

// FindPath finds a path between two nodes (BFS)
func (q *Query) FindPath(fromUID, toUID string) []*Node {
	if fromUID == toUID {
		if node := q.graph.GetNode(fromUID); node != nil {
			return []*Node{node}
		}
		return nil
	}

	visited := make(map[string]bool)
	parent := make(map[string]string)
	queue := []string{fromUID}
	visited[fromUID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Check all connected nodes (both children and parents for undirected search)
		for _, edge := range q.graph.Edges {
			var neighbor string
			if edge.From == current {
				neighbor = edge.To
			} else if edge.To == current {
				neighbor = edge.From
			} else {
				continue
			}

			if !visited[neighbor] {
				visited[neighbor] = true
				parent[neighbor] = current
				queue = append(queue, neighbor)

				if neighbor == toUID {
					// Reconstruct path
					return q.reconstructPath(parent, fromUID, toUID)
				}
			}
		}
	}

	return nil // No path found
}

func (q *Query) reconstructPath(parent map[string]string, fromUID, toUID string) []*Node {
	var path []*Node
	current := toUID

	for current != "" {
		if node := q.graph.GetNode(current); node != nil {
			path = append([]*Node{node}, path...)
		}
		if current == fromUID {
			break
		}
		current = parent[current]
	}

	return path
}

// GetSubgraph returns a subgraph containing the node and all its descendants
func (q *Query) GetSubgraph(rootUID string) *Graph {
	subgraph := New()

	visited := make(map[string]bool)
	queue := []string{rootUID}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		if node := q.graph.GetNode(current); node != nil {
			subgraph.AddNode(node)
		}

		// Add all children
		for _, edge := range q.graph.Edges {
			if edge.From == current {
				queue = append(queue, edge.To)
				if visited[edge.To] {
					// Add edge even if already visited (for complete graph)
				}
			}
		}
	}

	// Add edges between nodes in subgraph
	for _, edge := range q.graph.Edges {
		if _, inFrom := subgraph.Nodes[edge.From]; inFrom {
			if _, inTo := subgraph.Nodes[edge.To]; inTo {
				subgraph.AddEdge(edge.From, edge.To, edge.Relation)
			}
		}
	}

	return subgraph
}

// GetAncestors returns all ancestor nodes (transitive parents)
func (q *Query) GetAncestors(uid string) []*Node {
	ancestors := make(map[string]*Node)
	queue := []string{uid}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range q.graph.Edges {
			if edge.To == current {
				parentUID := edge.From
				if _, seen := ancestors[parentUID]; !seen && parentUID != uid {
					if parent := q.graph.GetNode(parentUID); parent != nil {
						ancestors[parentUID] = parent
						queue = append(queue, parentUID)
					}
				}
			}
		}
	}

	result := make([]*Node, 0, len(ancestors))
	for _, node := range ancestors {
		result = append(result, node)
	}
	return result
}

// GetDescendants returns all descendant nodes (transitive children)
func (q *Query) GetDescendants(uid string) []*Node {
	descendants := make(map[string]*Node)
	queue := []string{uid}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range q.graph.Edges {
			if edge.From == current {
				childUID := edge.To
				if _, seen := descendants[childUID]; !seen && childUID != uid {
					if child := q.graph.GetNode(childUID); child != nil {
						descendants[childUID] = child
						queue = append(queue, childUID)
					}
				}
			}
		}
	}

	result := make([]*Node, 0, len(descendants))
	for _, node := range descendants {
		result = append(result, node)
	}
	return result
}

// FindByKind returns all nodes of a specific kind
func (q *Query) FindByKind(kind string) []*Node {
	var result []*Node
	for _, node := range q.graph.Nodes {
		if node.Kind == kind {
			result = append(result, node)
		}
	}
	return result
}

// FindByNamespace returns all nodes in a namespace
func (q *Query) FindByNamespace(namespace string) []*Node {
	var result []*Node
	for _, node := range q.graph.Nodes {
		if node.Namespace == namespace {
			result = append(result, node)
		}
	}
	return result
}

// FindByStatus returns all nodes with a specific status
func (q *Query) FindByStatus(status NodeStatus) []*Node {
	var result []*Node
	for _, node := range q.graph.Nodes {
		if node.Status == status {
			result = append(result, node)
		}
	}
	return result
}

// FindUnhealthy returns all nodes that are not healthy
func (q *Query) FindUnhealthy() []*Node {
	var result []*Node
	for _, node := range q.graph.Nodes {
		if node.Status == StatusError || node.Status == StatusWarning {
			result = append(result, node)
		}
	}
	return result
}

// GetOwnerChain returns the chain of owners for a resource
func (q *Query) GetOwnerChain(uid string) []*Node {
	var chain []*Node
	current := uid

	for {
		parents := q.graph.GetParents(current)
		if len(parents) == 0 {
			break
		}

		// Follow the first owner (typically there's only one)
		owner := parents[0]
		chain = append([]*Node{owner}, chain...)
		current = owner.UID
	}

	// Add the original node at the end
	if node := q.graph.GetNode(uid); node != nil {
		chain = append(chain, node)
	}

	return chain
}
