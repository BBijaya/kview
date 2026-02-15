package analyzer

import (
	"time"
)

// Severity represents the severity of a diagnosis
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Diagnosis represents a detected problem with root cause analysis
type Diagnosis struct {
	ID               string
	ResourceUID      string
	ResourceKind     string
	ResourceName     string
	Namespace        string
	ClusterID        string
	Problem          string       // Short description
	RootCause        string       // Detailed explanation
	Severity         Severity
	Suggestions      []Suggestion
	RelatedEvents    []Event
	RelatedResources []string // UIDs
	DetectedAt       time.Time
}

// Suggestion represents a suggested fix for a problem
type Suggestion struct {
	Title       string
	Description string
	Command     string // Optional kubectl command
	Action      string // Action ID for workflow
	Risk        string // low, medium, high
}

// Event represents a Kubernetes event
type Event struct {
	ID           string
	ClusterID    string
	Namespace    string
	ResourceUID  string
	ResourceKind string
	ResourceName string
	Type         string // Normal, Warning
	Reason       string
	Message      string
	Count        int
	FirstSeen    time.Time
	LastSeen     time.Time
}

// DiagnosisSummary provides a summary of diagnoses
type DiagnosisSummary struct {
	Total    int
	Critical int
	Warning  int
	Info     int
}

// Summarize returns a summary of a list of diagnoses
func Summarize(diagnoses []Diagnosis) DiagnosisSummary {
	summary := DiagnosisSummary{Total: len(diagnoses)}
	for _, d := range diagnoses {
		switch d.Severity {
		case SeverityCritical:
			summary.Critical++
		case SeverityWarning:
			summary.Warning++
		case SeverityInfo:
			summary.Info++
		}
	}
	return summary
}

// FilterBySeverity filters diagnoses by severity
func FilterBySeverity(diagnoses []Diagnosis, severity Severity) []Diagnosis {
	var result []Diagnosis
	for _, d := range diagnoses {
		if d.Severity == severity {
			result = append(result, d)
		}
	}
	return result
}

// FilterByNamespace filters diagnoses by namespace
func FilterByNamespace(diagnoses []Diagnosis, namespace string) []Diagnosis {
	var result []Diagnosis
	for _, d := range diagnoses {
		if d.Namespace == namespace {
			result = append(result, d)
		}
	}
	return result
}

// FilterByResource filters diagnoses for a specific resource
func FilterByResource(diagnoses []Diagnosis, uid string) []Diagnosis {
	var result []Diagnosis
	for _, d := range diagnoses {
		if d.ResourceUID == uid {
			result = append(result, d)
		}
	}
	return result
}
