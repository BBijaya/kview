package rules

import (
	"fmt"
	"strings"
	"time"

	"github.com/bijaya/kview/internal/analyzer"
	"github.com/bijaya/kview/internal/k8s"
)

// ImageTagRule detects containers using :latest or untagged images
type ImageTagRule struct{}

func (r *ImageTagRule) Name() string {
	return "image-tag-best-practice"
}

func (r *ImageTagRule) Description() string {
	return "Detects containers using :latest or untagged images"
}

func (r *ImageTagRule) Analyze(resources []k8s.Resource, pods []k8s.PodInfo, events []analyzer.Event) []analyzer.Diagnosis {
	var diagnoses []analyzer.Diagnosis

	for _, pod := range pods {
		// Skip completed/succeeded pods
		if pod.Phase == "Succeeded" || pod.Phase == "Completed" {
			continue
		}

		for _, container := range pod.Containers {
			bad, reason := hasLatestOrNoTag(container.Image)
			if !bad {
				continue
			}

			problem := fmt.Sprintf("Container %s uses image with %s (%s)", container.Name, reason, container.Image)

			diagnosis := analyzer.Diagnosis{
				ID:           fmt.Sprintf("imagetag-%s-%s", pod.UID, container.Name),
				ResourceUID:  pod.UID,
				ResourceKind: "Pod",
				ResourceName: pod.Name,
				Namespace:    pod.Namespace,
				Severity:     analyzer.SeverityWarning,
				Problem:      problem,
				RootCause:    r.analyzeRootCause(pod, container, reason),
				Suggestions:  r.getSuggestions(pod, container),
				DetectedAt:   time.Now(),
			}
			diagnoses = append(diagnoses, diagnosis)
		}
	}

	return diagnoses
}

// hasLatestOrNoTag checks if an image reference uses :latest or has no tag.
// It handles registry:port/path format by distinguishing port colons from tag colons.
func hasLatestOrNoTag(image string) (bool, string) {
	// Find the part after the last slash to isolate the image name+tag
	// from any registry:port prefix
	tagPart := image
	if slashIdx := strings.LastIndex(image, "/"); slashIdx >= 0 {
		tagPart = image[slashIdx:]
	}

	colonIdx := strings.LastIndex(tagPart, ":")
	if colonIdx < 0 {
		return true, "no tag"
	}

	tag := tagPart[colonIdx+1:]
	if tag == "latest" {
		return true, ":latest tag"
	}

	return false, ""
}

func (r *ImageTagRule) analyzeRootCause(pod k8s.PodInfo, container k8s.ContainerInfo, reason string) string {
	return fmt.Sprintf(
		"Container '%s' in pod '%s/%s' uses image '%s' with %s. "+
			"This is problematic because:\n\n"+
			"1. Deployments are not reproducible — the same tag can point to different images over time\n"+
			"2. Rollbacks may not work as expected if the tag was overwritten\n"+
			"3. It's impossible to audit which exact image version is running\n"+
			"4. Image pull policy defaults to 'Always' for :latest, causing unnecessary pulls\n\n"+
			"Best practice is to pin images to a specific version tag or digest.",
		container.Name, pod.Namespace, pod.Name, container.Image, reason,
	)
}

func (r *ImageTagRule) getSuggestions(pod k8s.PodInfo, container k8s.ContainerInfo) []analyzer.Suggestion {
	return []analyzer.Suggestion{
		{
			Title:       "Pin to a specific image tag",
			Description: "Update the image reference to use a specific version tag (e.g., nginx:1.25.3) or SHA256 digest",
			Risk:        "medium",
		},
		{
			Title:       "Check current image details",
			Description: "Inspect the running container to see the resolved image ID",
			Command:     fmt.Sprintf("kubectl describe pod %s -n %s", pod.Name, pod.Namespace),
			Risk:        "low",
		},
	}
}
