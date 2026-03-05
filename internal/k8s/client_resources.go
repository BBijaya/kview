package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListPods returns detailed pod information
func (c *K8sClient) ListPods(ctx context.Context, namespace string) ([]PodInfo, error) {
	var pods *corev1.PodList
	var err error

	if namespace == "" {
		pods, err = c.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	} else {
		pods, err = c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []PodInfo
	for _, pod := range pods.Items {
		result = append(result, c.podToPodInfo(&pod))
	}
	return result, nil
}

// ListDeployments returns detailed deployment information
func (c *K8sClient) ListDeployments(ctx context.Context, namespace string) ([]DeploymentInfo, error) {
	var deployments *appsv1.DeploymentList
	var err error

	if namespace == "" {
		deployments, err = c.clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	} else {
		deployments, err = c.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []DeploymentInfo
	for _, dep := range deployments.Items {
		result = append(result, c.deploymentToDeploymentInfo(&dep))
	}
	return result, nil
}

// ListServices returns detailed service information
func (c *K8sClient) ListServices(ctx context.Context, namespace string) ([]ServiceInfo, error) {
	var services *corev1.ServiceList
	var err error

	if namespace == "" {
		services, err = c.clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	} else {
		services, err = c.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []ServiceInfo
	for _, svc := range services.Items {
		result = append(result, c.serviceToServiceInfo(&svc))
	}
	return result, nil
}

// ListEndpoints returns detailed endpoint information
func (c *K8sClient) ListEndpoints(ctx context.Context, namespace string) ([]EndpointInfo, error) {
	var endpoints *corev1.EndpointsList
	var err error

	if namespace == "" {
		endpoints, err = c.clientset.CoreV1().Endpoints("").List(ctx, metav1.ListOptions{})
	} else {
		endpoints, err = c.clientset.CoreV1().Endpoints(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []EndpointInfo
	for _, ep := range endpoints.Items {
		result = append(result, c.endpointToEndpointInfo(&ep))
	}
	return result, nil
}

// ListEndpointSlices returns detailed endpoint slice information
func (c *K8sClient) ListEndpointSlices(ctx context.Context, namespace string) ([]EndpointSliceInfo, error) {
	var list *discoveryv1.EndpointSliceList
	var err error

	if namespace == "" {
		list, err = c.clientset.DiscoveryV1().EndpointSlices("").List(ctx, metav1.ListOptions{})
	} else {
		list, err = c.clientset.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []EndpointSliceInfo
	for _, es := range list.Items {
		result = append(result, c.endpointSliceToEndpointSliceInfo(&es))
	}
	return result, nil
}

// ListConfigMaps returns detailed configmap information
func (c *K8sClient) ListConfigMaps(ctx context.Context, namespace string) ([]ConfigMapInfo, error) {
	var configmaps *corev1.ConfigMapList
	var err error

	if namespace == "" {
		configmaps, err = c.clientset.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{})
	} else {
		configmaps, err = c.clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []ConfigMapInfo
	for _, cm := range configmaps.Items {
		result = append(result, c.configMapToConfigMapInfo(&cm))
	}
	return result, nil
}

// ListSecrets returns detailed secret information
func (c *K8sClient) ListSecrets(ctx context.Context, namespace string) ([]SecretInfo, error) {
	var secrets *corev1.SecretList
	var err error

	if namespace == "" {
		secrets, err = c.clientset.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
	} else {
		secrets, err = c.clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []SecretInfo
	for _, secret := range secrets.Items {
		result = append(result, c.secretToSecretInfo(&secret))
	}
	return result, nil
}

// GetSecretDecoded fetches a Secret and returns its data fields base64-decoded.
func (c *K8sClient) GetSecretDecoded(ctx context.Context, namespace, name string) (string, error) {
	secret, err := c.clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Secret: %s/%s\n", namespace, name))
	b.WriteString(fmt.Sprintf("Type:   %s\n", secret.Type))
	b.WriteString(fmt.Sprintf("Keys:   %d\n", len(secret.Data)))
	b.WriteString("\n")

	if len(secret.Data) == 0 {
		b.WriteString("(no data)\n")
		return b.String(), nil
	}

	for key, val := range secret.Data {
		b.WriteString(fmt.Sprintf("--- %s ---\n", key))
		b.WriteString(string(val))
		if len(val) > 0 && val[len(val)-1] != '\n' {
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String(), nil
}

// ListIngresses returns detailed ingress information
func (c *K8sClient) ListIngresses(ctx context.Context, namespace string) ([]IngressInfo, error) {
	var ingresses *networkingv1.IngressList
	var err error

	if namespace == "" {
		ingresses, err = c.clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	} else {
		ingresses, err = c.clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []IngressInfo
	for _, ing := range ingresses.Items {
		result = append(result, c.ingressToIngressInfo(&ing))
	}
	return result, nil
}

// ListPVCs returns detailed PVC information
func (c *K8sClient) ListPVCs(ctx context.Context, namespace string) ([]PVCInfo, error) {
	var pvcs *corev1.PersistentVolumeClaimList
	var err error

	if namespace == "" {
		pvcs, err = c.clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	} else {
		pvcs, err = c.clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []PVCInfo
	for _, pvc := range pvcs.Items {
		result = append(result, c.pvcToPVCInfo(&pvc))
	}
	return result, nil
}

// ListStatefulSets returns detailed statefulset information
func (c *K8sClient) ListStatefulSets(ctx context.Context, namespace string) ([]StatefulSetInfo, error) {
	var statefulsets *appsv1.StatefulSetList
	var err error

	if namespace == "" {
		statefulsets, err = c.clientset.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	} else {
		statefulsets, err = c.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []StatefulSetInfo
	for _, sts := range statefulsets.Items {
		result = append(result, c.statefulSetToStatefulSetInfo(&sts))
	}
	return result, nil
}

// ListNodes returns detailed node information (cluster-scoped)
func (c *K8sClient) ListNodes(ctx context.Context) ([]NodeInfo, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Count pods per node
	podCounts := make(map[string]int)
	pods, err := c.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pod := range pods.Items {
			if pod.Spec.NodeName != "" {
				podCounts[pod.Spec.NodeName]++
			}
		}
	}

	var result []NodeInfo
	for _, node := range nodes.Items {
		info := c.nodeToNodeInfo(&node)
		info.PodCount = podCounts[node.Name]
		result = append(result, info)
	}
	return result, nil
}

// ListEvents returns detailed event information
func (c *K8sClient) ListEvents(ctx context.Context, namespace string) ([]EventInfo, error) {
	var events *corev1.EventList
	var err error

	if namespace == "" {
		events, err = c.clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	} else {
		events, err = c.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []EventInfo
	for _, event := range events.Items {
		result = append(result, c.eventToEventInfo(&event))
	}

	// Sort by LastSeen descending (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastSeen.After(result[j].LastSeen)
	})

	return result, nil
}

// ListReplicaSets returns detailed replicaset information
func (c *K8sClient) ListReplicaSets(ctx context.Context, namespace string) ([]ReplicaSetInfo, error) {
	var replicasets *appsv1.ReplicaSetList
	var err error

	if namespace == "" {
		replicasets, err = c.clientset.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{})
	} else {
		replicasets, err = c.clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []ReplicaSetInfo
	for _, rs := range replicasets.Items {
		result = append(result, c.replicaSetToReplicaSetInfo(&rs))
	}
	return result, nil
}

// ListDaemonSets returns detailed daemonset information
func (c *K8sClient) ListDaemonSets(ctx context.Context, namespace string) ([]DaemonSetInfo, error) {
	var daemonsets *appsv1.DaemonSetList
	var err error

	if namespace == "" {
		daemonsets, err = c.clientset.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	} else {
		daemonsets, err = c.clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []DaemonSetInfo
	for _, ds := range daemonsets.Items {
		result = append(result, c.daemonSetToDaemonSetInfo(&ds))
	}
	return result, nil
}

// ListJobs returns detailed job information
func (c *K8sClient) ListJobs(ctx context.Context, namespace string) ([]JobInfo, error) {
	var jobs *batchv1.JobList
	var err error

	if namespace == "" {
		jobs, err = c.clientset.BatchV1().Jobs("").List(ctx, metav1.ListOptions{})
	} else {
		jobs, err = c.clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []JobInfo
	for _, job := range jobs.Items {
		result = append(result, c.jobToJobInfo(&job))
	}
	return result, nil
}

// ListCronJobs returns detailed cronjob information
func (c *K8sClient) ListCronJobs(ctx context.Context, namespace string) ([]CronJobInfo, error) {
	var cronjobs *batchv1.CronJobList
	var err error

	if namespace == "" {
		cronjobs, err = c.clientset.BatchV1().CronJobs("").List(ctx, metav1.ListOptions{})
	} else {
		cronjobs, err = c.clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []CronJobInfo
	for _, cj := range cronjobs.Items {
		result = append(result, c.cronJobToCronJobInfo(&cj))
	}
	return result, nil
}

// GetNamespaces returns a list of namespace names
func (c *K8sClient) GetNamespaces(ctx context.Context) ([]string, error) {
	namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []string
	for _, ns := range namespaces.Items {
		result = append(result, ns.Name)
	}
	return result, nil
}

// ListNamespaceInfos returns namespace details including status and age
func (c *K8sClient) ListNamespaceInfos(ctx context.Context) ([]NamespaceInfo, error) {
	namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []NamespaceInfo
	for _, ns := range namespaces.Items {
		info := NamespaceInfo{
			Name:   ns.Name,
			Status: string(ns.Status.Phase),
		}
		if !ns.CreationTimestamp.Time.IsZero() {
			info.Age = time.Since(ns.CreationTimestamp.Time)
		}
		result = append(result, info)
	}
	return result, nil
}

// ListHPAs returns detailed HPA information
func (c *K8sClient) ListHPAs(ctx context.Context, namespace string) ([]HPAInfo, error) {
	var hpas *autoscalingv2.HorizontalPodAutoscalerList
	var err error

	if namespace == "" {
		hpas, err = c.clientset.AutoscalingV2().HorizontalPodAutoscalers("").List(ctx, metav1.ListOptions{})
	} else {
		hpas, err = c.clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []HPAInfo
	for _, hpa := range hpas.Items {
		result = append(result, c.hpaToHPAInfo(&hpa))
	}
	return result, nil
}

// ListPVs returns detailed PersistentVolume information (cluster-scoped)
func (c *K8sClient) ListPVs(ctx context.Context) ([]PVInfo, error) {
	pvs, err := c.clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []PVInfo
	for _, pv := range pvs.Items {
		result = append(result, c.pvToPVInfo(&pv))
	}
	return result, nil
}

// ListRoleBindings returns detailed RoleBinding information
func (c *K8sClient) ListRoleBindings(ctx context.Context, namespace string) ([]RoleBindingInfo, error) {
	var rbs *rbacv1.RoleBindingList
	var err error

	if namespace == "" {
		rbs, err = c.clientset.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
	} else {
		rbs, err = c.clientset.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var result []RoleBindingInfo
	for _, rb := range rbs.Items {
		result = append(result, c.roleBindingToRoleBindingInfo(&rb))
	}
	return result, nil
}

// GetContexts returns a list of available context names
func (c *K8sClient) GetContexts() ([]string, error) {
	return GetAvailableContexts()
}
