package rules

import (
	"fmt"
	"time"

	"github.com/bijaya/kview/internal/analyzer"
	"github.com/bijaya/kview/internal/k8s"
)

// ResourceLimitsRule detects pods without resource limits
type ResourceLimitsRule struct{}

func (r *ResourceLimitsRule) Name() string {
	return "missing-resource-limits"
}

func (r *ResourceLimitsRule) Description() string {
	return "Detects containers without resource limits configured"
}

func (r *ResourceLimitsRule) Analyze(resources []k8s.Resource, pods []k8s.PodInfo, events []analyzer.Event) []analyzer.Diagnosis {
	var diagnoses []analyzer.Diagnosis

	// Check resources from the raw unstructured data
	for _, resource := range resources {
		if resource.Kind != "Pod" || resource.Raw == nil {
			continue
		}

		spec, ok := resource.Raw.Object["spec"].(map[string]interface{})
		if !ok {
			continue
		}

		containers, ok := spec["containers"].([]interface{})
		if !ok {
			continue
		}

		for _, c := range containers {
			container, ok := c.(map[string]interface{})
			if !ok {
				continue
			}

			containerName, _ := container["name"].(string)
			hasLimits := r.hasResourceLimits(container)

			if !hasLimits {
				diagnosis := analyzer.Diagnosis{
					ID:           fmt.Sprintf("nolimits-%s-%s", resource.UID, containerName),
					ResourceUID:  resource.UID,
					ResourceKind: "Pod",
					ResourceName: resource.Name,
					Namespace:    resource.Namespace,
					Severity:     analyzer.SeverityWarning,
					Problem:      fmt.Sprintf("Container %s has no resource limits", containerName),
					RootCause:    r.analyzeRootCause(resource, containerName),
					Suggestions:  r.getSuggestions(resource, containerName),
					DetectedAt:   time.Now(),
				}
				diagnoses = append(diagnoses, diagnosis)
			}
		}
	}

	return diagnoses
}

func (r *ResourceLimitsRule) hasResourceLimits(container map[string]interface{}) bool {
	resources, ok := container["resources"].(map[string]interface{})
	if !ok {
		return false
	}

	limits, ok := resources["limits"].(map[string]interface{})
	if !ok {
		return false
	}

	// Check for CPU or memory limits
	_, hasCPU := limits["cpu"]
	_, hasMemory := limits["memory"]

	return hasCPU || hasMemory
}

func (r *ResourceLimitsRule) analyzeRootCause(resource k8s.Resource, containerName string) string {
	return fmt.Sprintf(
		"Container '%s' in pod '%s/%s' does not have resource limits configured. "+
			"Without limits, the container can consume unbounded resources, potentially "+
			"affecting other workloads on the same node and leading to:\n\n"+
			"1. Resource starvation for other pods\n"+
			"2. Node instability\n"+
			"3. Unpredictable application behavior under load\n"+
			"4. Difficulty in capacity planning\n\n"+
			"Best practice is to set both CPU and memory limits for all containers.",
		containerName, resource.Namespace, resource.Name,
	)
}

func (r *ResourceLimitsRule) getSuggestions(resource k8s.Resource, containerName string) []analyzer.Suggestion {
	return []analyzer.Suggestion{
		{
			Title:       "Set resource limits",
			Description: "Add CPU and memory limits to the container spec",
			Command:     fmt.Sprintf("kubectl set resources deployment/<name> -n %s --limits=cpu=500m,memory=512Mi", resource.Namespace),
			Risk:        "medium",
		},
		{
			Title:       "Check current resource usage",
			Description: "View current resource consumption to determine appropriate limits",
			Command:     fmt.Sprintf("kubectl top pod %s -n %s --containers", resource.Name, resource.Namespace),
			Risk:        "low",
		},
		{
			Title:       "Create LimitRange",
			Description: "Set default limits for the namespace",
			Command:     fmt.Sprintf("kubectl apply -f - <<EOF\napiVersion: v1\nkind: LimitRange\nmetadata:\n  name: default-limits\n  namespace: %s\nspec:\n  limits:\n  - default:\n      cpu: 500m\n      memory: 512Mi\n    defaultRequest:\n      cpu: 100m\n      memory: 128Mi\n    type: Container\nEOF", resource.Namespace),
			Risk:        "low",
		},
	}
}
