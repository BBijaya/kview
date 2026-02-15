package rules

import (
	"github.com/bijaya/kview/internal/analyzer"
	"github.com/bijaya/kview/internal/k8s"
)

// Rule defines the interface for diagnostic rules
type Rule interface {
	// Name returns the rule name
	Name() string

	// Description returns a description of what the rule detects
	Description() string

	// Analyze analyzes resources and events to find problems
	Analyze(resources []k8s.Resource, pods []k8s.PodInfo, events []analyzer.Event) []analyzer.Diagnosis
}

// RuleSet is a collection of rules
type RuleSet struct {
	rules []Rule
}

// NewRuleSet creates a new rule set with default rules
func NewRuleSet() *RuleSet {
	rs := &RuleSet{}
	rs.RegisterDefaults()
	return rs
}

// RegisterDefaults registers the default set of rules
func (rs *RuleSet) RegisterDefaults() {
	rs.Register(&OOMKilledRule{})
	rs.Register(&CrashLoopRule{})
	rs.Register(&PendingPodRule{})
	rs.Register(&ImagePullRule{})
	rs.Register(&ResourceLimitsRule{})
	rs.Register(&ImageTagRule{})
	rs.Register(&HealthProbeRule{})
	rs.Register(&SecurityRule{})
}

// Register adds a rule to the set
func (rs *RuleSet) Register(rule Rule) {
	rs.rules = append(rs.rules, rule)
}

// Rules returns all registered rules
func (rs *RuleSet) Rules() []Rule {
	return rs.rules
}

// Analyze runs all rules and returns combined diagnoses
func (rs *RuleSet) Analyze(resources []k8s.Resource, pods []k8s.PodInfo, events []analyzer.Event) []analyzer.Diagnosis {
	var allDiagnoses []analyzer.Diagnosis
	for _, rule := range rs.rules {
		diagnoses := rule.Analyze(resources, pods, events)
		allDiagnoses = append(allDiagnoses, diagnoses...)
	}
	return allDiagnoses
}
