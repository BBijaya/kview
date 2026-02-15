package rules

import (
	"fmt"
	"strings"
	"time"

	"github.com/bijaya/kview/internal/analyzer"
	"github.com/bijaya/kview/internal/k8s"
)

// PendingPodRule detects pods stuck in Pending state
type PendingPodRule struct{}

func (r *PendingPodRule) Name() string {
	return "pending-pod"
}

func (r *PendingPodRule) Description() string {
	return "Detects pods stuck in Pending state"
}

func (r *PendingPodRule) Analyze(resources []k8s.Resource, pods []k8s.PodInfo, events []analyzer.Event) []analyzer.Diagnosis {
	var diagnoses []analyzer.Diagnosis

	for _, pod := range pods {
		// Only flag pods pending for more than 30 seconds
		if pod.Phase == "Pending" && pod.Age > 30*time.Second {
			diagnosis := analyzer.Diagnosis{
				ID:           fmt.Sprintf("pending-%s", pod.UID),
				ResourceUID:  pod.UID,
				ResourceKind: "Pod",
				ResourceName: pod.Name,
				Namespace:    pod.Namespace,
				Severity:     r.determineSeverity(pod),
				Problem:      "Pod is stuck in Pending state",
				RootCause:    r.analyzeRootCause(pod, events),
				Suggestions:  r.getSuggestions(pod, events),
				DetectedAt:   time.Now(),
			}
			diagnoses = append(diagnoses, diagnosis)
		}
	}

	return diagnoses
}

func (r *PendingPodRule) determineSeverity(pod k8s.PodInfo) analyzer.Severity {
	if pod.Age > 10*time.Minute {
		return analyzer.SeverityCritical
	}
	if pod.Age > 2*time.Minute {
		return analyzer.SeverityWarning
	}
	return analyzer.SeverityInfo
}

func (r *PendingPodRule) analyzeRootCause(pod k8s.PodInfo, events []analyzer.Event) string {
	var rootCause strings.Builder

	rootCause.WriteString(fmt.Sprintf(
		"Pod '%s/%s' has been in Pending state for %s. ",
		pod.Namespace, pod.Name, formatDuration(pod.Age),
	))

	// Look for relevant events
	for _, event := range events {
		if event.ResourceUID == pod.UID && event.Type == "Warning" {
			if strings.Contains(event.Reason, "FailedScheduling") {
				rootCause.WriteString("\n\nScheduling failed: ")
				rootCause.WriteString(event.Message)
			}
		}
	}

	rootCause.WriteString("\n\nCommon causes for Pending pods:\n")
	rootCause.WriteString("1. Insufficient cluster resources (CPU, memory)\n")
	rootCause.WriteString("2. Node selectors or affinity rules cannot be satisfied\n")
	rootCause.WriteString("3. Taints on nodes without matching tolerations\n")
	rootCause.WriteString("4. PersistentVolumeClaim pending\n")
	rootCause.WriteString("5. ResourceQuota exceeded\n")

	return rootCause.String()
}

func (r *PendingPodRule) getSuggestions(pod k8s.PodInfo, events []analyzer.Event) []analyzer.Suggestion {
	suggestions := []analyzer.Suggestion{
		{
			Title:       "Describe pod",
			Description: "Check pod events and conditions for details",
			Command:     fmt.Sprintf("kubectl describe pod %s -n %s", pod.Name, pod.Namespace),
			Risk:        "low",
		},
		{
			Title:       "Check node resources",
			Description: "View available resources on cluster nodes",
			Command:     "kubectl top nodes",
			Risk:        "low",
		},
	}

	// Check for PVC issues in events
	for _, event := range events {
		if event.ResourceUID == pod.UID && strings.Contains(event.Message, "persistentvolumeclaim") {
			suggestions = append(suggestions, analyzer.Suggestion{
				Title:       "Check PVC status",
				Description: "Verify PersistentVolumeClaim is bound",
				Command:     fmt.Sprintf("kubectl get pvc -n %s", pod.Namespace),
				Risk:        "low",
			})
			break
		}
	}

	suggestions = append(suggestions, analyzer.Suggestion{
		Title:       "Check resource quotas",
		Description: "View resource quotas in the namespace",
		Command:     fmt.Sprintf("kubectl get resourcequota -n %s", pod.Namespace),
		Risk:        "low",
	})

	return suggestions
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
