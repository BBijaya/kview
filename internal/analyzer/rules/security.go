package rules

import (
	"fmt"
	"time"

	"github.com/bijaya/kview/internal/analyzer"
	"github.com/bijaya/kview/internal/k8s"
)

// SecurityRule detects security policy violations in pods
type SecurityRule struct{}

func (r *SecurityRule) Name() string {
	return "security-policy"
}

func (r *SecurityRule) Description() string {
	return "Detects privileged containers, hostNetwork, and containers running as root"
}

func (r *SecurityRule) Analyze(resources []k8s.Resource, pods []k8s.PodInfo, events []analyzer.Event) []analyzer.Diagnosis {
	var diagnoses []analyzer.Diagnosis

	for _, pod := range pods {
		// Skip completed/succeeded pods
		if pod.Phase == "Succeeded" || pod.Phase == "Completed" {
			continue
		}

		// Check hostNetwork at pod level
		if pod.HostNetwork {
			diagnoses = append(diagnoses, analyzer.Diagnosis{
				ID:           fmt.Sprintf("security-hostnet-%s", pod.UID),
				ResourceUID:  pod.UID,
				ResourceKind: "Pod",
				ResourceName: pod.Name,
				Namespace:    pod.Namespace,
				Severity:     analyzer.SeverityWarning,
				Problem:      fmt.Sprintf("Pod %s uses hostNetwork", pod.Name),
				RootCause: fmt.Sprintf(
					"Pod '%s/%s' has hostNetwork enabled, which means it shares the host's network namespace. "+
						"This gives the pod direct access to the host's network interfaces and can:\n\n"+
						"1. Bypass network policies\n"+
						"2. Access services listening on localhost on the host\n"+
						"3. Conflict with ports used by other pods or the host\n"+
						"4. Reduce isolation between the pod and the host",
					pod.Namespace, pod.Name,
				),
				Suggestions: []analyzer.Suggestion{
					{
						Title:       "Remove hostNetwork if not required",
						Description: "Set hostNetwork: false in the pod spec unless the pod genuinely needs host-level network access (e.g., CNI plugins, ingress controllers)",
						Risk:        "medium",
					},
					{
						Title:       "Describe pod to review network configuration",
						Description: "Check the current pod spec to understand why hostNetwork is enabled",
						Command:     fmt.Sprintf("kubectl describe pod %s -n %s", pod.Name, pod.Namespace),
						Risk:        "low",
					},
				},
				DetectedAt: time.Now(),
			})
		}

		// Check per-container security issues
		for _, container := range pod.Containers {
			// Privileged container
			if container.Privileged {
				diagnoses = append(diagnoses, analyzer.Diagnosis{
					ID:           fmt.Sprintf("security-privileged-%s-%s", pod.UID, container.Name),
					ResourceUID:  pod.UID,
					ResourceKind: "Pod",
					ResourceName: pod.Name,
					Namespace:    pod.Namespace,
					Severity:     analyzer.SeverityCritical,
					Problem:      fmt.Sprintf("Container %s runs in privileged mode", container.Name),
					RootCause: fmt.Sprintf(
						"Container '%s' in pod '%s/%s' is running in privileged mode. "+
							"This grants the container almost all capabilities of the host, including:\n\n"+
							"1. Full access to all host devices\n"+
							"2. Ability to modify kernel parameters\n"+
							"3. Ability to escape container isolation\n"+
							"4. Access to all host filesystems via /dev\n\n"+
							"This is a critical security risk and should be avoided unless absolutely necessary.",
						container.Name, pod.Namespace, pod.Name,
					),
					Suggestions: []analyzer.Suggestion{
						{
							Title:       "Remove privileged flag",
							Description: "Set privileged: false in the container's securityContext",
							Risk:        "medium",
						},
						{
							Title:       "Use specific capabilities instead",
							Description: "Replace privileged mode with specific Linux capabilities (e.g., NET_ADMIN, SYS_PTRACE) that the container actually needs",
							Risk:        "medium",
						},
						{
							Title:       "Describe pod security context",
							Description: "Review the current security context configuration",
							Command:     fmt.Sprintf("kubectl get pod %s -n %s -o jsonpath='{.spec.containers[?(@.name==\"%s\")].securityContext}'", pod.Name, pod.Namespace, container.Name),
							Risk:        "low",
						},
					},
					DetectedAt: time.Now(),
				})
			}

			// Running as root (RunAsNonRoot not set or false)
			if container.RunAsNonRoot == nil || !*container.RunAsNonRoot {
				diagnoses = append(diagnoses, analyzer.Diagnosis{
					ID:           fmt.Sprintf("security-root-%s-%s", pod.UID, container.Name),
					ResourceUID:  pod.UID,
					ResourceKind: "Pod",
					ResourceName: pod.Name,
					Namespace:    pod.Namespace,
					Severity:     analyzer.SeverityInfo,
					Problem:      fmt.Sprintf("Container %s may run as root", container.Name),
					RootCause: fmt.Sprintf(
						"Container '%s' in pod '%s/%s' does not enforce non-root execution. "+
							"Without runAsNonRoot: true, the container process may run as UID 0 (root), which:\n\n"+
							"1. Increases the impact of a container breakout\n"+
							"2. Allows modification of container filesystem as root\n"+
							"3. May violate organizational security policies\n\n"+
							"While many containers work fine as root, enforcing non-root is a security best practice.",
						container.Name, pod.Namespace, pod.Name,
					),
					Suggestions: []analyzer.Suggestion{
						{
							Title:       "Set runAsNonRoot: true",
							Description: "Add runAsNonRoot: true to the container or pod securityContext to prevent running as root",
							Risk:        "medium",
						},
						{
							Title:       "Set a non-root user with runAsUser",
							Description: "Specify an explicit non-root UID (e.g., runAsUser: 1000) in the securityContext",
							Risk:        "medium",
						},
					},
					DetectedAt: time.Now(),
				})
			}
		}
	}

	return diagnoses
}
