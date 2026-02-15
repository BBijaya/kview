package workflow

import (
	"time"
)

// Workflow represents a saved workflow/runbook
type Workflow struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Author      string            `yaml:"author,omitempty"`
	Version     string            `yaml:"version,omitempty"`
	Tags        []string          `yaml:"tags,omitempty"`
	Variables   map[string]string `yaml:"variables,omitempty"`
	Steps       []Step            `yaml:"steps"`
}

// Step represents a single step in a workflow
type Step struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Action      string            `yaml:"action"`
	Args        map[string]string `yaml:"args,omitempty"`
	Condition   string            `yaml:"condition,omitempty"` // Condition to evaluate
	Confirm     bool              `yaml:"confirm,omitempty"`   // Require confirmation
	ContinueOn  string            `yaml:"continueOn,omitempty"` // error, success, always
	Timeout     time.Duration     `yaml:"timeout,omitempty"`
	Retries     int               `yaml:"retries,omitempty"`
}

// ExecutionStatus represents the status of a workflow or step execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusSkipped   ExecutionStatus = "skipped"
	StatusCancelled ExecutionStatus = "cancelled"
)

// StepResult represents the result of executing a step
type StepResult struct {
	StepName  string
	Status    ExecutionStatus
	Output    string
	Error     error
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
}

// ExecutionResult represents the result of executing a workflow
type ExecutionResult struct {
	WorkflowName string
	Status       ExecutionStatus
	Steps        []StepResult
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Variables    map[string]string
}

// ActionType represents the type of action
type ActionType string

const (
	ActionKubectl   ActionType = "kubectl"
	ActionShell     ActionType = "shell"
	ActionWait      ActionType = "wait"
	ActionConfirm   ActionType = "confirm"
	ActionLog       ActionType = "log"
	ActionScale     ActionType = "scale"
	ActionRestart   ActionType = "restart"
	ActionDelete    ActionType = "delete"
	ActionApply     ActionType = "apply"
	ActionDescribe  ActionType = "describe"
	ActionLogs      ActionType = "logs"
)

// ActionContext provides context for action execution
type ActionContext struct {
	Namespace  string
	Resource   string
	Name       string
	Cluster    string
	Variables  map[string]string
	DryRun     bool
}

// WorkflowLibrary stores a collection of workflows
type WorkflowLibrary struct {
	Workflows map[string]*Workflow
}

// NewWorkflowLibrary creates a new workflow library
func NewWorkflowLibrary() *WorkflowLibrary {
	return &WorkflowLibrary{
		Workflows: make(map[string]*Workflow),
	}
}

// Add adds a workflow to the library
func (l *WorkflowLibrary) Add(workflow *Workflow) {
	l.Workflows[workflow.Name] = workflow
}

// Get retrieves a workflow by name
func (l *WorkflowLibrary) Get(name string) (*Workflow, bool) {
	w, ok := l.Workflows[name]
	return w, ok
}

// List returns all workflow names
func (l *WorkflowLibrary) List() []string {
	names := make([]string, 0, len(l.Workflows))
	for name := range l.Workflows {
		names = append(names, name)
	}
	return names
}

// Remove removes a workflow from the library
func (l *WorkflowLibrary) Remove(name string) {
	delete(l.Workflows, name)
}
