package analyzer

import (
	"strings"
)

// Suggester generates fix suggestions based on diagnoses
type Suggester struct{}

// NewSuggester creates a new suggester
func NewSuggester() *Suggester {
	return &Suggester{}
}

// EnhanceSuggestions adds context-aware suggestions to diagnoses
func (s *Suggester) EnhanceSuggestions(diagnoses []Diagnosis) []Diagnosis {
	enhanced := make([]Diagnosis, len(diagnoses))
	copy(enhanced, diagnoses)

	for i := range enhanced {
		enhanced[i].Suggestions = s.enhance(enhanced[i])
	}

	return enhanced
}

func (s *Suggester) enhance(d Diagnosis) []Suggestion {
	suggestions := make([]Suggestion, len(d.Suggestions))
	copy(suggestions, d.Suggestions)

	// Add general troubleshooting suggestions based on severity
	switch d.Severity {
	case SeverityCritical:
		suggestions = append(suggestions, Suggestion{
			Title:       "Get cluster events",
			Description: "Check cluster-wide events for related issues",
			Command:     "kubectl get events --sort-by='.lastTimestamp' -A | head -50",
			Risk:        "low",
		})
	}

	// Add resource-specific suggestions
	switch d.ResourceKind {
	case "Pod":
		suggestions = s.addPodSuggestions(d, suggestions)
	case "Deployment":
		suggestions = s.addDeploymentSuggestions(d, suggestions)
	case "Service":
		suggestions = s.addServiceSuggestions(d, suggestions)
	}

	return suggestions
}

func (s *Suggester) addPodSuggestions(d Diagnosis, suggestions []Suggestion) []Suggestion {
	// Check if network-related
	if strings.Contains(strings.ToLower(d.Problem), "network") ||
		strings.Contains(strings.ToLower(d.RootCause), "network") {
		suggestions = append(suggestions, Suggestion{
			Title:       "Check network policies",
			Description: "Review network policies affecting the pod",
			Command:     "kubectl get networkpolicy -n " + d.Namespace,
			Risk:        "low",
		})
	}

	// Check if storage-related
	if strings.Contains(strings.ToLower(d.Problem), "volume") ||
		strings.Contains(strings.ToLower(d.RootCause), "pvc") {
		suggestions = append(suggestions, Suggestion{
			Title:       "Check PVC status",
			Description: "Verify PersistentVolumeClaim status",
			Command:     "kubectl get pvc -n " + d.Namespace,
			Risk:        "low",
		})
	}

	return suggestions
}

func (s *Suggester) addDeploymentSuggestions(d Diagnosis, suggestions []Suggestion) []Suggestion {
	suggestions = append(suggestions, Suggestion{
		Title:       "Check rollout status",
		Description: "View deployment rollout status",
		Command:     "kubectl rollout status deployment/" + d.ResourceName + " -n " + d.Namespace,
		Risk:        "low",
	})

	suggestions = append(suggestions, Suggestion{
		Title:       "View rollout history",
		Description: "Check deployment revision history",
		Command:     "kubectl rollout history deployment/" + d.ResourceName + " -n " + d.Namespace,
		Risk:        "low",
	})

	return suggestions
}

func (s *Suggester) addServiceSuggestions(d Diagnosis, suggestions []Suggestion) []Suggestion {
	suggestions = append(suggestions, Suggestion{
		Title:       "Check endpoints",
		Description: "Verify service endpoints",
		Command:     "kubectl get endpoints " + d.ResourceName + " -n " + d.Namespace,
		Risk:        "low",
	})

	suggestions = append(suggestions, Suggestion{
		Title:       "Test service connectivity",
		Description: "Run a test pod to check service connectivity",
		Command:     "kubectl run test --rm -it --image=busybox --restart=Never -- wget -qO- " + d.ResourceName + "." + d.Namespace + ".svc.cluster.local",
		Risk:        "low",
	})

	return suggestions
}

// PrioritizeSuggestions orders suggestions by risk and relevance
func (s *Suggester) PrioritizeSuggestions(suggestions []Suggestion) []Suggestion {
	// Order: low risk first, then medium, then high
	prioritized := make([]Suggestion, 0, len(suggestions))

	for _, sug := range suggestions {
		if sug.Risk == "low" {
			prioritized = append(prioritized, sug)
		}
	}
	for _, sug := range suggestions {
		if sug.Risk == "medium" {
			prioritized = append(prioritized, sug)
		}
	}
	for _, sug := range suggestions {
		if sug.Risk == "high" {
			prioritized = append(prioritized, sug)
		}
	}

	return prioritized
}
