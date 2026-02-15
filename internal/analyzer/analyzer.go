package analyzer

import (
	"context"
	"sync"

	"github.com/bijaya/kview/internal/k8s"
)

// Analyzer is the main analysis engine
type Analyzer interface {
	// Analyze runs all registered rules and returns diagnoses
	Analyze(ctx context.Context, resources []k8s.Resource, pods []k8s.PodInfo, events []Event) ([]Diagnosis, error)

	// AnalyzePod analyzes a single pod
	AnalyzePod(ctx context.Context, pod k8s.PodInfo, events []Event) ([]Diagnosis, error)
}

// Rule defines the interface for diagnostic rules
type Rule interface {
	Name() string
	Analyze(resources []k8s.Resource, pods []k8s.PodInfo, events []Event) []Diagnosis
}

// Engine is the default analyzer implementation
type Engine struct {
	mu    sync.RWMutex
	rules []Rule
}

// NewEngine creates a new analyzer engine
func NewEngine() *Engine {
	return &Engine{
		rules: make([]Rule, 0),
	}
}

// RegisterRule adds a rule to the engine
func (e *Engine) RegisterRule(rule Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = append(e.rules, rule)
}

// Analyze runs all registered rules
func (e *Engine) Analyze(ctx context.Context, resources []k8s.Resource, pods []k8s.PodInfo, events []Event) ([]Diagnosis, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var allDiagnoses []Diagnosis

	for _, rule := range e.rules {
		select {
		case <-ctx.Done():
			return allDiagnoses, ctx.Err()
		default:
			diagnoses := rule.Analyze(resources, pods, events)
			allDiagnoses = append(allDiagnoses, diagnoses...)
		}
	}

	return allDiagnoses, nil
}

// AnalyzePod analyzes a single pod
func (e *Engine) AnalyzePod(ctx context.Context, pod k8s.PodInfo, events []Event) ([]Diagnosis, error) {
	return e.Analyze(ctx, nil, []k8s.PodInfo{pod}, events)
}

// QuickAnalysis performs a quick analysis without events
func (e *Engine) QuickAnalysis(ctx context.Context, pods []k8s.PodInfo) ([]Diagnosis, error) {
	return e.Analyze(ctx, nil, pods, nil)
}
