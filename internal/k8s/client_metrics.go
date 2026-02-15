package k8s

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GetClusterMetrics fetches aggregated CPU and memory metrics from the metrics API
func (c *K8sClient) GetClusterMetrics(ctx context.Context) (*ClusterMetrics, error) {
	gvr := schema.GroupVersionResource{
		Group:    "metrics.k8s.io",
		Version:  "v1beta1",
		Resource: "nodes",
	}

	result, err := c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("metrics-server not available: %w", err)
	}

	var totalCPUNanos int64
	var totalMemBytes int64

	for _, item := range result.Items {
		usage, found, _ := unstructured.NestedMap(item.Object, "usage")
		if !found {
			continue
		}
		if cpuStr, ok := usage["cpu"].(string); ok {
			totalCPUNanos += parseCPU(cpuStr)
		}
		if memStr, ok := usage["memory"].(string); ok {
			totalMemBytes += parseMemory(memStr)
		}
	}

	// Get node capacity for percentages
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return &ClusterMetrics{
			CPUUsage: FormatCPU(totalCPUNanos),
			MemUsage: FormatMemory(totalMemBytes),
		}, nil
	}

	var capacityCPUNanos int64
	var capacityMemBytes int64
	for _, node := range nodes.Items {
		cpuCap := node.Status.Capacity[corev1.ResourceCPU]
		capacityCPUNanos += cpuCap.MilliValue() * 1000000 // milli to nano
		memCap := node.Status.Capacity[corev1.ResourceMemory]
		capacityMemBytes += memCap.Value()
	}

	cpuPct := ""
	memPct := ""
	if capacityCPUNanos > 0 {
		cpuPct = fmt.Sprintf("%d%%", totalCPUNanos*100/capacityCPUNanos)
	}
	if capacityMemBytes > 0 {
		memPct = fmt.Sprintf("%d%%", totalMemBytes*100/capacityMemBytes)
	}

	return &ClusterMetrics{
		CPUUsage:    cpuPct,
		CPUCapacity: FormatCPU(capacityCPUNanos),
		MemUsage:    memPct,
		MemCapacity: FormatMemory(capacityMemBytes),
	}, nil
}

// ListPodMetrics fetches per-pod CPU and memory usage from the metrics API
func (c *K8sClient) ListPodMetrics(ctx context.Context, namespace string) ([]PodMetrics, error) {
	gvr := schema.GroupVersionResource{
		Group:    "metrics.k8s.io",
		Version:  "v1beta1",
		Resource: "pods",
	}

	var result *unstructured.UnstructuredList
	var err error
	if namespace == "" {
		result, err = c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	} else {
		result, err = c.dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("metrics-server not available: %w", err)
	}

	var metrics []PodMetrics
	for _, item := range result.Items {
		pm := PodMetrics{
			Namespace: item.GetNamespace(),
			Name:      item.GetName(),
		}

		containers, found, _ := unstructured.NestedSlice(item.Object, "containers")
		if !found {
			continue
		}
		for _, c := range containers {
			cMap, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			usage, ok := cMap["usage"].(map[string]interface{})
			if !ok {
				continue
			}
			if cpuStr, ok := usage["cpu"].(string); ok {
				pm.CPUUsage += parseCPU(cpuStr)
			}
			if memStr, ok := usage["memory"].(string); ok {
				pm.MemUsage += parseMemory(memStr)
			}
		}
		metrics = append(metrics, pm)
	}

	return metrics, nil
}

// ListNodeMetrics fetches per-node CPU and memory usage from the metrics API
func (c *K8sClient) ListNodeMetrics(ctx context.Context) ([]NodeMetrics, error) {
	gvr := schema.GroupVersionResource{
		Group:    "metrics.k8s.io",
		Version:  "v1beta1",
		Resource: "nodes",
	}

	result, err := c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("metrics-server not available: %w", err)
	}

	var metrics []NodeMetrics
	for _, item := range result.Items {
		nm := NodeMetrics{
			Name: item.GetName(),
		}
		usage, found, _ := unstructured.NestedMap(item.Object, "usage")
		if !found {
			continue
		}
		if cpuStr, ok := usage["cpu"].(string); ok {
			nm.CPUUsage = parseCPU(cpuStr)
		}
		if memStr, ok := usage["memory"].(string); ok {
			nm.MemUsage = parseMemory(memStr)
		}
		metrics = append(metrics, nm)
	}

	return metrics, nil
}

// parseCPU parses Kubernetes CPU quantity string to nanocores
func parseCPU(s string) int64 {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "n") {
		s = strings.TrimSuffix(s, "n")
		val := parseInt64(s)
		return val
	}
	if strings.HasSuffix(s, "u") {
		s = strings.TrimSuffix(s, "u")
		return parseInt64(s) * 1000
	}
	if strings.HasSuffix(s, "m") {
		s = strings.TrimSuffix(s, "m")
		return parseInt64(s) * 1000000
	}
	// Whole cores
	return parseInt64(s) * 1000000000
}

// parseMemory parses Kubernetes memory quantity string to bytes
func parseMemory(s string) int64 {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "Ki") {
		return parseInt64(strings.TrimSuffix(s, "Ki")) * 1024
	}
	if strings.HasSuffix(s, "Mi") {
		return parseInt64(strings.TrimSuffix(s, "Mi")) * 1024 * 1024
	}
	if strings.HasSuffix(s, "Gi") {
		return parseInt64(strings.TrimSuffix(s, "Gi")) * 1024 * 1024 * 1024
	}
	if strings.HasSuffix(s, "Ti") {
		return parseInt64(strings.TrimSuffix(s, "Ti")) * 1024 * 1024 * 1024 * 1024
	}
	if strings.HasSuffix(s, "k") {
		return parseInt64(strings.TrimSuffix(s, "k")) * 1000
	}
	if strings.HasSuffix(s, "M") {
		return parseInt64(strings.TrimSuffix(s, "M")) * 1000000
	}
	if strings.HasSuffix(s, "G") {
		return parseInt64(strings.TrimSuffix(s, "G")) * 1000000000
	}
	return parseInt64(s)
}

func parseInt64(s string) int64 {
	s = strings.TrimSpace(s)
	result, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return result
}

// FormatCPU formats nanocores into a human-readable CPU string (e.g. "250m", "1.5")
func FormatCPU(nanos int64) string {
	millis := nanos / 1000000
	if millis >= 1000 {
		return fmt.Sprintf("%.1f", float64(millis)/1000)
	}
	return fmt.Sprintf("%dm", millis)
}

// FormatMemory formats bytes into a human-readable memory string (e.g. "128Mi", "2Gi")
func FormatMemory(bytes int64) string {
	gi := bytes / (1024 * 1024 * 1024)
	if gi > 0 {
		return fmt.Sprintf("%dGi", gi)
	}
	mi := bytes / (1024 * 1024)
	return fmt.Sprintf("%dMi", mi)
}
