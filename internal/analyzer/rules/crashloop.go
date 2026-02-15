package rules

import (
	"fmt"
	"time"

	"github.com/bijaya/kview/internal/analyzer"
	"github.com/bijaya/kview/internal/k8s"
)

// CrashLoopRule detects pods in CrashLoopBackOff state
type CrashLoopRule struct{}

func (r *CrashLoopRule) Name() string {
	return "crashloop-backoff"
}

func (r *CrashLoopRule) Description() string {
	return "Detects pods stuck in CrashLoopBackOff state"
}

func (r *CrashLoopRule) Analyze(resources []k8s.Resource, pods []k8s.PodInfo, events []analyzer.Event) []analyzer.Diagnosis {
	var diagnoses []analyzer.Diagnosis

	for _, pod := range pods {
		for _, container := range pod.Containers {
			if container.StateReason == "CrashLoopBackOff" {
				diagnosis := analyzer.Diagnosis{
					ID:           fmt.Sprintf("crashloop-%s-%s", pod.UID, container.Name),
					ResourceUID:  pod.UID,
					ResourceKind: "Pod",
					ResourceName: pod.Name,
					Namespace:    pod.Namespace,
					Severity:     analyzer.SeverityCritical,
					Problem:      fmt.Sprintf("Container %s is in CrashLoopBackOff", container.Name),
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

func (r *CrashLoopRule) analyzeRootCause(pod k8s.PodInfo, container k8s.ContainerInfo, events []analyzer.Event) string {
	rootCause := fmt.Sprintf(
		"Container '%s' in pod '%s/%s' is repeatedly crashing and being restarted by Kubernetes. ",
		container.Name, pod.Namespace, pod.Name,
	)

	rootCause += fmt.Sprintf("The container has crashed %d times. ", container.RestartCount)

	rootCause += "\n\nPossible causes:\n"
	rootCause += "1. Application error or exception on startup\n"
	rootCause += "2. Missing configuration or environment variables\n"
	rootCause += "3. Failed health checks (liveness probe)\n"
	rootCause += "4. Missing dependencies or services\n"
	rootCause += "5. Insufficient resources\n"

	// Check for termination message
	if container.StateMessage != "" {
		rootCause += fmt.Sprintf("\n\nLast known error: %s", container.StateMessage)
	}

	return rootCause
}

func (r *CrashLoopRule) getSuggestions(pod k8s.PodInfo, container k8s.ContainerInfo) []analyzer.Suggestion {
	return []analyzer.Suggestion{
		{
			Title:       "Check container logs",
			Description: "Review the container logs to identify the crash cause",
			Command:     fmt.Sprintf("kubectl logs %s -n %s -c %s --previous", pod.Name, pod.Namespace, container.Name),
			Risk:        "low",
		},
		{
			Title:       "Describe pod for events",
			Description: "Check pod events for additional context",
			Command:     fmt.Sprintf("kubectl describe pod %s -n %s", pod.Name, pod.Namespace),
			Risk:        "low",
		},
		{
			Title:       "Check environment variables",
			Description: "Verify all required environment variables are set",
			Command:     fmt.Sprintf("kubectl get pod %s -n %s -o jsonpath='{.spec.containers[?(@.name==\"%s\")].env}'", pod.Name, pod.Namespace, container.Name),
			Risk:        "low",
		},
		{
			Title:       "Exec into container (if running)",
			Description: "Debug by getting a shell in the container",
			Command:     fmt.Sprintf("kubectl exec -it %s -n %s -c %s -- /bin/sh", pod.Name, pod.Namespace, container.Name),
			Risk:        "low",
		},
	}
}
