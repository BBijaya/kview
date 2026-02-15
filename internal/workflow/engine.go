package workflow

import (
	"context"
	"fmt"
	"time"
)

// Engine executes workflows
type Engine struct {
	registry *ActionRegistry
	library  *WorkflowLibrary
}

// NewEngine creates a new workflow engine
func NewEngine() *Engine {
	return &Engine{
		registry: NewActionRegistry(),
		library:  NewWorkflowLibrary(),
	}
}

// LoadWorkflow loads a workflow from a file
func (e *Engine) LoadWorkflow(path string) error {
	parser := NewParser()
	workflow, err := parser.ParseFile(path)
	if err != nil {
		return err
	}
	e.library.Add(workflow)
	return nil
}

// LoadWorkflowsFromDirectory loads all workflows from a directory
func (e *Engine) LoadWorkflowsFromDirectory(dir string) error {
	parser := NewParser()
	workflows, err := parser.ParseDirectory(dir)
	for _, w := range workflows {
		e.library.Add(w)
	}
	return err
}

// ListWorkflows returns all loaded workflow names
func (e *Engine) ListWorkflows() []string {
	return e.library.List()
}

// GetWorkflow returns a workflow by name
func (e *Engine) GetWorkflow(name string) (*Workflow, bool) {
	return e.library.Get(name)
}

// Execute executes a workflow
func (e *Engine) Execute(ctx context.Context, workflow *Workflow, variables map[string]string, opts ExecuteOptions) (*ExecutionResult, error) {
	result := &ExecutionResult{
		WorkflowName: workflow.Name,
		Status:       StatusRunning,
		Steps:        make([]StepResult, 0, len(workflow.Steps)),
		StartTime:    time.Now(),
		Variables:    make(map[string]string),
	}

	// Merge variables
	for k, v := range workflow.Variables {
		result.Variables[k] = v
	}
	for k, v := range variables {
		result.Variables[k] = v
	}

	// Create action context
	actx := &ActionContext{
		Namespace: result.Variables["namespace"],
		Variables: result.Variables,
		DryRun:    opts.DryRun,
	}

	// Execute each step
	for i, step := range workflow.Steps {
		select {
		case <-ctx.Done():
			result.Status = StatusCancelled
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, ctx.Err()
		default:
		}

		stepResult := e.executeStep(ctx, step, actx, opts)
		result.Steps = append(result.Steps, stepResult)

		// Handle step result
		if stepResult.Status == StatusFailed {
			continueOn := step.ContinueOn
			if continueOn == "" {
				continueOn = "success" // Default: stop on error
			}

			if continueOn != "error" && continueOn != "always" {
				result.Status = StatusFailed
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				return result, stepResult.Error
			}
		}

		// Notify progress
		if opts.OnProgress != nil {
			opts.OnProgress(i+1, len(workflow.Steps), stepResult)
		}
	}

	result.Status = StatusCompleted
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// ExecuteOptions provides options for workflow execution
type ExecuteOptions struct {
	DryRun      bool
	Confirm     func(step Step) bool
	OnProgress  func(current, total int, result StepResult)
}

func (e *Engine) executeStep(ctx context.Context, step Step, actx *ActionContext, opts ExecuteOptions) StepResult {
	result := StepResult{
		StepName:  step.Name,
		Status:    StatusRunning,
		StartTime: time.Now(),
	}

	// Skip steps with conditions (condition evaluation not yet implemented)
	if step.Condition != "" {
		result.Status = StatusSkipped
		result.Output = fmt.Sprintf("Skipped: condition evaluation not implemented (condition: %s)", step.Condition)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// Require confirmation if needed
	if step.Confirm && opts.Confirm != nil {
		if !opts.Confirm(step) {
			result.Status = StatusSkipped
			result.Output = "Skipped by user"
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}
	}

	// Get action
	action, ok := e.registry.Get(step.Action)
	if !ok {
		result.Status = StatusFailed
		result.Error = fmt.Errorf("unknown action: %s", step.Action)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// Set timeout
	stepCtx := ctx
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		stepCtx, cancel = context.WithTimeout(ctx, step.Timeout)
		defer cancel()
	}

	// Execute with retries
	var output string
	var err error
	attempts := step.Retries + 1
	if attempts < 1 {
		attempts = 1
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		output, err = action.Execute(stepCtx, step.Args, actx)
		if err == nil {
			break
		}

		// Wait before retry
		if attempt < attempts {
			select {
			case <-stepCtx.Done():
				break
			case <-time.After(time.Second * time.Duration(attempt)):
				// Exponential backoff
			}
		}
	}

	result.Output = output
	if err != nil {
		result.Status = StatusFailed
		result.Error = err
	} else {
		result.Status = StatusCompleted
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// ExecuteByName executes a workflow by name
func (e *Engine) ExecuteByName(ctx context.Context, name string, variables map[string]string, opts ExecuteOptions) (*ExecutionResult, error) {
	workflow, ok := e.library.Get(name)
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", name)
	}
	return e.Execute(ctx, workflow, variables, opts)
}

// ValidateWorkflow validates a workflow without executing it
func (e *Engine) ValidateWorkflow(workflow *Workflow) []error {
	var errors []error

	for i, step := range workflow.Steps {
		// Check action exists
		if _, ok := e.registry.Get(step.Action); !ok {
			errors = append(errors, fmt.Errorf("step %d (%s): unknown action '%s'", i+1, step.Name, step.Action))
		}

		// Validate timeout format
		if step.Timeout != 0 && step.Timeout < 0 {
			errors = append(errors, fmt.Errorf("step %d (%s): invalid timeout", i+1, step.Name))
		}
	}

	return errors
}

// RegisterAction registers a custom action
func (e *Engine) RegisterAction(action Action) {
	e.registry.Register(action)
}
