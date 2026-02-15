package graph

import (
	"fmt"
	"strings"

	"github.com/bijaya/kview/internal/k8s"
)

// Builder builds a resource graph from Kubernetes resources
type Builder struct {
	graph *Graph
}

// NewBuilder creates a new graph builder
func NewBuilder() *Builder {
	return &Builder{
		graph: New(),
	}
}

// Build returns the constructed graph
func (b *Builder) Build() *Graph {
	return b.graph
}

// AddResources adds generic resources to the graph
func (b *Builder) AddResources(resources []k8s.Resource) {
	for i := range resources {
		b.addResource(&resources[i])
	}
}

// AddPods adds pods to the graph
func (b *Builder) AddPods(pods []k8s.PodInfo) {
	for i := range pods {
		pod := &pods[i]
		// Count ready containers
		readyCnt := 0
		totalCnt := len(pod.Containers)
		for _, c := range pod.Containers {
			if c.Ready {
				readyCnt++
			}
		}
		node := &Node{
			UID:       pod.UID,
			Kind:      "Pod",
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    b.getPodStatus(pod),
			Extras: map[string]string{
				"info": fmt.Sprintf("%d/%d", readyCnt, totalCnt),
			},
		}
		b.graph.AddNode(node)

		// Add owner relationships
		for _, owner := range pod.OwnerRefs {
			b.graph.AddEdge(owner.UID, pod.UID, RelationOwns)
		}
	}
}

// AddDeployments adds deployments to the graph
func (b *Builder) AddDeployments(deployments []k8s.DeploymentInfo) {
	for i := range deployments {
		dep := &deployments[i]
		node := &Node{
			UID:       dep.UID,
			Kind:      "Deployment",
			Name:      dep.Name,
			Namespace: dep.Namespace,
			Status:    b.getDeploymentStatus(dep),
			Extras: map[string]string{
				"info": fmt.Sprintf("%d/%d/%d", dep.AvailableReplicas, dep.Replicas, dep.Replicas-dep.AvailableReplicas),
			},
		}
		b.graph.AddNode(node)
	}
}

// AddServices adds services to the graph
func (b *Builder) AddServices(services []k8s.ServiceInfo) {
	for i := range services {
		svc := &services[i]
		node := &Node{
			UID:       svc.UID,
			Kind:      "Service",
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Status:    StatusHealthy,
		}
		b.graph.AddNode(node)
	}
}

// LinkServicesToPods creates edges from services to their selected pods
func (b *Builder) LinkServicesToPods(services []k8s.ServiceInfo, pods []k8s.PodInfo) {
	for _, svc := range services {
		if len(svc.Selector) == 0 {
			continue
		}

		for _, pod := range pods {
			if pod.Namespace != svc.Namespace {
				continue
			}

			if b.matchLabels(svc.Selector, pod.Labels) {
				b.graph.AddEdge(svc.UID, pod.UID, RelationSelects)
			}
		}
	}
}

// AddStatefulSets adds statefulsets to the graph
func (b *Builder) AddStatefulSets(stss []k8s.StatefulSetInfo) {
	for i := range stss {
		sts := &stss[i]
		status := StatusUnknown
		if sts.ReadyReplicas == sts.Replicas && sts.Replicas > 0 {
			status = StatusHealthy
		} else if sts.ReadyReplicas > 0 {
			status = StatusWarning
		} else if sts.Replicas > 0 {
			status = StatusError
		}
		b.graph.AddNode(&Node{
			UID: sts.UID, Kind: "StatefulSet", Name: sts.Name,
			Namespace: sts.Namespace, Status: status,
			Extras: map[string]string{
				"info": fmt.Sprintf("%d/%d", sts.ReadyReplicas, sts.Replicas),
			},
		})
	}
}

// AddDaemonSets adds daemonsets to the graph
func (b *Builder) AddDaemonSets(dss []k8s.DaemonSetInfo) {
	for i := range dss {
		ds := &dss[i]
		status := StatusHealthy
		if ds.ReadyNumber < ds.DesiredNumber {
			status = StatusWarning
		}
		if ds.ReadyNumber == 0 && ds.DesiredNumber > 0 {
			status = StatusError
		}
		b.graph.AddNode(&Node{
			UID: ds.UID, Kind: "DaemonSet", Name: ds.Name,
			Namespace: ds.Namespace, Status: status,
			Extras: map[string]string{
				"info": fmt.Sprintf("%d/%d", ds.ReadyNumber, ds.DesiredNumber),
			},
		})
	}
}

// AddReplicaSets adds replicasets to the graph
func (b *Builder) AddReplicaSets(rss []k8s.ReplicaSetInfo) {
	for i := range rss {
		rs := &rss[i]
		status := StatusUnknown
		if rs.ReadyReplicas == rs.DesiredReplicas && rs.DesiredReplicas > 0 {
			status = StatusHealthy
		} else if rs.ReadyReplicas > 0 {
			status = StatusWarning
		} else if rs.DesiredReplicas > 0 {
			status = StatusError
		}
		b.graph.AddNode(&Node{
			UID: rs.UID, Kind: "ReplicaSet", Name: rs.Name,
			Namespace: rs.Namespace, Status: status,
			Extras: map[string]string{
				"info": fmt.Sprintf("%d/%d", rs.ReadyReplicas, rs.DesiredReplicas),
			},
		})
		// Add owner relationships
		for _, owner := range rs.OwnerRefs {
			b.graph.AddEdge(owner.UID, rs.UID, RelationOwns)
		}
	}
}

// AddJobs adds jobs to the graph
func (b *Builder) AddJobs(jobs []k8s.JobInfo) {
	for i := range jobs {
		job := &jobs[i]
		status := StatusUnknown
		switch job.Status {
		case "Complete":
			status = StatusHealthy
		case "Failed":
			status = StatusError
		case "Running":
			status = StatusHealthy
		}
		if job.Active > 0 {
			status = StatusHealthy
		}
		b.graph.AddNode(&Node{
			UID: job.UID, Kind: "Job", Name: job.Name,
			Namespace: job.Namespace, Status: status,
		})
		for _, owner := range job.OwnerRefs {
			b.graph.AddEdge(owner.UID, job.UID, RelationOwns)
		}
	}
}

// AddCronJobs adds cronjobs to the graph
func (b *Builder) AddCronJobs(cjs []k8s.CronJobInfo) {
	for i := range cjs {
		cj := &cjs[i]
		status := StatusHealthy
		if cj.Suspend {
			status = StatusWarning
		}
		b.graph.AddNode(&Node{
			UID: cj.UID, Kind: "CronJob", Name: cj.Name,
			Namespace: cj.Namespace, Status: status,
		})
	}
}

// AddIngresses adds ingresses to the graph
func (b *Builder) AddIngresses(ings []k8s.IngressInfo) {
	for i := range ings {
		ing := &ings[i]
		b.graph.AddNode(&Node{
			UID: ing.UID, Kind: "Ingress", Name: ing.Name,
			Namespace: ing.Namespace, Status: StatusHealthy,
		})
	}
}

// AddConfigMaps adds configmaps to the graph
func (b *Builder) AddConfigMaps(cms []k8s.ConfigMapInfo) {
	for i := range cms {
		cm := &cms[i]
		b.graph.AddNode(&Node{
			UID: cm.UID, Kind: "ConfigMap", Name: cm.Name,
			Namespace: cm.Namespace, Status: StatusHealthy,
		})
	}
}

// AddSecrets adds secrets to the graph
func (b *Builder) AddSecrets(secs []k8s.SecretInfo) {
	for i := range secs {
		sec := &secs[i]
		b.graph.AddNode(&Node{
			UID: sec.UID, Kind: "Secret", Name: sec.Name,
			Namespace: sec.Namespace, Status: StatusHealthy,
		})
	}
}

// AddPVCs adds persistent volume claims to the graph
func (b *Builder) AddPVCs(pvcs []k8s.PVCInfo) {
	for i := range pvcs {
		pvc := &pvcs[i]
		status := StatusHealthy
		if pvc.Status == "Pending" {
			status = StatusPending
		} else if pvc.Status == "Lost" {
			status = StatusError
		}
		b.graph.AddNode(&Node{
			UID: pvc.UID, Kind: "PersistentVolumeClaim", Name: pvc.Name,
			Namespace: pvc.Namespace, Status: status,
		})
	}
}

// AddPVs adds persistent volumes to the graph
func (b *Builder) AddPVs(pvs []k8s.PVInfo) {
	for i := range pvs {
		pv := &pvs[i]
		status := StatusHealthy
		if pv.Status == "Pending" || pv.Status == "Available" {
			status = StatusPending
		} else if pv.Status == "Failed" {
			status = StatusError
		}
		b.graph.AddNode(&Node{
			UID: pv.UID, Kind: "PersistentVolume", Name: pv.Name,
			Namespace: "", Status: status,
		})
	}
}

// AddHPAs adds horizontal pod autoscalers to the graph
func (b *Builder) AddHPAs(hpas []k8s.HPAInfo) {
	for i := range hpas {
		hpa := &hpas[i]
		b.graph.AddNode(&Node{
			UID: hpa.UID, Kind: "HorizontalPodAutoscaler", Name: hpa.Name,
			Namespace: hpa.Namespace, Status: StatusHealthy,
		})
	}
}

// AddContainers adds synthetic container nodes under pods
func (b *Builder) AddContainers(pods []k8s.PodInfo) {
	for i := range pods {
		pod := &pods[i]
		for _, c := range pod.Containers {
			uid := pod.UID + "/co/" + c.Name
			status := StatusHealthy
			if !c.Ready {
				status = StatusWarning
			}
			if c.StateReason == "CrashLoopBackOff" || c.StateReason == "OOMKilled" {
				status = StatusError
			}
			if c.State == "waiting" {
				status = StatusPending
			}
			b.graph.AddNode(&Node{
				UID: uid, Kind: "Container", Name: c.Name,
				Namespace: pod.Namespace, Status: status,
			})
			b.graph.AddEdge(pod.UID, uid, RelationOwns)
		}
		for _, c := range pod.InitContainers {
			uid := pod.UID + "/ic/" + c.Name
			status := StatusHealthy
			if !c.Ready {
				status = StatusWarning
			}
			if c.StateReason == "CrashLoopBackOff" || c.StateReason == "OOMKilled" {
				status = StatusError
			}
			b.graph.AddNode(&Node{
				UID: uid, Kind: "Container", Name: c.Name,
				Namespace: pod.Namespace, Status: status,
			})
			b.graph.AddEdge(pod.UID, uid, RelationOwns)
		}
	}
}

// AddK8sNodes adds Kubernetes worker nodes to the graph
func (b *Builder) AddK8sNodes(nodes []k8s.NodeInfo) {
	for i := range nodes {
		n := &nodes[i]
		status := StatusHealthy
		if n.Status != "Ready" {
			status = StatusError
		}
		b.graph.AddNode(&Node{
			UID: n.UID, Kind: "Node", Name: n.Name,
			Namespace: "", Status: status,
		})
	}
}

// LinkIngressesToServices creates edges from ingresses to the services they route to
func (b *Builder) LinkIngressesToServices(ings []k8s.IngressInfo) {
	for _, ing := range ings {
		for _, rule := range ing.Rules {
			for _, path := range rule.Paths {
				if path.ServiceName == "" {
					continue
				}
				// Find matching service node by name in same namespace
				for _, node := range b.graph.Nodes {
					if node.Kind == "Service" && node.Name == path.ServiceName && node.Namespace == ing.Namespace {
						b.graph.AddEdge(ing.UID, node.UID, RelationRoutes)
						break
					}
				}
			}
		}
	}
}

// LinkPVCsToPVs creates edges from PVCs to their bound PVs
func (b *Builder) LinkPVCsToPVs(pvcs []k8s.PVCInfo) {
	for _, pvc := range pvcs {
		if pvc.Volume == "" {
			continue
		}
		for _, node := range b.graph.Nodes {
			if node.Kind == "PersistentVolume" && node.Name == pvc.Volume {
				b.graph.AddEdge(pvc.UID, node.UID, RelationBinds)
				break
			}
		}
	}
}

// LinkHPAsToTargets creates edges from HPAs to their target resources
func (b *Builder) LinkHPAsToTargets(hpas []k8s.HPAInfo) {
	for _, hpa := range hpas {
		if hpa.Reference == "" {
			continue
		}
		// HPAInfo.Reference is "Kind/Name" e.g. "Deployment/nginx"
		parts := strings.SplitN(hpa.Reference, "/", 2)
		if len(parts) != 2 {
			continue
		}
		targetKind, targetName := parts[0], parts[1]
		for _, node := range b.graph.Nodes {
			if node.Name == targetName && node.Namespace == hpa.Namespace &&
				strings.EqualFold(node.Kind, targetKind) {
				b.graph.AddEdge(hpa.UID, node.UID, RelationTargets)
				break
			}
		}
	}
}

// LinkStatefulSetsToServices creates edges from StatefulSets to their headless services
func (b *Builder) LinkStatefulSetsToServices(stss []k8s.StatefulSetInfo) {
	for _, sts := range stss {
		if sts.ServiceName == "" {
			continue
		}
		for _, node := range b.graph.Nodes {
			if node.Kind == "Service" && node.Name == sts.ServiceName && node.Namespace == sts.Namespace {
				b.graph.AddEdge(node.UID, sts.UID, RelationSelects)
				break
			}
		}
	}
}

// LinkPodsToVolumes creates edges from pods to ConfigMaps, Secrets, and PVCs used as volumes.
// Uses typed PodInfo fields (VolumeSecrets, VolumeConfigMaps, VolumePVCs).
func (b *Builder) LinkPodsToVolumes(pods []k8s.PodInfo) {
	for _, pod := range pods {
		for _, name := range pod.VolumeConfigMaps {
			for _, node := range b.graph.Nodes {
				if node.Kind == "ConfigMap" && node.Name == name && node.Namespace == pod.Namespace {
					b.graph.AddEdge(pod.UID, node.UID, RelationMounts)
					break
				}
			}
		}
		for _, name := range pod.VolumeSecrets {
			for _, node := range b.graph.Nodes {
				if node.Kind == "Secret" && node.Name == name && node.Namespace == pod.Namespace {
					b.graph.AddEdge(pod.UID, node.UID, RelationMounts)
					break
				}
			}
		}
		for _, name := range pod.VolumePVCs {
			for _, node := range b.graph.Nodes {
				if node.Kind == "PersistentVolumeClaim" && node.Name == name && node.Namespace == pod.Namespace {
					b.graph.AddEdge(pod.UID, node.UID, RelationMounts)
					break
				}
			}
		}
	}
}

// AddServiceAccounts adds synthetic ServiceAccount nodes under pods
func (b *Builder) AddServiceAccounts(pods []k8s.PodInfo) {
	for i := range pods {
		pod := &pods[i]
		saName := pod.ServiceAccountName
		if saName == "" {
			saName = "default"
		}
		uid := pod.UID + "/sa/" + saName
		b.graph.AddNode(&Node{
			UID: uid, Kind: "ServiceAccount", Name: saName,
			Namespace: pod.Namespace, Status: StatusHealthy,
		})
		b.graph.AddEdge(pod.UID, uid, RelationUses)
	}
}

// LinkContainersToEnvRefs creates edges from container nodes to Secrets/ConfigMaps
// referenced via env vars (envFrom, env[].valueFrom).
func (b *Builder) LinkContainersToEnvRefs(pods []k8s.PodInfo) {
	for _, pod := range pods {
		for _, c := range pod.Containers {
			containerUID := pod.UID + "/co/" + c.Name
			for _, name := range c.EnvRefSecrets {
				for _, node := range b.graph.Nodes {
					if node.Kind == "Secret" && node.Name == name && node.Namespace == pod.Namespace {
						b.graph.AddEdge(containerUID, node.UID, RelationReferences)
						break
					}
				}
			}
			for _, name := range c.EnvRefConfigMaps {
				for _, node := range b.graph.Nodes {
					if node.Kind == "ConfigMap" && node.Name == name && node.Namespace == pod.Namespace {
						b.graph.AddEdge(containerUID, node.UID, RelationReferences)
						break
					}
				}
			}
		}
		for _, c := range pod.InitContainers {
			containerUID := pod.UID + "/ic/" + c.Name
			for _, name := range c.EnvRefSecrets {
				for _, node := range b.graph.Nodes {
					if node.Kind == "Secret" && node.Name == name && node.Namespace == pod.Namespace {
						b.graph.AddEdge(containerUID, node.UID, RelationReferences)
						break
					}
				}
			}
			for _, name := range c.EnvRefConfigMaps {
				for _, node := range b.graph.Nodes {
					if node.Kind == "ConfigMap" && node.Name == name && node.Namespace == pod.Namespace {
						b.graph.AddEdge(containerUID, node.UID, RelationReferences)
						break
					}
				}
			}
		}
	}
}

// LinkPodsToNodes creates edges from pods to the nodes they're scheduled on
func (b *Builder) LinkPodsToNodes(pods []k8s.PodInfo) {
	for _, pod := range pods {
		if pod.NodeName == "" {
			continue
		}
		for _, node := range b.graph.Nodes {
			if node.Kind == "Node" && node.Name == pod.NodeName {
				b.graph.AddEdge(pod.UID, node.UID, RelationUses)
				break
			}
		}
	}
}

func (b *Builder) addResource(r *k8s.Resource) {
	node := &Node{
		UID:       r.UID,
		Kind:      r.Kind,
		Name:      r.Name,
		Namespace: r.Namespace,
		Status:    b.getResourceStatus(r),
		Resource:  r,
	}
	b.graph.AddNode(node)

	// Add owner relationships
	for _, owner := range r.OwnerRefs {
		b.graph.AddEdge(owner.UID, r.UID, RelationOwns)
	}
}

func (b *Builder) getPodStatus(pod *k8s.PodInfo) NodeStatus {
	switch pod.Phase {
	case "Running":
		// Check if all containers are ready
		for _, c := range pod.Containers {
			if !c.Ready {
				return StatusWarning
			}
			if c.StateReason == "CrashLoopBackOff" || c.StateReason == "OOMKilled" {
				return StatusError
			}
		}
		return StatusHealthy
	case "Pending":
		return StatusPending
	case "Failed":
		return StatusError
	case "Succeeded":
		return StatusHealthy
	default:
		return StatusUnknown
	}
}

func (b *Builder) getDeploymentStatus(dep *k8s.DeploymentInfo) NodeStatus {
	if dep.ReadyReplicas == dep.Replicas && dep.Replicas > 0 {
		return StatusHealthy
	}
	if dep.ReadyReplicas < dep.Replicas && dep.ReadyReplicas > 0 {
		return StatusWarning
	}
	if dep.ReadyReplicas == 0 && dep.Replicas > 0 {
		return StatusError
	}
	return StatusUnknown
}

func (b *Builder) getResourceStatus(r *k8s.Resource) NodeStatus {
	// Check conditions if available
	for _, c := range r.Conditions {
		if c.Type == "Ready" || c.Type == "Available" {
			if c.Status == "True" {
				return StatusHealthy
			} else if c.Status == "False" {
				return StatusError
			}
		}
	}
	return StatusUnknown
}

func (b *Builder) matchLabels(selector, labels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}

	for k, v := range selector {
		if labelValue, ok := labels[k]; !ok || labelValue != v {
			return false
		}
	}
	return true
}

// CalculateDepths calculates the depth (distance from root) for all nodes
func (b *Builder) CalculateDepths() {
	// Start from roots (nodes with no parents)
	roots := b.graph.GetRoots()

	// BFS to calculate depths
	visited := make(map[string]bool)
	queue := make([]*Node, 0)

	for _, root := range roots {
		root.Depth = 0
		visited[root.UID] = true
		queue = append(queue, root)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, child := range b.graph.GetChildren(current.UID) {
			if !visited[child.UID] {
				child.Depth = current.Depth + 1
				visited[child.UID] = true
				queue = append(queue, child)
			}
		}
	}
}
