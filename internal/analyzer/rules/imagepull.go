package rules

import (
	"fmt"
	"strings"
	"time"

	"github.com/bijaya/kview/internal/analyzer"
	"github.com/bijaya/kview/internal/k8s"
)

// ImagePullRule detects image pull errors
type ImagePullRule struct{}

func (r *ImagePullRule) Name() string {
	return "image-pull-error"
}

func (r *ImagePullRule) Description() string {
	return "Detects containers with image pull errors"
}

func (r *ImagePullRule) Analyze(resources []k8s.Resource, pods []k8s.PodInfo, events []analyzer.Event) []analyzer.Diagnosis {
	var diagnoses []analyzer.Diagnosis

	for _, pod := range pods {
		for _, container := range pod.Containers {
			if r.isImagePullError(container.StateReason) {
				diagnosis := analyzer.Diagnosis{
					ID:           fmt.Sprintf("imagepull-%s-%s", pod.UID, container.Name),
					ResourceUID:  pod.UID,
					ResourceKind: "Pod",
					ResourceName: pod.Name,
					Namespace:    pod.Namespace,
					Severity:     analyzer.SeverityCritical,
					Problem:      fmt.Sprintf("Container %s cannot pull image", container.Name),
					RootCause:    r.analyzeRootCause(pod, container, events),
					Suggestions:  r.getSuggestions(pod, container),
					DetectedAt:   time.Now(),
				}
				diagnoses = append(diagnoses, diagnosis)
			}
		}

		// Also check init containers
		for _, container := range pod.InitContainers {
			if r.isImagePullError(container.StateReason) {
				diagnosis := analyzer.Diagnosis{
					ID:           fmt.Sprintf("imagepull-%s-init-%s", pod.UID, container.Name),
					ResourceUID:  pod.UID,
					ResourceKind: "Pod",
					ResourceName: pod.Name,
					Namespace:    pod.Namespace,
					Severity:     analyzer.SeverityCritical,
					Problem:      fmt.Sprintf("Init container %s cannot pull image", container.Name),
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

func (r *ImagePullRule) isImagePullError(reason string) bool {
	pullErrors := []string{
		"ImagePullBackOff",
		"ErrImagePull",
		"ImageInspectError",
		"ErrImageNeverPull",
		"RegistryUnavailable",
	}

	for _, err := range pullErrors {
		if reason == err {
			return true
		}
	}
	return false
}

func (r *ImagePullRule) analyzeRootCause(pod k8s.PodInfo, container k8s.ContainerInfo, events []analyzer.Event) string {
	var rootCause strings.Builder

	rootCause.WriteString(fmt.Sprintf(
		"Container '%s' in pod '%s/%s' failed to pull image '%s'. ",
		container.Name, pod.Namespace, pod.Name, container.Image,
	))

	rootCause.WriteString(fmt.Sprintf("Error: %s\n", container.StateReason))

	if container.StateMessage != "" {
		rootCause.WriteString(fmt.Sprintf("\nDetails: %s\n", container.StateMessage))
	}

	rootCause.WriteString("\nPossible causes:\n")
	rootCause.WriteString("1. Image name or tag is incorrect\n")
	rootCause.WriteString("2. Image doesn't exist in the registry\n")
	rootCause.WriteString("3. Registry requires authentication (missing imagePullSecret)\n")
	rootCause.WriteString("4. Network issues preventing registry access\n")
	rootCause.WriteString("5. Rate limiting by the registry\n")

	return rootCause.String()
}

func (r *ImagePullRule) getSuggestions(pod k8s.PodInfo, container k8s.ContainerInfo) []analyzer.Suggestion {
	return []analyzer.Suggestion{
		{
			Title:       "Verify image exists",
			Description: "Check if the image exists in the registry",
			Command:     fmt.Sprintf("docker pull %s", container.Image),
			Risk:        "low",
		},
		{
			Title:       "Check image pull secrets",
			Description: "Verify imagePullSecrets are configured correctly",
			Command:     fmt.Sprintf("kubectl get pod %s -n %s -o jsonpath='{.spec.imagePullSecrets}'", pod.Name, pod.Namespace),
			Risk:        "low",
		},
		{
			Title:       "List available secrets",
			Description: "Check available secrets in the namespace",
			Command:     fmt.Sprintf("kubectl get secrets -n %s", pod.Namespace),
			Risk:        "low",
		},
		{
			Title:       "Describe pod for events",
			Description: "View detailed error messages from pod events",
			Command:     fmt.Sprintf("kubectl describe pod %s -n %s", pod.Name, pod.Namespace),
			Risk:        "low",
		},
	}
}
