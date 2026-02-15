package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/graph"
	"github.com/bijaya/kview/internal/ui/components"
)

// flattenTree converts the graph to a flat list of xrayNodes
func (v *XrayView) flattenTree() []*xrayNode {
	if v.graph == nil {
		return nil
	}

	switch v.mode {
	case xrayModeType:
		return v.flattenTypeMode()
	case xrayModeResource:
		return v.flattenResourceMode()
	}
	return nil
}

// flattenTypeMode flattens tree for Mode 1: k9s-style with namespace grouping
func (v *XrayView) flattenTypeMode() []*xrayNode {
	var nodes []*xrayNode

	q := graph.NewQuery(v.graph)
	kindNodes := q.FindByKind(v.rootKind)

	// Group by namespace
	nsByName := make(map[string][]*graph.Node)
	for _, n := range kindNodes {
		ns := n.Namespace
		if ns == "" {
			ns = "(cluster)"
		}
		nsByName[ns] = append(nsByName[ns], n)
	}

	var nsNames []string
	for ns := range nsByName {
		nsNames = append(nsNames, ns)
	}
	sort.Strings(nsNames)

	for _, ns := range nsNames {
		nsResources := nsByName[ns]
		sort.Slice(nsResources, func(i, j int) bool {
			return nsResources[i].Name < nsResources[j].Name
		})

		nsUID := "ns/" + ns
		nsExpanded := v.expanded[nsUID]
		nsNode := &xrayNode{
			uid:         nsUID,
			kind:        "Namespace",
			name:        ns,
			ns:          ns,
			depth:       0,
			hasChildren: true,
			isExpanded:  nsExpanded,
			isNsHeader:  true,
			childCount:  len(nsResources),
		}
		nodes = append(nodes, nsNode)

		if !nsExpanded {
			continue
		}

		visited := make(map[string]bool)
		for _, root := range nsResources {
			v.flattenNodeRecursive(root, 1, &nodes, visited)
		}
	}

	setTreePrefixes(nodes)
	return nodes
}

// flattenResourceMode flattens tree for Mode 2: focused resource with grouped relationship sections
func (v *XrayView) flattenResourceMode() []*xrayNode {
	if v.focusUID == "" {
		return nil
	}

	focusNode := v.graph.GetNode(v.focusUID)
	if focusNode == nil {
		return nil
	}

	var nodes []*xrayNode

	// 1. Emit focused node at depth 0
	nodes = append(nodes, &xrayNode{
		uid:       v.focusUID,
		kind:      focusNode.Kind,
		name:      focusNode.Name,
		ns:        focusNode.Namespace,
		status:    focusNode.Status,
		depth:     0,
		isFocused: true,
		info:      nodeInfo(focusNode),
	})

	// 2. Get sections for this resource kind
	sections := resourceSections(focusNode.Kind)
	anySection := false

	// 3. For each section, find related nodes and emit
	for _, sec := range sections {
		related := sec.finder(v.graph, v.focusUID)
		if len(related) == 0 {
			continue
		}
		anySection = true

		// Sort related nodes by name
		sort.Slice(related, func(i, j int) bool {
			return related[i].Name < related[j].Name
		})

		sectionUID := fmt.Sprintf("section/%s/%s", v.focusUID, sec.label)
		isExpanded := v.expanded[sectionUID]

		// Emit section header
		nodes = append(nodes, &xrayNode{
			uid:             sectionUID,
			name:            sec.label,
			depth:           1,
			hasChildren:     true,
			isExpanded:      isExpanded,
			isSectionHeader: true,
			childCount:      len(related),
			relation:        sec.direction,
		})

		if !isExpanded {
			continue
		}

		// Emit related resource nodes
		for _, rel := range related {
			info := nodeInfo(rel)
			name := rel.Name

			// Container nodes in "Env Referenced By" need pod context
			if rel.Kind == "Container" && sec.label == "Env Referenced By" {
				if podName, _ := v.getContainerParentPod(rel.UID); podName != "" {
					name = rel.Name + " (" + podName + ")"
				}
			}

			nodes = append(nodes, &xrayNode{
				uid:    rel.UID,
				kind:   rel.Kind,
				name:   name,
				ns:     rel.Namespace,
				status: rel.Status,
				depth:  2,
				info:   info,
			})

			// Owner hint for pods
			if sec.ownerHint && rel.Kind == "Pod" {
				owner := getOwnershipRoot(v.graph, rel.UID)
				if owner != nil && owner.UID != rel.UID {
					nodes = append(nodes, &xrayNode{
						uid:         "hint/" + rel.UID + "/" + owner.UID,
						kind:        owner.Kind,
						name:        owner.Name,
						ns:          owner.Namespace,
						status:      owner.Status,
						depth:       3,
						isOwnerHint: true,
						info:        nodeInfo(owner),
					})
				}
			}
		}
	}

	// 4. If no sections had content, show placeholder
	if !anySection {
		nodes = append(nodes, &xrayNode{
			uid:             "placeholder/" + v.focusUID,
			name:            "(no references found)",
			depth:           1,
			isSectionHeader: true,
		})
	}

	setTreePrefixes(nodes)
	return nodes
}

// flattenNodeRecursive flattens a node and its children into the list.
// Skips ReplicaSet nodes (shows their pods directly under the deployment).
// Uses pod-specific rendering for Pod nodes.
func (v *XrayView) flattenNodeRecursive(node *graph.Node, depth int, nodes *[]*xrayNode, visited map[string]bool) {
	if visited[node.UID] {
		return
	}
	visited[node.UID] = true

	// Pod nodes use special rendering (SA, containers, volumes, init containers)
	if node.Kind == "Pod" {
		v.flattenPodNode(node, depth, nodes, visited)
		return
	}

	// Get all children (owned + non-owned refs)
	allChildren := v.graph.GetXrayChildren(node.UID)

	// Skip ReplicaSet owned children: flatten through them to show pods directly
	var visibleChildren []*graph.Node
	for _, child := range allChildren {
		if child.Kind == "ReplicaSet" && v.graph.GetEdgeRelation(node.UID, child.UID) == graph.RelationOwns {
			visited[child.UID] = true // mark RS as visited
			rsChildren := v.graph.GetOwnedChildren(child.UID)
			visibleChildren = append(visibleChildren, rsChildren...)
		} else {
			visibleChildren = append(visibleChildren, child)
		}
	}

	sort.Slice(visibleChildren, func(i, j int) bool {
		return visibleChildren[i].Name < visibleChildren[j].Name
	})

	isExpanded := v.expanded[node.UID]

	xn := &xrayNode{
		uid:         node.UID,
		kind:        node.Kind,
		name:        node.Name,
		ns:          node.Namespace,
		status:      node.Status,
		depth:       depth,
		hasChildren: len(visibleChildren) > 0,
		isExpanded:  isExpanded,
		childCount:  len(visibleChildren),
		info:        nodeInfo(node),
	}
	*nodes = append(*nodes, xn)

	if xn.hasChildren && isExpanded {
		for _, child := range visibleChildren {
			if !visited[child.UID] {
				v.flattenNodeRecursive(child, depth+1, nodes, visited)
			}
		}
	}
}

// flattenPodNode renders a Pod with k9s-style child ordering:
// ServiceAccount -> regular containers (with env refs) -> volume refs -> init containers
func (v *XrayView) flattenPodNode(pod *graph.Node, depth int, nodes *[]*xrayNode, visited map[string]bool) {
	var saNodes, containerNodes, volumeNodes, initContainerNodes []*graph.Node

	for _, edge := range v.graph.Edges {
		if edge.From != pod.UID {
			continue
		}
		child := v.graph.Nodes[edge.To]
		if child == nil {
			continue
		}
		switch {
		case child.Kind == "ServiceAccount":
			saNodes = append(saNodes, child)
		case child.Kind == "Container" && strings.Contains(child.UID, "/co/"):
			containerNodes = append(containerNodes, child)
		case child.Kind == "Container" && strings.Contains(child.UID, "/ic/"):
			initContainerNodes = append(initContainerNodes, child)
		default:
			// Secret, ConfigMap, PVC volume refs
			if edge.Relation == graph.RelationMounts {
				volumeNodes = append(volumeNodes, child)
			}
		}
	}

	totalChildren := len(saNodes) + len(containerNodes) + len(volumeNodes) + len(initContainerNodes)
	isExpanded := v.expanded[pod.UID]

	xn := &xrayNode{
		uid:         pod.UID,
		kind:        "Pod",
		name:        pod.Name,
		ns:          pod.Namespace,
		status:      pod.Status,
		depth:       depth,
		hasChildren: totalChildren > 0,
		isExpanded:  isExpanded,
		childCount:  totalChildren,
		info:        nodeInfo(pod),
	}
	*nodes = append(*nodes, xn)

	if totalChildren == 0 || !isExpanded {
		return
	}

	childDepth := depth + 1

	// 1. ServiceAccount
	for _, sa := range saNodes {
		*nodes = append(*nodes, &xrayNode{
			uid: sa.UID, kind: sa.Kind, name: sa.Name,
			ns: sa.Namespace, status: sa.Status, depth: childDepth,
		})
	}

	// 2. Regular containers with env ref children
	sort.Slice(containerNodes, func(i, j int) bool {
		return containerNodes[i].Name < containerNodes[j].Name
	})
	for _, c := range containerNodes {
		v.flattenContainerNode(c, childDepth, nodes)
	}

	// 3. Volume refs (secrets, configmaps, PVCs)
	sort.Slice(volumeNodes, func(i, j int) bool {
		return volumeNodes[i].Name < volumeNodes[j].Name
	})
	for _, vol := range volumeNodes {
		*nodes = append(*nodes, &xrayNode{
			uid: vol.UID, kind: vol.Kind, name: vol.Name,
			ns: vol.Namespace, status: vol.Status, depth: childDepth,
		})
	}

	// 4. Init containers
	sort.Slice(initContainerNodes, func(i, j int) bool {
		return initContainerNodes[i].Name < initContainerNodes[j].Name
	})
	for _, ic := range initContainerNodes {
		v.flattenContainerNode(ic, childDepth, nodes)
	}
}

// flattenContainerNode renders a Container node with env ref children
// (Secrets/ConfigMaps referenced via envFrom or env[].valueFrom)
func (v *XrayView) flattenContainerNode(container *graph.Node, depth int, nodes *[]*xrayNode) {
	// Get env ref children (RelationReferences edges)
	var envRefChildren []*graph.Node
	for _, edge := range v.graph.Edges {
		if edge.From != container.UID || edge.Relation != graph.RelationReferences {
			continue
		}
		child := v.graph.Nodes[edge.To]
		if child != nil {
			envRefChildren = append(envRefChildren, child)
		}
	}

	sort.Slice(envRefChildren, func(i, j int) bool {
		return envRefChildren[i].Name < envRefChildren[j].Name
	})

	isExpanded := v.expanded[container.UID]

	xn := &xrayNode{
		uid:         container.UID,
		kind:        container.Kind,
		name:        container.Name,
		ns:          container.Namespace,
		status:      container.Status,
		depth:       depth,
		hasChildren: len(envRefChildren) > 0,
		isExpanded:  isExpanded,
		childCount:  len(envRefChildren),
	}
	*nodes = append(*nodes, xn)

	if len(envRefChildren) > 0 && isExpanded {
		for _, ref := range envRefChildren {
			// Use a synthetic UID so the same Secret/ConfigMap can appear
			// both as an env ref under a container AND as a volume ref under the pod
			*nodes = append(*nodes, &xrayNode{
				uid:    container.UID + "/ref/" + ref.Name,
				kind:   ref.Kind,
				name:   ref.Name,
				ns:     ref.Namespace,
				status: ref.Status,
				depth:  depth + 1,
			})
		}
	}
}

// flattenNodeWithRelation is like flattenNodeRecursive but carries a relation label (Mode 2)
func (v *XrayView) flattenNodeWithRelation(node *graph.Node, depth int, nodes *[]*xrayNode, visited map[string]bool, relation string) {
	if visited[node.UID] {
		return
	}
	visited[node.UID] = true

	xrayChildren := v.graph.GetXrayChildren(node.UID)
	hasChildren := len(xrayChildren) > 0
	isExpanded := v.expanded[node.UID]

	xn := &xrayNode{
		uid:         node.UID,
		kind:        node.Kind,
		name:        node.Name,
		ns:          node.Namespace,
		status:      node.Status,
		depth:       depth,
		hasChildren: hasChildren,
		isExpanded:  isExpanded,
		relation:    relation,
		childCount:  len(xrayChildren),
		info:        nodeInfo(node),
	}
	*nodes = append(*nodes, xn)

	if hasChildren && isExpanded {
		sortNodesByKindName(xrayChildren)
		for _, child := range xrayChildren {
			if visited[child.UID] {
				continue
			}
			rel := v.graph.GetEdgeRelation(node.UID, child.UID)
			if rel == graph.RelationOwns {
				v.flattenNodeRecursive(child, depth+1, nodes, visited)
			} else {
				*nodes = append(*nodes, &xrayNode{
					uid: child.UID, kind: child.Kind, name: child.Name,
					ns: child.Namespace, status: child.Status, depth: depth + 1,
					info: nodeInfo(child),
				})
			}
		}
	}
}

// nodesToRows converts xrayNodes to table rows (single column, k9s-style)
func (v *XrayView) nodesToRows(nodes []*xrayNode) []components.Row {
	rows := make([]components.Row, len(nodes))
	for i, node := range nodes {
		var line strings.Builder
		line.WriteString(node.prefix)

		if node.isNsHeader {
			// Namespace header: folder emoji + name
			line.WriteString(xrayEmoji("Namespace"))
			line.WriteString(" ")
			line.WriteString(node.name)
			if node.childCount > 0 {
				line.WriteString(fmt.Sprintf("(%d)", node.childCount))
			}
		} else if node.isSectionHeader {
			// Section header: direction arrow + label + (count)
			if node.relation != "" {
				line.WriteString(directionArrow(node.relation))
				line.WriteString(" ")
			}
			line.WriteString(node.name)
			if node.childCount > 0 {
				line.WriteString(fmt.Sprintf(" (%d)", node.childCount))
			}
		} else if node.isOwnerHint {
			// Owner hint: ↳ + emoji + name + [kind label] + [info]
			line.WriteString("↳ ")
			line.WriteString(xrayEmoji(node.kind))
			line.WriteString(" ")
			line.WriteString(node.name)

			// Kind label in Mode 2
			if v.mode == xrayModeResource {
				if kl := kindLabel(node.kind); kl != "" {
					currentWidth := lipgloss.Width(line.String())
					if currentWidth < 45 {
						line.WriteString(strings.Repeat(" ", 45-currentWidth))
					} else {
						line.WriteString("  ")
					}
					line.WriteString(kl)
				}
			}

			if node.info != "" {
				line.WriteString(" [")
				line.WriteString(node.info)
				line.WriteString("]")
			}
		} else {
			// Resource node: emoji + name + [kind label] + (childCount) + [info] + [STATUS]
			emoji := xrayEmoji(node.kind)
			line.WriteString(emoji)
			line.WriteString(" ")

			if node.isFocused {
				line.WriteString("► ")
			}

			line.WriteString(node.name)

			// Kind label in Mode 2
			if v.mode == xrayModeResource {
				if kl := kindLabel(node.kind); kl != "" {
					currentWidth := lipgloss.Width(line.String())
					if currentWidth < 45 {
						line.WriteString(strings.Repeat(" ", 45-currentWidth))
					} else {
						line.WriteString("  ")
					}
					line.WriteString(kl)
				}
			}

			// Child count
			if node.childCount > 0 {
				line.WriteString(fmt.Sprintf("(%d)", node.childCount))
			}

			// Info annotation (ready counts, replicas)
			if node.info != "" {
				line.WriteString(" [")
				line.WriteString(node.info)
				line.WriteString("]")
			}

			// Status label for unhealthy resources
			statusLabel := xrayStatusLabel(node.status)
			if statusLabel != "" {
				line.WriteString("  ")
				line.WriteString(statusLabel)
			}
		}

		rows[i] = components.Row{
			ID:     node.uid,
			Values: []string{line.String()},
			Status: statusToRowStatus(node.status),
		}
	}
	return rows
}

// setTreePrefixes calculates tree-drawing prefixes for all nodes
func setTreePrefixes(nodes []*xrayNode) {
	for i, node := range nodes {
		if node.depth == 0 {
			node.prefix = ""
			continue
		}

		isLast := true
		for j := i + 1; j < len(nodes); j++ {
			if nodes[j].depth < node.depth {
				break
			}
			if nodes[j].depth == node.depth {
				isLast = false
				break
			}
		}

		var prefix strings.Builder
		for d := 1; d < node.depth; d++ {
			hasContinuation := false
			for j := i + 1; j < len(nodes); j++ {
				if nodes[j].depth < d {
					break
				}
				if nodes[j].depth == d {
					hasContinuation = true
					break
				}
			}
			if hasContinuation {
				prefix.WriteString("│   ")
			} else {
				prefix.WriteString("    ")
			}
		}

		if isLast {
			prefix.WriteString("└── ")
		} else {
			prefix.WriteString("├── ")
		}

		node.prefix = prefix.String()
	}
}
