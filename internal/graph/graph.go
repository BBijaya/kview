package graph

import (
	"context"

	"github.com/bijaya/kview/internal/k8s"
)

// ResourceGraph manages the resource relationship graph
type ResourceGraph struct {
	client k8s.Client
	graph  *Graph
}

// NewResourceGraph creates a new resource graph
func NewResourceGraph(client k8s.Client) *ResourceGraph {
	return &ResourceGraph{
		client: client,
		graph:  New(),
	}
}

// Build builds the complete graph from the cluster
func (rg *ResourceGraph) Build(ctx context.Context, namespace string) error {
	builder := NewBuilder()

	// Fetch all resource types (errors are non-fatal — we build what we can)
	//
	// ORDERING: Parents must be added before children so that AddEdge
	// (which checks both nodes exist) can create ownership edges.
	//   Tier 0: top-level owners — Deployments, DaemonSets, StatefulSets, CronJobs
	//   Tier 1: mid-level — ReplicaSets (owned by Deployments), Jobs (owned by CronJobs)
	//   Tier 2: Pods (owned by RS, DS, STS, Jobs)
	//   Leaf:   everything else (no ownership edges to add)

	// Tier 0 — top-level workload owners
	deployments, _ := rg.client.ListDeployments(ctx, namespace)
	builder.AddDeployments(deployments)

	daemonsets, _ := rg.client.ListDaemonSets(ctx, namespace)
	builder.AddDaemonSets(daemonsets)

	statefulsets, _ := rg.client.ListStatefulSets(ctx, namespace)
	builder.AddStatefulSets(statefulsets)

	cronjobs, _ := rg.client.ListCronJobs(ctx, namespace)
	builder.AddCronJobs(cronjobs)

	// Tier 1 — owned by tier 0
	replicasets, _ := rg.client.ListReplicaSets(ctx, namespace)
	builder.AddReplicaSets(replicasets)

	jobs, _ := rg.client.ListJobs(ctx, namespace)
	builder.AddJobs(jobs)

	// Tier 2 — owned by tier 0 or 1
	pods, _ := rg.client.ListPods(ctx, namespace)
	builder.AddPods(pods)

	// Tier 3 — containers and service accounts (synthetic nodes owned by pods)
	builder.AddContainers(pods)
	builder.AddServiceAccounts(pods)

	// Leaf resources (no ownership edges)
	services, _ := rg.client.ListServices(ctx, namespace)
	builder.AddServices(services)

	ingresses, _ := rg.client.ListIngresses(ctx, namespace)
	builder.AddIngresses(ingresses)

	configmaps, _ := rg.client.ListConfigMaps(ctx, namespace)
	builder.AddConfigMaps(configmaps)

	secrets, _ := rg.client.ListSecrets(ctx, namespace)
	builder.AddSecrets(secrets)

	pvcs, _ := rg.client.ListPVCs(ctx, namespace)
	builder.AddPVCs(pvcs)

	pvs, _ := rg.client.ListPVs(ctx)
	builder.AddPVs(pvs)

	hpas, _ := rg.client.ListHPAs(ctx, namespace)
	builder.AddHPAs(hpas)

	nodes, _ := rg.client.ListNodes(ctx)
	builder.AddK8sNodes(nodes)

	// Link non-ownership relationships (all nodes exist now)
	builder.LinkServicesToPods(services, pods)
	builder.LinkIngressesToServices(ingresses)
	builder.LinkPVCsToPVs(pvcs)
	builder.LinkHPAsToTargets(hpas)
	builder.LinkStatefulSetsToServices(statefulsets)
	builder.LinkPodsToVolumes(pods)
	builder.LinkContainersToEnvRefs(pods)
	builder.LinkPodsToNodes(pods)

	// Calculate depths
	builder.CalculateDepths()

	rg.graph = builder.Build()
	return nil
}

// GetGraph returns the built graph
func (rg *ResourceGraph) GetGraph() *Graph {
	return rg.graph
}

// Query returns a query helper for the graph
func (rg *ResourceGraph) Query() *Query {
	return NewQuery(rg.graph)
}

// GetResourceTree returns a tree representation starting from a resource
func (rg *ResourceGraph) GetResourceTree(uid string) *TreeNode {
	node := rg.graph.GetNode(uid)
	if node == nil {
		return nil
	}

	return rg.buildTree(node, make(map[string]bool))
}

// TreeNode represents a node in a tree view of the graph
type TreeNode struct {
	Node     *Node
	Children []*TreeNode
}

func (rg *ResourceGraph) buildTree(node *Node, visited map[string]bool) *TreeNode {
	if visited[node.UID] {
		return nil // Prevent cycles
	}
	visited[node.UID] = true

	treeNode := &TreeNode{
		Node:     node,
		Children: make([]*TreeNode, 0),
	}

	for _, child := range rg.graph.GetChildren(node.UID) {
		if childTree := rg.buildTree(child, visited); childTree != nil {
			treeNode.Children = append(treeNode.Children, childTree)
		}
	}

	return treeNode
}

// RenderTree renders a tree to a string for display
func RenderTree(tree *TreeNode, prefix string, isLast bool) string {
	if tree == nil {
		return ""
	}

	var result string

	// Draw the current node
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	if prefix == "" {
		connector = ""
	}

	statusIcon := getStatusIcon(tree.Node.Status)
	result = prefix + connector + statusIcon + " " + tree.Node.Kind + "/" + tree.Node.Name + "\n"

	// Update prefix for children
	childPrefix := prefix
	if prefix != "" {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	// Render children
	for i, child := range tree.Children {
		isChildLast := i == len(tree.Children)-1
		result += RenderTree(child, childPrefix, isChildLast)
	}

	return result
}

func getStatusIcon(status NodeStatus) string {
	switch status {
	case StatusHealthy:
		return "●"
	case StatusWarning:
		return "◐"
	case StatusError:
		return "○"
	case StatusPending:
		return "◌"
	default:
		return "?"
	}
}
