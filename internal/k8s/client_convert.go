package k8s

import (
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (c *K8sClient) configMapToConfigMapInfo(cm *corev1.ConfigMap) ConfigMapInfo {
	info := ConfigMapInfo{
		Resource: Resource{
			UID:         string(cm.UID),
			APIVersion:  "v1",
			Kind:        "ConfigMap",
			Namespace:   cm.Namespace,
			Name:        cm.Name,
			ClusterID:   c.clusterID,
			Labels:      cm.Labels,
			Annotations: cm.Annotations,
			FetchedAt:   time.Now(),
		},
		DataCount: len(cm.Data) + len(cm.BinaryData),
	}

	if !cm.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(cm.CreationTimestamp.Time)
	}

	return info
}

func (c *K8sClient) secretToSecretInfo(secret *corev1.Secret) SecretInfo {
	info := SecretInfo{
		Resource: Resource{
			UID:         string(secret.UID),
			APIVersion:  "v1",
			Kind:        "Secret",
			Namespace:   secret.Namespace,
			Name:        secret.Name,
			ClusterID:   c.clusterID,
			Labels:      secret.Labels,
			Annotations: secret.Annotations,
			FetchedAt:   time.Now(),
		},
		Type:      string(secret.Type),
		DataCount: len(secret.Data),
	}

	if !secret.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(secret.CreationTimestamp.Time)
	}

	return info
}

func (c *K8sClient) ingressToIngressInfo(ing *networkingv1.Ingress) IngressInfo {
	info := IngressInfo{
		Resource: Resource{
			UID:         string(ing.UID),
			APIVersion:  "networking.k8s.io/v1",
			Kind:        "Ingress",
			Namespace:   ing.Namespace,
			Name:        ing.Name,
			ClusterID:   c.clusterID,
			Labels:      ing.Labels,
			Annotations: ing.Annotations,
			FetchedAt:   time.Now(),
		},
	}

	if !ing.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(ing.CreationTimestamp.Time)
	}

	// Ingress class
	if ing.Spec.IngressClassName != nil {
		info.Class = *ing.Spec.IngressClassName
	}

	// Collect hosts and rules
	var hosts []string
	for _, rule := range ing.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}

		ingressRule := IngressRule{Host: rule.Host}
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				pathType := "Prefix"
				if path.PathType != nil {
					pathType = string(*path.PathType)
				}
				serviceName := ""
				servicePort := ""
				if path.Backend.Service != nil {
					serviceName = path.Backend.Service.Name
					if path.Backend.Service.Port.Number != 0 {
						servicePort = fmt.Sprintf("%d", path.Backend.Service.Port.Number)
					} else {
						servicePort = path.Backend.Service.Port.Name
					}
				}
				ingressRule.Paths = append(ingressRule.Paths, IngressPath{
					Path:        path.Path,
					PathType:    pathType,
					ServiceName: serviceName,
					ServicePort: servicePort,
				})
			}
		}
		info.Rules = append(info.Rules, ingressRule)
	}
	info.Hosts = hosts

	// Load balancer address
	var addresses []string
	for _, lbi := range ing.Status.LoadBalancer.Ingress {
		if lbi.IP != "" {
			addresses = append(addresses, lbi.IP)
		} else if lbi.Hostname != "" {
			addresses = append(addresses, lbi.Hostname)
		}
	}
	info.Address = strings.Join(addresses, ",")

	// Ports - typically 80/443 for ingress
	info.Ports = "80, 443"

	return info
}

func (c *K8sClient) pvcToPVCInfo(pvc *corev1.PersistentVolumeClaim) PVCInfo {
	info := PVCInfo{
		Resource: Resource{
			UID:         string(pvc.UID),
			APIVersion:  "v1",
			Kind:        "PersistentVolumeClaim",
			Namespace:   pvc.Namespace,
			Name:        pvc.Name,
			ClusterID:   c.clusterID,
			Labels:      pvc.Labels,
			Annotations: pvc.Annotations,
			FetchedAt:   time.Now(),
		},
		Status: string(pvc.Status.Phase),
		Volume: pvc.Spec.VolumeName,
	}

	if !pvc.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(pvc.CreationTimestamp.Time)
	}

	// Capacity
	if storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
		info.Capacity = storage.String()
	}

	// Access modes
	for _, mode := range pvc.Spec.AccessModes {
		switch mode {
		case corev1.ReadWriteOnce:
			info.AccessModes = append(info.AccessModes, "RWO")
		case corev1.ReadOnlyMany:
			info.AccessModes = append(info.AccessModes, "ROX")
		case corev1.ReadWriteMany:
			info.AccessModes = append(info.AccessModes, "RWX")
		case corev1.ReadWriteOncePod:
			info.AccessModes = append(info.AccessModes, "RWOP")
		}
	}

	// Storage class
	if pvc.Spec.StorageClassName != nil {
		info.StorageClass = *pvc.Spec.StorageClassName
	}

	return info
}

func (c *K8sClient) statefulSetToStatefulSetInfo(sts *appsv1.StatefulSet) StatefulSetInfo {
	info := StatefulSetInfo{
		Resource: Resource{
			UID:         string(sts.UID),
			APIVersion:  "apps/v1",
			Kind:        "StatefulSet",
			Namespace:   sts.Namespace,
			Name:        sts.Name,
			ClusterID:   c.clusterID,
			Labels:      sts.Labels,
			Annotations: sts.Annotations,
			FetchedAt:   time.Now(),
		},
		Replicas:      1, // default if nil
		ReadyReplicas: sts.Status.ReadyReplicas,
		ServiceName:   sts.Spec.ServiceName,
	}

	if sts.Spec.Replicas != nil {
		info.Replicas = *sts.Spec.Replicas
	}

	if !sts.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(sts.CreationTimestamp.Time)
	}

	return info
}

func (c *K8sClient) podToPodInfo(pod *corev1.Pod) PodInfo {
	info := PodInfo{
		Resource: Resource{
			UID:         string(pod.UID),
			APIVersion:  "v1",
			Kind:        "Pod",
			Namespace:   pod.Namespace,
			Name:        pod.Name,
			ClusterID:   c.clusterID,
			Labels:      pod.Labels,
			Annotations: pod.Annotations,
			FetchedAt:   time.Now(),
		},
		Phase:    string(pod.Status.Phase),
		NodeName: pod.Spec.NodeName,
		IP:       pod.Status.PodIP,
	}

	// A pod with a deletion timestamp is terminating
	if pod.DeletionTimestamp != nil {
		info.Phase = "Terminating"
	}

	if pod.CreationTimestamp.Time.IsZero() {
		info.Age = 0
	} else {
		info.Age = time.Since(pod.CreationTimestamp.Time)
	}

	// Calculate ready count and restarts
	readyCount := 0
	totalContainers := len(pod.Spec.Containers)
	var totalRestarts int32

	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			readyCount++
		}
		totalRestarts += cs.RestartCount

		state := "Unknown"
		stateReason := ""
		stateMessage := ""
		if cs.State.Running != nil {
			state = "Running"
		} else if cs.State.Waiting != nil {
			state = "Waiting"
			stateReason = cs.State.Waiting.Reason
			stateMessage = cs.State.Waiting.Message
		} else if cs.State.Terminated != nil {
			state = "Terminated"
			stateReason = cs.State.Terminated.Reason
			stateMessage = cs.State.Terminated.Message
		}

		var lastTerminatedAt time.Time
		if cs.LastTerminationState.Terminated != nil {
			lastTerminatedAt = cs.LastTerminationState.Terminated.FinishedAt.Time
		}

		info.Containers = append(info.Containers, ContainerInfo{
			Name:             cs.Name,
			Image:            cs.Image,
			Ready:            cs.Ready,
			RestartCount:     cs.RestartCount,
			State:            state,
			StateReason:      stateReason,
			StateMessage:     stateMessage,
			LastTerminatedAt: lastTerminatedAt,
		})
	}

	for _, cs := range pod.Status.InitContainerStatuses {
		state := "Unknown"
		stateReason := ""
		stateMessage := ""
		if cs.State.Running != nil {
			state = "Running"
		} else if cs.State.Waiting != nil {
			state = "Waiting"
			stateReason = cs.State.Waiting.Reason
			stateMessage = cs.State.Waiting.Message
		} else if cs.State.Terminated != nil {
			state = "Terminated"
			stateReason = cs.State.Terminated.Reason
			stateMessage = cs.State.Terminated.Message
		}

		info.InitContainers = append(info.InitContainers, ContainerInfo{
			Name:         cs.Name,
			Image:        cs.Image,
			Ready:        cs.Ready,
			RestartCount: cs.RestartCount,
			State:        state,
			StateReason:  stateReason,
			StateMessage: stateMessage,
		})
	}

	info.Ready = fmt.Sprintf("%d/%d", readyCount, totalContainers)
	info.Restarts = totalRestarts

	// Pod-level fields
	info.HostNetwork = pod.Spec.HostNetwork

	// Determine pod-level RunAsNonRoot default
	var podRunAsNonRoot *bool
	if pod.Spec.SecurityContext != nil {
		podRunAsNonRoot = pod.Spec.SecurityContext.RunAsNonRoot
	}

	// Extract resource requests/limits and ports from pod spec
	for i, specContainer := range pod.Spec.Containers {
		cpuReq := specContainer.Resources.Requests[corev1.ResourceCPU]
		cpuLim := specContainer.Resources.Limits[corev1.ResourceCPU]
		memReq := specContainer.Resources.Requests[corev1.ResourceMemory]
		memLim := specContainer.Resources.Limits[corev1.ResourceMemory]

		cpuReqNanos := cpuReq.MilliValue() * 1_000_000
		cpuLimNanos := cpuLim.MilliValue() * 1_000_000
		memReqBytes := memReq.Value()
		memLimBytes := memLim.Value()

		info.CPURequest += cpuReqNanos
		info.CPULimit += cpuLimNanos
		info.MemRequest += memReqBytes
		info.MemLimit += memLimBytes

		// Populate per-container fields if matching by index
		if i < len(info.Containers) {
			info.Containers[i].CPURequest = cpuReqNanos
			info.Containers[i].CPULimit = cpuLimNanos
			info.Containers[i].MemRequest = memReqBytes
			info.Containers[i].MemLimit = memLimBytes

			// Populate container ports
			for _, p := range specContainer.Ports {
				info.Containers[i].Ports = append(info.Containers[i].Ports, ContainerPort{
					Name:          p.Name,
					ContainerPort: p.ContainerPort,
					Protocol:      string(p.Protocol),
				})
			}

			// Probes
			info.Containers[i].HasLivenessProbe = specContainer.LivenessProbe != nil
			info.Containers[i].HasReadinessProbe = specContainer.ReadinessProbe != nil

			// Security context
			if specContainer.SecurityContext != nil {
				if specContainer.SecurityContext.Privileged != nil {
					info.Containers[i].Privileged = *specContainer.SecurityContext.Privileged
				}
				info.Containers[i].RunAsNonRoot = specContainer.SecurityContext.RunAsNonRoot
			}
			// Fall back to pod-level RunAsNonRoot if container doesn't override
			if info.Containers[i].RunAsNonRoot == nil && podRunAsNonRoot != nil {
				info.Containers[i].RunAsNonRoot = podRunAsNonRoot
			}
		}
	}

	// Service account
	info.ServiceAccountName = pod.Spec.ServiceAccountName

	// Extract volume references (Secret, ConfigMap, PVC)
	for _, vol := range pod.Spec.Volumes {
		if vol.Secret != nil {
			info.VolumeSecrets = append(info.VolumeSecrets, vol.Secret.SecretName)
		}
		if vol.ConfigMap != nil {
			info.VolumeConfigMaps = append(info.VolumeConfigMaps, vol.ConfigMap.Name)
		}
		if vol.PersistentVolumeClaim != nil {
			info.VolumePVCs = append(info.VolumePVCs, vol.PersistentVolumeClaim.ClaimName)
		}
	}

	// Extract container env var references (Secret, ConfigMap)
	for i, specContainer := range pod.Spec.Containers {
		if i >= len(info.Containers) {
			break
		}
		secretsSeen := make(map[string]bool)
		cmSeen := make(map[string]bool)
		for _, envFrom := range specContainer.EnvFrom {
			if envFrom.SecretRef != nil && !secretsSeen[envFrom.SecretRef.Name] {
				info.Containers[i].EnvRefSecrets = append(info.Containers[i].EnvRefSecrets, envFrom.SecretRef.Name)
				secretsSeen[envFrom.SecretRef.Name] = true
			}
			if envFrom.ConfigMapRef != nil && !cmSeen[envFrom.ConfigMapRef.Name] {
				info.Containers[i].EnvRefConfigMaps = append(info.Containers[i].EnvRefConfigMaps, envFrom.ConfigMapRef.Name)
				cmSeen[envFrom.ConfigMapRef.Name] = true
			}
		}
		for _, env := range specContainer.Env {
			if env.ValueFrom != nil {
				if env.ValueFrom.SecretKeyRef != nil && !secretsSeen[env.ValueFrom.SecretKeyRef.Name] {
					info.Containers[i].EnvRefSecrets = append(info.Containers[i].EnvRefSecrets, env.ValueFrom.SecretKeyRef.Name)
					secretsSeen[env.ValueFrom.SecretKeyRef.Name] = true
				}
				if env.ValueFrom.ConfigMapKeyRef != nil && !cmSeen[env.ValueFrom.ConfigMapKeyRef.Name] {
					info.Containers[i].EnvRefConfigMaps = append(info.Containers[i].EnvRefConfigMaps, env.ValueFrom.ConfigMapKeyRef.Name)
					cmSeen[env.ValueFrom.ConfigMapKeyRef.Name] = true
				}
			}
		}
	}

	// Extract init container env var references
	for i, specContainer := range pod.Spec.InitContainers {
		if i >= len(info.InitContainers) {
			break
		}
		secretsSeen := make(map[string]bool)
		cmSeen := make(map[string]bool)
		for _, envFrom := range specContainer.EnvFrom {
			if envFrom.SecretRef != nil && !secretsSeen[envFrom.SecretRef.Name] {
				info.InitContainers[i].EnvRefSecrets = append(info.InitContainers[i].EnvRefSecrets, envFrom.SecretRef.Name)
				secretsSeen[envFrom.SecretRef.Name] = true
			}
			if envFrom.ConfigMapRef != nil && !cmSeen[envFrom.ConfigMapRef.Name] {
				info.InitContainers[i].EnvRefConfigMaps = append(info.InitContainers[i].EnvRefConfigMaps, envFrom.ConfigMapRef.Name)
				cmSeen[envFrom.ConfigMapRef.Name] = true
			}
		}
		for _, env := range specContainer.Env {
			if env.ValueFrom != nil {
				if env.ValueFrom.SecretKeyRef != nil && !secretsSeen[env.ValueFrom.SecretKeyRef.Name] {
					info.InitContainers[i].EnvRefSecrets = append(info.InitContainers[i].EnvRefSecrets, env.ValueFrom.SecretKeyRef.Name)
					secretsSeen[env.ValueFrom.SecretKeyRef.Name] = true
				}
				if env.ValueFrom.ConfigMapKeyRef != nil && !cmSeen[env.ValueFrom.ConfigMapKeyRef.Name] {
					info.InitContainers[i].EnvRefConfigMaps = append(info.InitContainers[i].EnvRefConfigMaps, env.ValueFrom.ConfigMapKeyRef.Name)
					cmSeen[env.ValueFrom.ConfigMapKeyRef.Name] = true
				}
			}
		}
	}

	// Convert owner references
	for _, ref := range pod.OwnerReferences {
		info.OwnerRefs = append(info.OwnerRefs, OwnerReference{
			Kind: ref.Kind,
			Name: ref.Name,
			UID:  string(ref.UID),
		})
	}

	return info
}

func (c *K8sClient) deploymentToDeploymentInfo(dep *appsv1.Deployment) DeploymentInfo {
	info := DeploymentInfo{
		Resource: Resource{
			UID:         string(dep.UID),
			APIVersion:  "apps/v1",
			Kind:        "Deployment",
			Namespace:   dep.Namespace,
			Name:        dep.Name,
			ClusterID:   c.clusterID,
			Labels:      dep.Labels,
			Annotations: dep.Annotations,
			FetchedAt:   time.Now(),
		},
		Replicas:          1, // default if nil
		ReadyReplicas:     dep.Status.ReadyReplicas,
		UpdatedReplicas:   dep.Status.UpdatedReplicas,
		AvailableReplicas: dep.Status.AvailableReplicas,
		Strategy:          string(dep.Spec.Strategy.Type),
	}

	if dep.Spec.Replicas != nil {
		info.Replicas = *dep.Spec.Replicas
	}

	if !dep.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(dep.CreationTimestamp.Time)
	}

	return info
}

func (c *K8sClient) serviceToServiceInfo(svc *corev1.Service) ServiceInfo {
	info := ServiceInfo{
		Resource: Resource{
			UID:         string(svc.UID),
			APIVersion:  "v1",
			Kind:        "Service",
			Namespace:   svc.Namespace,
			Name:        svc.Name,
			ClusterID:   c.clusterID,
			Labels:      svc.Labels,
			Annotations: svc.Annotations,
			FetchedAt:   time.Now(),
		},
		Type:      string(svc.Spec.Type),
		ClusterIP: svc.Spec.ClusterIP,
		Selector:  svc.Spec.Selector,
	}

	if !svc.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(svc.CreationTimestamp.Time)
	}

	// External IPs
	if len(svc.Spec.ExternalIPs) > 0 {
		info.ExternalIP = strings.Join(svc.Spec.ExternalIPs, ",")
	} else if len(svc.Status.LoadBalancer.Ingress) > 0 {
		var ips []string
		for _, ing := range svc.Status.LoadBalancer.Ingress {
			if ing.IP != "" {
				ips = append(ips, ing.IP)
			} else if ing.Hostname != "" {
				ips = append(ips, ing.Hostname)
			}
		}
		info.ExternalIP = strings.Join(ips, ",")
	}

	// Ports
	for _, port := range svc.Spec.Ports {
		info.Ports = append(info.Ports, ServicePort{
			Name:       port.Name,
			Port:       port.Port,
			TargetPort: port.TargetPort.String(),
			Protocol:   string(port.Protocol),
			NodePort:   port.NodePort,
		})
	}

	return info
}

func (c *K8sClient) nodeToNodeInfo(node *corev1.Node) NodeInfo {
	info := NodeInfo{
		Resource: Resource{
			UID:         string(node.UID),
			APIVersion:  "v1",
			Kind:        "Node",
			Name:        node.Name,
			ClusterID:   c.clusterID,
			Labels:      node.Labels,
			Annotations: node.Annotations,
			FetchedAt:   time.Now(),
		},
		Version: node.Status.NodeInfo.KubeletVersion,
		OS:      node.Status.NodeInfo.OperatingSystem,
		Arch:    node.Status.NodeInfo.Architecture,
	}

	if !node.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(node.CreationTimestamp.Time)
	}

	// Determine node status from conditions
	info.Status = "Unknown"
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				info.Status = "Ready"
			} else {
				info.Status = "NotReady"
			}
			break
		}
	}

	// Extract roles from labels
	for label := range node.Labels {
		if strings.HasPrefix(label, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
			if role != "" {
				info.Roles = append(info.Roles, role)
			}
		}
	}
	if len(info.Roles) == 0 {
		info.Roles = []string{"<none>"}
	}

	// Extract IPs
	for _, addr := range node.Status.Addresses {
		switch addr.Type {
		case corev1.NodeInternalIP:
			info.InternalIP = addr.Address
		case corev1.NodeExternalIP:
			info.ExternalIP = addr.Address
		}
	}

	// Extract taints
	for _, taint := range node.Spec.Taints {
		info.Taints = append(info.Taints, fmt.Sprintf("%s:%s", taint.Key, taint.Effect))
	}
	if len(info.Taints) == 0 {
		info.Taints = []string{"<none>"}
	}

	// Extract allocatable resources
	if cpuAlloc, ok := node.Status.Allocatable[corev1.ResourceCPU]; ok {
		info.CPUAllocatable = cpuAlloc.MilliValue() * 1000000 // milli to nano
	}
	if memAlloc, ok := node.Status.Allocatable[corev1.ResourceMemory]; ok {
		info.MemAllocatable = memAlloc.Value()
	}

	return info
}

func (c *K8sClient) eventToEventInfo(event *corev1.Event) EventInfo {
	info := EventInfo{
		Resource: Resource{
			UID:         string(event.UID),
			APIVersion:  "v1",
			Kind:        "Event",
			Namespace:   event.Namespace,
			Name:        event.Name,
			ClusterID:   c.clusterID,
			Labels:      event.Labels,
			Annotations: event.Annotations,
			FetchedAt:   time.Now(),
		},
		Type:       event.Type,
		Reason:     event.Reason,
		Message:    event.Message,
		ObjectKind: event.InvolvedObject.Kind,
		ObjectName: event.InvolvedObject.Name,
		Count:      event.Count,
	}

	if !event.FirstTimestamp.Time.IsZero() {
		info.FirstSeen = event.FirstTimestamp.Time
	} else if !event.EventTime.Time.IsZero() {
		info.FirstSeen = event.EventTime.Time
	}

	if !event.LastTimestamp.Time.IsZero() {
		info.LastSeen = event.LastTimestamp.Time
		info.Age = time.Since(event.LastTimestamp.Time)
	} else if !event.EventTime.Time.IsZero() {
		info.LastSeen = event.EventTime.Time
		info.Age = time.Since(event.EventTime.Time)
	}

	return info
}

func (c *K8sClient) replicaSetToReplicaSetInfo(rs *appsv1.ReplicaSet) ReplicaSetInfo {
	info := ReplicaSetInfo{
		Resource: Resource{
			UID:         string(rs.UID),
			APIVersion:  "apps/v1",
			Kind:        "ReplicaSet",
			Namespace:   rs.Namespace,
			Name:        rs.Name,
			ClusterID:   c.clusterID,
			Labels:      rs.Labels,
			Annotations: rs.Annotations,
			FetchedAt:   time.Now(),
		},
		ReadyReplicas:     rs.Status.ReadyReplicas,
		AvailableReplicas: rs.Status.AvailableReplicas,
	}

	if rs.Spec.Replicas != nil {
		info.DesiredReplicas = *rs.Spec.Replicas
	}

	if !rs.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(rs.CreationTimestamp.Time)
	}

	// Get owner info
	for _, ref := range rs.OwnerReferences {
		info.OwnerKind = ref.Kind
		info.OwnerName = ref.Name
		info.OwnerRefs = append(info.OwnerRefs, OwnerReference{
			Kind: ref.Kind,
			Name: ref.Name,
			UID:  string(ref.UID),
		})
	}

	return info
}

func (c *K8sClient) daemonSetToDaemonSetInfo(ds *appsv1.DaemonSet) DaemonSetInfo {
	info := DaemonSetInfo{
		Resource: Resource{
			UID:         string(ds.UID),
			APIVersion:  "apps/v1",
			Kind:        "DaemonSet",
			Namespace:   ds.Namespace,
			Name:        ds.Name,
			ClusterID:   c.clusterID,
			Labels:      ds.Labels,
			Annotations: ds.Annotations,
			FetchedAt:   time.Now(),
		},
		DesiredNumber:   ds.Status.DesiredNumberScheduled,
		CurrentNumber:   ds.Status.CurrentNumberScheduled,
		ReadyNumber:     ds.Status.NumberReady,
		AvailableNumber: ds.Status.NumberAvailable,
	}

	if !ds.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(ds.CreationTimestamp.Time)
	}

	// Extract node selector
	if ds.Spec.Template.Spec.NodeSelector != nil {
		var selectors []string
		for k, v := range ds.Spec.Template.Spec.NodeSelector {
			selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
		}
		info.NodeSelector = strings.Join(selectors, ",")
	}

	return info
}

func (c *K8sClient) jobToJobInfo(job *batchv1.Job) JobInfo {
	info := JobInfo{
		Resource: Resource{
			UID:         string(job.UID),
			APIVersion:  "batch/v1",
			Kind:        "Job",
			Namespace:   job.Namespace,
			Name:        job.Name,
			ClusterID:   c.clusterID,
			Labels:      job.Labels,
			Annotations: job.Annotations,
			FetchedAt:   time.Now(),
		},
		Succeeded: job.Status.Succeeded,
		Failed:    job.Status.Failed,
		Active:    job.Status.Active,
	}

	if job.Spec.Completions != nil {
		info.Completions = *job.Spec.Completions
	} else {
		info.Completions = 1 // Default for non-indexed jobs
	}

	if !job.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(job.CreationTimestamp.Time)
	}

	// Calculate duration
	if job.Status.StartTime != nil {
		if job.Status.CompletionTime != nil {
			info.Duration = job.Status.CompletionTime.Sub(job.Status.StartTime.Time)
		} else {
			info.Duration = time.Since(job.Status.StartTime.Time)
		}
	}

	// Determine status
	if info.Succeeded >= info.Completions {
		info.Status = "Complete"
	} else if info.Failed > 0 {
		info.Status = "Failed"
	} else if info.Active > 0 {
		info.Status = "Running"
	} else {
		info.Status = "Pending"
	}

	// Convert owner references (e.g., CronJob → Job)
	for _, ref := range job.OwnerReferences {
		info.OwnerRefs = append(info.OwnerRefs, OwnerReference{
			Kind: ref.Kind,
			Name: ref.Name,
			UID:  string(ref.UID),
		})
	}

	return info
}

func (c *K8sClient) cronJobToCronJobInfo(cj *batchv1.CronJob) CronJobInfo {
	info := CronJobInfo{
		Resource: Resource{
			UID:         string(cj.UID),
			APIVersion:  "batch/v1",
			Kind:        "CronJob",
			Namespace:   cj.Namespace,
			Name:        cj.Name,
			ClusterID:   c.clusterID,
			Labels:      cj.Labels,
			Annotations: cj.Annotations,
			FetchedAt:   time.Now(),
		},
		Schedule: cj.Spec.Schedule,
		Active:   int32(len(cj.Status.Active)),
	}

	if cj.Spec.Suspend != nil {
		info.Suspend = *cj.Spec.Suspend
	}

	if !cj.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(cj.CreationTimestamp.Time)
	}

	if cj.Status.LastScheduleTime != nil {
		info.LastSchedule = cj.Status.LastScheduleTime.Time
	}

	return info
}

func (c *K8sClient) unstructuredToResource(obj *unstructured.Unstructured) Resource {
	resource := Resource{
		UID:         string(obj.GetUID()),
		APIVersion:  obj.GetAPIVersion(),
		Kind:        obj.GetKind(),
		Namespace:   obj.GetNamespace(),
		Name:        obj.GetName(),
		ClusterID:   c.clusterID,
		Labels:      obj.GetLabels(),
		Annotations: obj.GetAnnotations(),
		Raw:         obj,
		FetchedAt:   time.Now(),
	}

	for _, ref := range obj.GetOwnerReferences() {
		resource.OwnerRefs = append(resource.OwnerRefs, OwnerReference{
			Kind: ref.Kind,
			Name: ref.Name,
			UID:  string(ref.UID),
		})
	}

	return resource
}

func (c *K8sClient) hpaToHPAInfo(hpa *autoscalingv2.HorizontalPodAutoscaler) HPAInfo {
	info := HPAInfo{
		Resource: Resource{
			UID:         string(hpa.UID),
			APIVersion:  "autoscaling/v2",
			Kind:        "HorizontalPodAutoscaler",
			Namespace:   hpa.Namespace,
			Name:        hpa.Name,
			ClusterID:   c.clusterID,
			Labels:      hpa.Labels,
			Annotations: hpa.Annotations,
			FetchedAt:   time.Now(),
		},
		Reference:   fmt.Sprintf("%s/%s", hpa.Spec.ScaleTargetRef.Kind, hpa.Spec.ScaleTargetRef.Name),
		MaxReplicas: hpa.Spec.MaxReplicas,
	}

	if !hpa.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(hpa.CreationTimestamp.Time)
	}

	if hpa.Spec.MinReplicas != nil {
		info.MinReplicas = *hpa.Spec.MinReplicas
	} else {
		info.MinReplicas = 1
	}

	if hpa.Status.CurrentReplicas != 0 {
		info.CurrentReplicas = hpa.Status.CurrentReplicas
	}

	// Build targets string from metrics
	var targets []string
	for _, metric := range hpa.Spec.Metrics {
		switch metric.Type {
		case autoscalingv2.ResourceMetricSourceType:
			if metric.Resource != nil && metric.Resource.Target.AverageUtilization != nil {
				current := "<unknown>"
				for _, status := range hpa.Status.CurrentMetrics {
					if status.Type == autoscalingv2.ResourceMetricSourceType &&
						status.Resource != nil &&
						status.Resource.Name == metric.Resource.Name &&
						status.Resource.Current.AverageUtilization != nil {
						current = fmt.Sprintf("%d%%", *status.Resource.Current.AverageUtilization)
					}
				}
				targets = append(targets, fmt.Sprintf("%s/%d%%", current, *metric.Resource.Target.AverageUtilization))
			}
		}
	}
	if len(targets) > 0 {
		info.Targets = strings.Join(targets, ", ")
	} else {
		info.Targets = "<none>"
	}

	return info
}

func (c *K8sClient) pvToPVInfo(pv *corev1.PersistentVolume) PVInfo {
	info := PVInfo{
		Resource: Resource{
			UID:         string(pv.UID),
			APIVersion:  "v1",
			Kind:        "PersistentVolume",
			Name:        pv.Name,
			ClusterID:   c.clusterID,
			Labels:      pv.Labels,
			Annotations: pv.Annotations,
			FetchedAt:   time.Now(),
		},
		Status:        string(pv.Status.Phase),
		Reason:        pv.Status.Reason,
		ReclaimPolicy: string(pv.Spec.PersistentVolumeReclaimPolicy),
	}

	if !pv.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(pv.CreationTimestamp.Time)
	}

	// Capacity
	if storage, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
		info.Capacity = storage.String()
	}

	// Access modes
	for _, mode := range pv.Spec.AccessModes {
		switch mode {
		case corev1.ReadWriteOnce:
			info.AccessModes = append(info.AccessModes, "RWO")
		case corev1.ReadOnlyMany:
			info.AccessModes = append(info.AccessModes, "ROX")
		case corev1.ReadWriteMany:
			info.AccessModes = append(info.AccessModes, "RWX")
		case corev1.ReadWriteOncePod:
			info.AccessModes = append(info.AccessModes, "RWOP")
		}
	}

	// Storage class
	info.StorageClass = pv.Spec.StorageClassName

	// Bound claim
	if pv.Spec.ClaimRef != nil {
		info.Claim = pv.Spec.ClaimRef.Namespace + "/" + pv.Spec.ClaimRef.Name
	}

	return info
}

func (c *K8sClient) roleBindingToRoleBindingInfo(rb *rbacv1.RoleBinding) RoleBindingInfo {
	info := RoleBindingInfo{
		Resource: Resource{
			UID:         string(rb.UID),
			APIVersion:  "rbac.authorization.k8s.io/v1",
			Kind:        "RoleBinding",
			Namespace:   rb.Namespace,
			Name:        rb.Name,
			ClusterID:   c.clusterID,
			Labels:      rb.Labels,
			Annotations: rb.Annotations,
			FetchedAt:   time.Now(),
		},
		RoleKind: rb.RoleRef.Kind,
		RoleName: rb.RoleRef.Name,
	}

	if !rb.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(rb.CreationTimestamp.Time)
	}

	var subjects []string
	for _, s := range rb.Subjects {
		switch s.Kind {
		case "ServiceAccount":
			if s.Namespace != "" {
				subjects = append(subjects, fmt.Sprintf("SA:%s/%s", s.Namespace, s.Name))
			} else {
				subjects = append(subjects, "SA:"+s.Name)
			}
		case "User":
			subjects = append(subjects, "User:"+s.Name)
		case "Group":
			subjects = append(subjects, "Group:"+s.Name)
		default:
			subjects = append(subjects, s.Kind+":"+s.Name)
		}
	}
	info.Subjects = strings.Join(subjects, ", ")

	return info
}

func (c *K8sClient) clusterRoleBindingToRoleBindingInfo(crb *rbacv1.ClusterRoleBinding) RoleBindingInfo {
	info := RoleBindingInfo{
		Resource: Resource{
			UID:         string(crb.UID),
			APIVersion:  "rbac.authorization.k8s.io/v1",
			Kind:        "ClusterRoleBinding",
			Name:        crb.Name,
			ClusterID:   c.clusterID,
			Labels:      crb.Labels,
			Annotations: crb.Annotations,
			FetchedAt:   time.Now(),
		},
		RoleKind: crb.RoleRef.Kind,
		RoleName: crb.RoleRef.Name,
	}

	if !crb.CreationTimestamp.Time.IsZero() {
		info.Age = time.Since(crb.CreationTimestamp.Time)
	}

	var subjects []string
	for _, s := range crb.Subjects {
		switch s.Kind {
		case "ServiceAccount":
			if s.Namespace != "" {
				subjects = append(subjects, fmt.Sprintf("SA:%s/%s", s.Namespace, s.Name))
			} else {
				subjects = append(subjects, "SA:"+s.Name)
			}
		case "User":
			subjects = append(subjects, "User:"+s.Name)
		case "Group":
			subjects = append(subjects, "Group:"+s.Name)
		default:
			subjects = append(subjects, s.Kind+":"+s.Name)
		}
	}
	info.Subjects = strings.Join(subjects, ", ")

	return info
}
