package workflow

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// Action represents an executable action
type Action interface {
	Name() string
	Execute(ctx context.Context, args map[string]string, actx *ActionContext) (string, error)
}

// ActionRegistry holds all registered actions
type ActionRegistry struct {
	actions map[string]Action
}

// NewActionRegistry creates a new action registry with default actions
func NewActionRegistry() *ActionRegistry {
	r := &ActionRegistry{
		actions: make(map[string]Action),
	}
	r.RegisterDefaults()
	return r
}

// RegisterDefaults registers the default set of actions
func (r *ActionRegistry) RegisterDefaults() {
	r.Register(&KubectlAction{})
	r.Register(&ShellAction{})
	r.Register(&WaitAction{})
	r.Register(&LogAction{})
	r.Register(&ScaleAction{})
	r.Register(&RestartAction{})
	r.Register(&DeleteAction{})
}

// Register adds an action to the registry
func (r *ActionRegistry) Register(action Action) {
	r.actions[action.Name()] = action
}

// Get returns an action by name
func (r *ActionRegistry) Get(name string) (Action, bool) {
	action, ok := r.actions[name]
	return action, ok
}

// KubectlAction executes kubectl commands
type KubectlAction struct{}

func (a *KubectlAction) Name() string { return "kubectl" }

func (a *KubectlAction) Execute(ctx context.Context, args map[string]string, actx *ActionContext) (string, error) {
	command := args["command"]
	if command == "" {
		return "", fmt.Errorf("kubectl: command is required")
	}

	// Expand variables
	command = expandVariables(command, actx.Variables)

	// Build full command
	cmdArgs := []string{}
	if actx.DryRun {
		cmdArgs = append(cmdArgs, "--dry-run=client")
	}
	cmdArgs = append(cmdArgs, strings.Fields(command)...)

	cmd := exec.CommandContext(ctx, "kubectl", cmdArgs...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ShellAction executes shell commands
type ShellAction struct{}

func (a *ShellAction) Name() string { return "shell" }

func (a *ShellAction) Execute(ctx context.Context, args map[string]string, actx *ActionContext) (string, error) {
	command := args["command"]
	if command == "" {
		return "", fmt.Errorf("shell: command is required")
	}

	command = expandVariables(command, actx.Variables)

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// WaitAction waits for a specified duration
type WaitAction struct{}

func (a *WaitAction) Name() string { return "wait" }

func (a *WaitAction) Execute(ctx context.Context, args map[string]string, actx *ActionContext) (string, error) {
	durationStr := args["duration"]
	if durationStr == "" {
		durationStr = "5s"
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return "", fmt.Errorf("wait: invalid duration: %w", err)
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(duration):
		return fmt.Sprintf("Waited for %s", duration), nil
	}
}

// LogAction logs a message
type LogAction struct{}

func (a *LogAction) Name() string { return "log" }

func (a *LogAction) Execute(ctx context.Context, args map[string]string, actx *ActionContext) (string, error) {
	message := args["message"]
	if message == "" {
		return "", nil
	}

	message = expandVariables(message, actx.Variables)
	return message, nil
}

// ScaleAction scales a deployment
type ScaleAction struct{}

func (a *ScaleAction) Name() string { return "scale" }

func (a *ScaleAction) Execute(ctx context.Context, args map[string]string, actx *ActionContext) (string, error) {
	resource := args["resource"]
	name := args["name"]
	namespace := args["namespace"]
	replicas := args["replicas"]

	if resource == "" {
		resource = "deployment"
	}
	if name == "" {
		name = actx.Name
	}
	if namespace == "" {
		namespace = actx.Namespace
	}

	name = expandVariables(name, actx.Variables)
	namespace = expandVariables(namespace, actx.Variables)
	replicas = expandVariables(replicas, actx.Variables)

	cmdArgs := []string{"scale", resource + "/" + name, "--replicas=" + replicas, "-n", namespace}
	if actx.DryRun {
		cmdArgs = append(cmdArgs, "--dry-run=client")
	}

	cmd := exec.CommandContext(ctx, "kubectl", cmdArgs...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// RestartAction restarts a deployment
type RestartAction struct{}

func (a *RestartAction) Name() string { return "restart" }

func (a *RestartAction) Execute(ctx context.Context, args map[string]string, actx *ActionContext) (string, error) {
	resource := args["resource"]
	name := args["name"]
	namespace := args["namespace"]

	if resource == "" {
		resource = "deployment"
	}
	if name == "" {
		name = actx.Name
	}
	if namespace == "" {
		namespace = actx.Namespace
	}

	name = expandVariables(name, actx.Variables)
	namespace = expandVariables(namespace, actx.Variables)

	cmdArgs := []string{"rollout", "restart", resource + "/" + name, "-n", namespace}
	if actx.DryRun {
		cmdArgs = append(cmdArgs, "--dry-run=client")
	}

	cmd := exec.CommandContext(ctx, "kubectl", cmdArgs...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// DeleteAction deletes a resource
type DeleteAction struct{}

func (a *DeleteAction) Name() string { return "delete" }

func (a *DeleteAction) Execute(ctx context.Context, args map[string]string, actx *ActionContext) (string, error) {
	resource := args["resource"]
	name := args["name"]
	namespace := args["namespace"]

	if resource == "" || name == "" {
		return "", fmt.Errorf("delete: resource and name are required")
	}
	if namespace == "" {
		namespace = actx.Namespace
	}

	name = expandVariables(name, actx.Variables)
	namespace = expandVariables(namespace, actx.Variables)

	cmdArgs := []string{"delete", resource, name, "-n", namespace}
	if actx.DryRun {
		cmdArgs = append(cmdArgs, "--dry-run=client")
	}

	cmd := exec.CommandContext(ctx, "kubectl", cmdArgs...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// expandVariables expands variables in a string.
// Keys are sorted longest-first to prevent partial replacement
// (e.g., "name" being replaced inside "namespace").
func expandVariables(s string, vars map[string]string) string {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	for _, k := range keys {
		v := vars[k]
		s = strings.ReplaceAll(s, "{{ ."+k+" }}", v)
		s = strings.ReplaceAll(s, "{{."+k+"}}", v)
	}
	return s
}
