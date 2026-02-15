package rules

import (
	"fmt"
	"strings"
	"time"

	"github.com/bijaya/kview/internal/analyzer"
	"github.com/bijaya/kview/internal/k8s"
)

// HealthProbeRule detects containers without liveness or readiness probes
type HealthProbeRule struct{}

func (r *HealthProbeRule) Name() string {
	return "missing-health-probes"
}

func (r *HealthProbeRule) Description() string {
	return "Detects containers without liveness or readiness probes configured"
}

func (r *HealthProbeRule) Analyze(resources []k8s.Resource, pods []k8s.PodInfo, events []analyzer.Event) []analyzer.Diagnosis {
	var diagnoses []analyzer.Diagnosis

	for _, pod := range pods {
		// Skip completed/succeeded pods — probes are irrelevant
		if pod.Phase == "Succeeded" || pod.Phase == "Completed" {
			continue
		}

		for _, container := range pod.Containers {
			var missing []string
			if !container.HasLivenessProbe {
				missing = append(missing, "liveness")
			}
			if !container.HasReadinessProbe {
				missing = append(missing, "readiness")
			}

			if len(missing) == 0 {
				continue
			}

			problem := fmt.Sprintf("Container %s missing %s probe(s)", container.Name, strings.Join(missing, " and "))

			diagnosis := analyzer.Diagnosis{
				ID:           fmt.Sprintf("probes-%s-%s", pod.UID, container.Name),
				ResourceUID:  pod.UID,
				ResourceKind: "Pod",
				ResourceName: pod.Name,
				Namespace:    pod.Namespace,
				Severity:     analyzer.SeverityWarning,
				Problem:      problem,
				RootCause:    r.analyzeRootCause(pod, container, missing),
				Suggestions:  r.getSuggestions(pod, container, missing),
				DetectedAt:   time.Now(),
			}
			diagnoses = append(diagnoses, diagnosis)
		}
	}

	return diagnoses
}

func (r *HealthProbeRule) analyzeRootCause(pod k8s.PodInfo, container k8s.ContainerInfo, missing []string) string {
	rootCause := fmt.Sprintf(
		"Container '%s' in pod '%s/%s' is missing %s probe(s).\n\n",
		container.Name, pod.Namespace, pod.Name, strings.Join(missing, " and "),
	)

	for _, probe := range missing {
		switch probe {
		case "liveness":
			rootCause += "Without a liveness probe, Kubernetes cannot detect if the process is hung or deadlocked. " +
				"The container will keep running even if the application is no longer functional.\n\n"
		case "readiness":
			rootCause += "Without a readiness probe, Kubernetes will route traffic to the container as soon as it starts, " +
				"even if the application is not yet ready to serve requests. This can cause errors for end users.\n\n"
		}
	}

	rootCause += "Best practice is to configure both probes for production workloads."
	return rootCause
}

func (r *HealthProbeRule) getSuggestions(pod k8s.PodInfo, container k8s.ContainerInfo, missing []string) []analyzer.Suggestion {
	var suggestions []analyzer.Suggestion

	for _, probe := range missing {
		switch probe {
		case "liveness":
			suggestions = append(suggestions, analyzer.Suggestion{
				Title:       "Add liveness probe",
				Description: "Configure a liveness probe (HTTP, TCP, or exec) so Kubernetes can restart the container if it becomes unresponsive",
				Risk:        "medium",
			})
		case "readiness":
			suggestions = append(suggestions, analyzer.Suggestion{
				Title:       "Add readiness probe",
				Description: "Configure a readiness probe so Kubernetes only sends traffic to the container when it's ready to handle requests",
				Risk:        "medium",
			})
		}
	}

	suggestions = append(suggestions, analyzer.Suggestion{
		Title:       "Describe pod to check probe configuration",
		Description: "Review the current pod spec to see existing probe settings",
		Command:     fmt.Sprintf("kubectl describe pod %s -n %s", pod.Name, pod.Namespace),
		Risk:        "low",
	})

	return suggestions
}
