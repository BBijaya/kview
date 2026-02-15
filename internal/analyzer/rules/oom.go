package rules

import (
	"fmt"
	"time"

	"github.com/bijaya/kview/internal/analyzer"
	"github.com/bijaya/kview/internal/k8s"
)

// OOMKilledRule detects pods that were OOM killed
type OOMKilledRule struct{}

func (r *OOMKilledRule) Name() string {
	return "oom-killed"
}

func (r *OOMKilledRule) Description() string {
	return "Detects containers that were killed due to out of memory"
}

func (r *OOMKilledRule) Analyze(resources []k8s.Resource, pods []k8s.PodInfo, events []analyzer.Event) []analyzer.Diagnosis {
	var diagnoses []analyzer.Diagnosis

	for _, pod := range pods {
		for _, container := range pod.Containers {
			if container.StateReason == "OOMKilled" {
				diagnosis := analyzer.Diagnosis{
					ID:           fmt.Sprintf("oom-%s-%s", pod.UID, container.Name),
					ResourceUID:  pod.UID,
					ResourceKind: "Pod",
					ResourceName: pod.Name,
					Namespace:    pod.Namespace,
					Severity:     analyzer.SeverityCritical,
					Problem:      fmt.Sprintf("Container %s was OOM killed", container.Name),
					RootCause:    r.analyzeRootCause(pod, container, events),
					Suggestions:  r.getSuggestions(pod, container),
					DetectedAt:   time.Now(),
				}
				diagnoses = append(diagnoses, diagnosis)
			}
		}
	}

	return diagnoses
}

func (r *OOMKilledRule) analyzeRootCause(pod k8s.PodInfo, container k8s.ContainerInfo, events []analyzer.Event) string {
	rootCause := fmt.Sprintf(
		"Container '%s' in pod '%s/%s' was terminated because it exceeded its memory limit. ",
		container.Name, pod.Namespace, pod.Name,
	)

	rootCause += "This typically occurs when:\n"
	rootCause += "1. The container's memory limit is set too low for the workload\n"
	rootCause += "2. The application has a memory leak\n"
	rootCause += "3. The application is processing more data than expected\n"

	if container.RestartCount > 0 {
		rootCause += fmt.Sprintf("\nThe container has been restarted %d times, suggesting this is a recurring issue.", container.RestartCount)
	}

	return rootCause
}

func (r *OOMKilledRule) getSuggestions(pod k8s.PodInfo, container k8s.ContainerInfo) []analyzer.Suggestion {
	return []analyzer.Suggestion{
		{
			Title:       "Increase memory limit",
			Description: "Increase the container's memory limit to accommodate the workload",
			Command:     fmt.Sprintf("kubectl set resources deployment/<deployment-name> -n %s --limits=memory=<new-limit>", pod.Namespace),
			Risk:        "low",
		},
		{
			Title:       "Check application memory usage",
			Description: "Profile the application to identify memory consumption patterns",
			Command:     fmt.Sprintf("kubectl top pod %s -n %s --containers", pod.Name, pod.Namespace),
			Risk:        "low",
		},
		{
			Title:       "Check for memory leaks",
			Description: "Review application logs and metrics for memory leak indicators",
			Command:     fmt.Sprintf("kubectl logs %s -n %s -c %s --tail=100", pod.Name, pod.Namespace, container.Name),
			Risk:        "low",
		},
	}
}
