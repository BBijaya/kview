package views

import (
	"sort"
	"strings"

	"github.com/bijaya/kview/internal/graph"
)

// sectionDef defines a relationship section in Mode 2
type sectionDef struct {
	direction string                                         // "up", "down", "lateral"
	label     string                                         // "Owned By", "Contains", etc.
	finder    func(g *graph.Graph, uid string) []*graph.Node // finds related nodes
	ownerHint bool                                           // show ↳ owner under each pod item
}

// resourceSections returns the ordered section list for each resource kind
func resourceSections(kind string) []sectionDef {
	switch kind {
	case "Pod":
		return []sectionDef{
			{direction: "up", label: "Owned By", finder: findOwnershipRootNode},
			{direction: "down", label: "Contains", finder: findOwnedChildren},
			{direction: "down", label: "Mounts", finder: findOutboundByRelation(graph.RelationMounts)},
			{direction: "down", label: "Env References", finder: findPodEnvRefs},
			{direction: "lateral", label: "Selected By", finder: findInboundByRelation(graph.RelationSelects)},
			{direction: "lateral", label: "Scheduled On", finder: findScheduledNode},
		}
	case "Deployment":
		return []sectionDef{
			{direction: "down", label: "Owns", finder: findOwnedPodsSkipRS},
			{direction: "down", label: "Mounts", finder: findAggregatedMounts},
			{direction: "down", label: "Env References", finder: findAggregatedEnvRefs},
			{direction: "down", label: "Uses ServiceAccount", finder: findAggregatedSAs},
			{direction: "lateral", label: "Selected By", finder: findInboundByRelation(graph.RelationSelects)},
			{direction: "lateral", label: "Targeted By", finder: findInboundByRelation(graph.RelationTargets)},
			{direction: "lateral", label: "Scheduled On", finder: findAggregatedScheduledNodes},
		}
	case "StatefulSet":
		return []sectionDef{
			{direction: "down", label: "Owns", finder: findOwnedChildren},
			{direction: "down", label: "Mounts", finder: findAggregatedMounts},
			{direction: "down", label: "Env References", finder: findAggregatedEnvRefs},
			{direction: "down", label: "Uses ServiceAccount", finder: findAggregatedSAs},
			{direction: "lateral", label: "Selected By", finder: findInboundByRelation(graph.RelationSelects)},
			{direction: "lateral", label: "Targeted By", finder: findInboundByRelation(graph.RelationTargets)},
			{direction: "lateral", label: "Scheduled On", finder: findAggregatedScheduledNodes},
		}
	case "DaemonSet":
		return []sectionDef{
			{direction: "down", label: "Owns", finder: findOwnedChildren},
			{direction: "down", label: "Mounts", finder: findAggregatedMounts},
			{direction: "down", label: "Env References", finder: findAggregatedEnvRefs},
			{direction: "down", label: "Uses ServiceAccount", finder: findAggregatedSAs},
			{direction: "lateral", label: "Selected By", finder: findInboundByRelation(graph.RelationSelects)},
			{direction: "lateral", label: "Targeted By", finder: findInboundByRelation(graph.RelationTargets)},
			{direction: "lateral", label: "Scheduled On", finder: findAggregatedScheduledNodes},
		}
	case "Job":
		return []sectionDef{
			{direction: "up", label: "Owned By", finder: findOwnershipRootNode},
			{direction: "down", label: "Owns", finder: findOwnedChildren},
			{direction: "down", label: "Mounts", finder: findAggregatedMounts},
			{direction: "down", label: "Env References", finder: findAggregatedEnvRefs},
			{direction: "down", label: "Uses ServiceAccount", finder: findAggregatedSAs},
			{direction: "lateral", label: "Scheduled On", finder: findAggregatedScheduledNodes},
		}
	case "CronJob":
		return []sectionDef{
			{direction: "down", label: "Owns", finder: findOwnedChildren},
			{direction: "down", label: "Mounts", finder: findAggregatedMounts},
			{direction: "down", label: "Env References", finder: findAggregatedEnvRefs},
			{direction: "down", label: "Uses ServiceAccount", finder: findAggregatedSAs},
			{direction: "lateral", label: "Scheduled On", finder: findAggregatedScheduledNodes},
		}
	case "ReplicaSet":
		return []sectionDef{
			{direction: "up", label: "Owned By", finder: findOwnershipRootNode},
			{direction: "down", label: "Owns", finder: findOwnedChildren},
			{direction: "down", label: "Mounts", finder: findAggregatedMounts},
			{direction: "down", label: "Env References", finder: findAggregatedEnvRefs},
			{direction: "down", label: "Uses ServiceAccount", finder: findAggregatedSAs},
			{direction: "lateral", label: "Scheduled On", finder: findAggregatedScheduledNodes},
		}
	case "Secret":
		return []sectionDef{
			{direction: "up", label: "Mounted By", finder: findInboundByRelation(graph.RelationMounts), ownerHint: true},
			{direction: "up", label: "Env Referenced By", finder: findInboundByRelation(graph.RelationReferences)},
			{direction: "up", label: "Owner Workloads", finder: findOwnerWorkloads},
		}
	case "ConfigMap":
		return []sectionDef{
			{direction: "up", label: "Mounted By", finder: findInboundByRelation(graph.RelationMounts), ownerHint: true},
			{direction: "up", label: "Env Referenced By", finder: findInboundByRelation(graph.RelationReferences)},
			{direction: "up", label: "Owner Workloads", finder: findOwnerWorkloads},
		}
	case "Service":
		return []sectionDef{
			{direction: "lateral", label: "Selects", finder: findOutboundByRelation(graph.RelationSelects), ownerHint: true},
			{direction: "up", label: "Routed By", finder: findInboundByRelation(graph.RelationRoutes)},
			{direction: "lateral", label: "Endpoint Nodes", finder: findEndpointNodes},
		}
	case "Ingress":
		return []sectionDef{
			{direction: "lateral", label: "Routes To", finder: findOutboundByRelation(graph.RelationRoutes)},
			{direction: "down", label: "Backend Pods", finder: findBackendPods, ownerHint: true},
		}
	case "Node":
		return []sectionDef{
			{direction: "lateral", label: "Runs", finder: findPodsOnNode, ownerHint: true},
			{direction: "lateral", label: "Workloads", finder: findNodeWorkloads},
		}
	case "PersistentVolumeClaim":
		return []sectionDef{
			{direction: "lateral", label: "Binds To", finder: findOutboundByRelation(graph.RelationBinds)},
			{direction: "up", label: "Mounted By", finder: findInboundByRelation(graph.RelationMounts), ownerHint: true},
			{direction: "up", label: "Owner Workloads", finder: findOwnerWorkloads},
		}
	case "HorizontalPodAutoscaler":
		return []sectionDef{
			{direction: "lateral", label: "Targets", finder: findOutboundByRelation(graph.RelationTargets)},
			{direction: "down", label: "Target's Pods", finder: findTargetPods},
		}
	default:
		return []sectionDef{
			{direction: "up", label: "Owned By", finder: findOwnershipRootNode},
			{direction: "down", label: "Owns", finder: findOwnedChildren},
			{direction: "lateral", label: "Related", finder: findNonOwnershipRelated},
		}
	}
}

// --- Finder functions ---
// All finders match the signature: func(g *graph.Graph, uid string) []*graph.Node

// findInboundByRelation returns a finder for inbound edges of a specific relation type
func findInboundByRelation(rel graph.Relation) func(g *graph.Graph, uid string) []*graph.Node {
	return func(g *graph.Graph, uid string) []*graph.Node {
		var nodes []*graph.Node
		seen := make(map[string]bool)
		for _, edge := range g.GetEdgesTo(uid) {
			if edge.Relation == rel && !seen[edge.From] {
				if n := g.GetNode(edge.From); n != nil {
					nodes = append(nodes, n)
					seen[edge.From] = true
				}
			}
		}
		return nodes
	}
}

// findOutboundByRelation returns a finder for outbound edges of a specific relation type
func findOutboundByRelation(rel graph.Relation) func(g *graph.Graph, uid string) []*graph.Node {
	return func(g *graph.Graph, uid string) []*graph.Node {
		var nodes []*graph.Node
		seen := make(map[string]bool)
		for _, edge := range g.GetEdgesFrom(uid) {
			if edge.Relation == rel && !seen[edge.To] {
				if n := g.GetNode(edge.To); n != nil {
					nodes = append(nodes, n)
					seen[edge.To] = true
				}
			}
		}
		return nodes
	}
}

// findOwnershipRootNode finds the top-level workload owner, skipping intermediate ReplicaSets
func findOwnershipRootNode(g *graph.Graph, uid string) []*graph.Node {
	root := getOwnershipRoot(g, uid)
	if root != nil {
		return []*graph.Node{root}
	}
	return nil
}

// findOwnedChildren finds direct owned children (outbound RelationOwns edges)
func findOwnedChildren(g *graph.Graph, uid string) []*graph.Node {
	var nodes []*graph.Node
	for _, edge := range g.GetEdgesFrom(uid) {
		if edge.Relation == graph.RelationOwns {
			if n := g.GetNode(edge.To); n != nil {
				nodes = append(nodes, n)
			}
		}
	}
	return nodes
}

// findOwnedPodsSkipRS finds owned pods, skipping through intermediate ReplicaSets
func findOwnedPodsSkipRS(g *graph.Graph, uid string) []*graph.Node {
	var pods []*graph.Node
	for _, edge := range g.GetEdgesFrom(uid) {
		if edge.Relation != graph.RelationOwns {
			continue
		}
		child := g.GetNode(edge.To)
		if child == nil {
			continue
		}
		if child.Kind == "ReplicaSet" {
			for _, rsEdge := range g.GetEdgesFrom(child.UID) {
				if rsEdge.Relation == graph.RelationOwns {
					if pod := g.GetNode(rsEdge.To); pod != nil {
						pods = append(pods, pod)
					}
				}
			}
		} else {
			pods = append(pods, child)
		}
	}
	return pods
}

// findScheduledNode finds the Node a Pod is scheduled on (outbound RelationUses to Node kind)
func findScheduledNode(g *graph.Graph, uid string) []*graph.Node {
	var nodes []*graph.Node
	for _, edge := range g.GetEdgesFrom(uid) {
		if edge.Relation == graph.RelationUses {
			if n := g.GetNode(edge.To); n != nil && n.Kind == "Node" {
				nodes = append(nodes, n)
			}
		}
	}
	return nodes
}

// findPodsOnNode finds Pods running on a Node (inbound RelationUses from Pod kind)
func findPodsOnNode(g *graph.Graph, uid string) []*graph.Node {
	var nodes []*graph.Node
	for _, edge := range g.GetEdgesTo(uid) {
		if edge.Relation == graph.RelationUses {
			if n := g.GetNode(edge.From); n != nil && n.Kind == "Pod" {
				nodes = append(nodes, n)
			}
		}
	}
	return nodes
}

// findNonOwnershipRelated finds all non-ownership connections (default fallback)
func findNonOwnershipRelated(g *graph.Graph, uid string) []*graph.Node {
	var nodes []*graph.Node
	seen := make(map[string]bool)

	for _, edge := range g.GetEdgesTo(uid) {
		if edge.Relation == graph.RelationOwns || seen[edge.From] {
			continue
		}
		if n := g.GetNode(edge.From); n != nil {
			nodes = append(nodes, n)
			seen[edge.From] = true
		}
	}
	for _, edge := range g.GetEdgesFrom(uid) {
		if edge.Relation == graph.RelationOwns || seen[edge.To] {
			continue
		}
		if n := g.GetNode(edge.To); n != nil {
			nodes = append(nodes, n)
			seen[edge.To] = true
		}
	}
	return nodes
}

// findAggregatedMounts finds all Secrets/ConfigMaps/PVCs mounted by owned pods.
func findAggregatedMounts(g *graph.Graph, uid string) []*graph.Node {
	pods := getOwnedPods(g, uid)
	return collectFromPods(g, pods, graph.RelationMounts, "")
}

// findAggregatedEnvRefs finds all Secrets/ConfigMaps referenced via env vars by containers of owned pods.
func findAggregatedEnvRefs(g *graph.Graph, uid string) []*graph.Node {
	pods := getOwnedPods(g, uid)
	var result []*graph.Node
	seenUID := make(map[string]bool)
	for _, pod := range pods {
		// Get containers owned by this pod
		for _, edge := range g.GetEdgesFrom(pod.UID) {
			if edge.Relation != graph.RelationOwns {
				continue
			}
			container := g.GetNode(edge.To)
			if container == nil || container.Kind != "Container" {
				continue
			}
			// Get env references from this container
			for _, refEdge := range g.GetEdgesFrom(container.UID) {
				if refEdge.Relation != graph.RelationReferences {
					continue
				}
				target := g.GetNode(refEdge.To)
				if target == nil || seenUID[target.UID] {
					continue
				}
				seenUID[target.UID] = true
				result = append(result, target)
			}
		}
	}
	return result
}

// findAggregatedSAs finds all ServiceAccounts used by owned pods.
func findAggregatedSAs(g *graph.Graph, uid string) []*graph.Node {
	pods := getOwnedPods(g, uid)
	return collectFromPods(g, pods, graph.RelationUses, "ServiceAccount")
}

// findAggregatedScheduledNodes finds all Nodes that owned pods are scheduled on.
func findAggregatedScheduledNodes(g *graph.Graph, uid string) []*graph.Node {
	pods := getOwnedPods(g, uid)
	return collectFromPods(g, pods, graph.RelationUses, "Node")
}

// findPodEnvRefs finds all Secrets/ConfigMaps referenced via env vars by a pod's containers.
func findPodEnvRefs(g *graph.Graph, uid string) []*graph.Node {
	var result []*graph.Node
	seenUID := make(map[string]bool)
	for _, edge := range g.GetEdgesFrom(uid) {
		if edge.Relation != graph.RelationOwns {
			continue
		}
		container := g.GetNode(edge.To)
		if container == nil || container.Kind != "Container" {
			continue
		}
		for _, refEdge := range g.GetEdgesFrom(container.UID) {
			if refEdge.Relation != graph.RelationReferences {
				continue
			}
			target := g.GetNode(refEdge.To)
			if target == nil || seenUID[target.UID] {
				continue
			}
			seenUID[target.UID] = true
			result = append(result, target)
		}
	}
	return result
}

// findEndpointNodes finds Nodes that a Service's selected pods run on.
func findEndpointNodes(g *graph.Graph, uid string) []*graph.Node {
	// Get pods selected by this service
	var pods []*graph.Node
	for _, edge := range g.GetEdgesFrom(uid) {
		if edge.Relation == graph.RelationSelects {
			if pod := g.GetNode(edge.To); pod != nil && pod.Kind == "Pod" {
				pods = append(pods, pod)
			}
		}
	}
	return collectFromPods(g, pods, graph.RelationUses, "Node")
}

// findBackendPods finds Pods behind an Ingress (via routes -> services -> selects).
func findBackendPods(g *graph.Graph, uid string) []*graph.Node {
	var result []*graph.Node
	seenUID := make(map[string]bool)
	// Ingress -> Services (RelationRoutes)
	for _, edge := range g.GetEdgesFrom(uid) {
		if edge.Relation != graph.RelationRoutes {
			continue
		}
		svc := g.GetNode(edge.To)
		if svc == nil {
			continue
		}
		// Service -> Pods (RelationSelects)
		for _, svcEdge := range g.GetEdgesFrom(svc.UID) {
			if svcEdge.Relation != graph.RelationSelects {
				continue
			}
			pod := g.GetNode(svcEdge.To)
			if pod == nil || pod.Kind != "Pod" || seenUID[pod.UID] {
				continue
			}
			seenUID[pod.UID] = true
			result = append(result, pod)
		}
	}
	return result
}

// findOwnerWorkloads finds the workload controllers that own pods mounting/referencing this resource.
func findOwnerWorkloads(g *graph.Graph, uid string) []*graph.Node {
	var result []*graph.Node
	seenUID := make(map[string]bool)

	// Collect pods that mount this resource
	for _, edge := range g.GetEdgesTo(uid) {
		if edge.Relation == graph.RelationMounts {
			pod := g.GetNode(edge.From)
			if pod != nil && pod.Kind == "Pod" {
				owner := getOwnershipRoot(g, pod.UID)
				if owner != nil && !seenUID[owner.UID] {
					seenUID[owner.UID] = true
					result = append(result, owner)
				}
			}
		}
	}

	// Collect containers that reference this resource, then find their parent pods
	for _, edge := range g.GetEdgesTo(uid) {
		if edge.Relation == graph.RelationReferences {
			container := g.GetNode(edge.From)
			if container == nil || container.Kind != "Container" {
				continue
			}
			// Find parent pod of this container
			for _, parentEdge := range g.GetEdgesTo(container.UID) {
				if parentEdge.Relation == graph.RelationOwns {
					pod := g.GetNode(parentEdge.From)
					if pod != nil && pod.Kind == "Pod" {
						owner := getOwnershipRoot(g, pod.UID)
						if owner != nil && !seenUID[owner.UID] {
							seenUID[owner.UID] = true
							result = append(result, owner)
						}
					}
				}
			}
		}
	}

	return result
}

// findNodeWorkloads finds the workload controllers for pods running on a Node.
func findNodeWorkloads(g *graph.Graph, uid string) []*graph.Node {
	pods := findPodsOnNode(g, uid)
	var result []*graph.Node
	seenUID := make(map[string]bool)
	for _, pod := range pods {
		owner := getOwnershipRoot(g, pod.UID)
		if owner != nil && !seenUID[owner.UID] {
			seenUID[owner.UID] = true
			result = append(result, owner)
		}
	}
	return result
}

// findTargetPods finds pods owned by the workload targeted by an HPA.
func findTargetPods(g *graph.Graph, uid string) []*graph.Node {
	for _, edge := range g.GetEdgesFrom(uid) {
		if edge.Relation == graph.RelationTargets {
			target := g.GetNode(edge.To)
			if target != nil {
				return getOwnedPods(g, target.UID)
			}
		}
	}
	return nil
}

// --- Graph traversal helpers ---

// getOwnershipRoot walks inbound RelationOwns edges upward to find the top-level workload owner.
// Skips intermediate ReplicaSets. Returns nil if the node has no owner.
func getOwnershipRoot(g *graph.Graph, uid string) *graph.Node {
	current := uid
	var lastOwner *graph.Node
	for {
		var owner *graph.Node
		for _, edge := range g.GetEdgesTo(current) {
			if edge.Relation == graph.RelationOwns {
				if n := g.GetNode(edge.From); n != nil {
					owner = n
					break
				}
			}
		}
		if owner == nil {
			break
		}
		lastOwner = owner
		current = owner.UID
	}
	return lastOwner
}

// getOwnedPods recursively follows RelationOwns edges down, collecting only Pod nodes.
// Handles all intermediate controllers (RS for Deployments, Job for CronJobs).
func getOwnedPods(g *graph.Graph, uid string) []*graph.Node {
	var pods []*graph.Node
	visited := make(map[string]bool)
	var walk func(id string)
	walk = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true
		for _, edge := range g.GetEdgesFrom(id) {
			if edge.Relation != graph.RelationOwns {
				continue
			}
			child := g.GetNode(edge.To)
			if child == nil {
				continue
			}
			if child.Kind == "Pod" {
				pods = append(pods, child)
			} else {
				walk(child.UID) // recurse through RS, Job, etc.
			}
		}
	}
	walk(uid)
	return pods
}

// collectFromPods follows outbound edges of a specific relation from each pod, with optional kind filter.
// Deduplicates by UID, except ServiceAccount nodes which deduplicate by Name (synthetic per-pod UIDs).
func collectFromPods(g *graph.Graph, pods []*graph.Node, rel graph.Relation, kindFilter string) []*graph.Node {
	var result []*graph.Node
	seenUID := make(map[string]bool)
	seenSAName := make(map[string]bool)
	for _, pod := range pods {
		for _, edge := range g.GetEdgesFrom(pod.UID) {
			if edge.Relation != rel {
				continue
			}
			target := g.GetNode(edge.To)
			if target == nil {
				continue
			}
			if kindFilter != "" && target.Kind != kindFilter {
				continue
			}
			if target.Kind == "ServiceAccount" {
				if seenSAName[target.Name] {
					continue
				}
				seenSAName[target.Name] = true
			} else {
				if seenUID[target.UID] {
					continue
				}
				seenUID[target.UID] = true
			}
			result = append(result, target)
		}
	}
	return result
}

// --- Utility functions ---

// directionArrow returns the Unicode arrow for a section direction
func directionArrow(dir string) string {
	switch dir {
	case "up":
		return "⬆"
	case "down":
		return "⬇"
	case "lateral":
		return "↔"
	default:
		return "↔"
	}
}

// nodeInfo returns the Extras["info"] string for a node, or ""
func nodeInfo(n *graph.Node) string {
	if n.Extras != nil {
		return n.Extras["info"]
	}
	return ""
}

// sortNodesByKindName sorts nodes by Kind then Name for consistent ordering
func sortNodesByKindName(nodes []*graph.Node) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Kind != nodes[j].Kind {
			return nodes[i].Kind < nodes[j].Kind
		}
		return nodes[i].Name < nodes[j].Name
	})
}

// xrayEmoji returns a k9s-style emoji icon for each resource kind
func xrayEmoji(kind string) string {
	switch kind {
	case "Namespace":
		return "\U0001F5C2" // 🗂
	case "Deployment":
		return "\U0001FA82" // 🪂
	case "ReplicaSet":
		return "\U0001F46F" // 👯
	case "Pod":
		return "\U0001F69B" // 🚛
	case "Container":
		return "\U0001F433" // 🐳
	case "Service":
		return "\U0001F481" // 💁
	case "ConfigMap":
		return "\U0001F5FA" // 🗺
	case "Secret":
		return "\U0001F512" // 🔒
	case "Ingress":
		return "\U0001F310" // 🌐
	case "StatefulSet":
		return "\U0001F38E" // 🎎
	case "DaemonSet":
		return "\U0001F608" // 😈
	case "Job":
		return "\U0001F3C3" // 🏃
	case "CronJob":
		return "\u23F0"     // ⏰
	case "PersistentVolumeClaim":
		return "\U0001F39F" // 🎟
	case "PersistentVolume":
		return "\U0001F4DA" // 📚
	case "HorizontalPodAutoscaler":
		return "\u264E"     // ♎
	case "Node":
		return "\U0001F5A5" // 🖥
	case "ServiceAccount":
		return "\U0001F4B3" // 💳
	default:
		return "\U0001F4CE" // 📎
	}
}

// xrayStatusLabel returns a status label for unhealthy resources (k9s-style TOAST)
func xrayStatusLabel(status graph.NodeStatus) string {
	switch status {
	case graph.StatusWarning:
		return "TOAST"
	case graph.StatusError:
		return "TOAST"
	case graph.StatusPending:
		return "PENDING"
	default:
		return ""
	}
}

// statusToRowStatus converts graph status to table row status string for color coding
func statusToRowStatus(status graph.NodeStatus) string {
	switch status {
	case graph.StatusHealthy:
		return "Running"
	case graph.StatusWarning:
		return "Warning"
	case graph.StatusError:
		return "Failed"
	case graph.StatusPending:
		return "Pending"
	default:
		return ""
	}
}

// xrayKindToResource converts a Kind to its plural API resource name
func xrayKindToResource(kind string) string {
	switch kind {
	case "Pod":
		return "pods"
	case "Deployment":
		return "deployments"
	case "Service":
		return "services"
	case "ReplicaSet":
		return "replicasets"
	case "DaemonSet":
		return "daemonsets"
	case "StatefulSet":
		return "statefulsets"
	case "Job":
		return "jobs"
	case "CronJob":
		return "cronjobs"
	case "ConfigMap":
		return "configmaps"
	case "Secret":
		return "secrets"
	case "Ingress":
		return "ingresses"
	case "PersistentVolumeClaim":
		return "persistentvolumeclaims"
	case "PersistentVolume":
		return "persistentvolumes"
	case "HorizontalPodAutoscaler":
		return "horizontalpodautoscalers"
	case "Node":
		return "nodes"
	default:
		return strings.ToLower(kind) + "s"
	}
}

// kindLabel returns a short display label for each resource kind
func kindLabel(kind string) string {
	switch kind {
	case "PersistentVolumeClaim":
		return "PVC"
	case "PersistentVolume":
		return "PV"
	case "HorizontalPodAutoscaler":
		return "HPA"
	default:
		return kind
	}
}

// matchKind checks if a node's kind matches the filter, handling case-insensitive
// comparison and plural forms (e.g., "certificates" matches "Certificate").
func matchKind(nodeKind, filter string) bool {
	if nodeKind == filter {
		return true
	}
	nk := strings.ToLower(nodeKind)
	fk := strings.ToLower(filter)
	if nk == fk {
		return true
	}
	// Handle plural: "certificates" → "certificate" matches "certificate"
	if strings.HasSuffix(fk, "s") && nk == fk[:len(fk)-1] {
		return true
	}
	// Handle reverse: filter "certificate" matches node "Certificate" (already covered above)
	// Handle plural node kind (unlikely but defensive)
	if strings.HasSuffix(nk, "s") && fk == nk[:len(nk)-1] {
		return true
	}
	return false
}
