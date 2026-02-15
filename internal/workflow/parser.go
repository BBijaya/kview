package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser parses workflow definitions from YAML
type Parser struct{}

// NewParser creates a new workflow parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile parses a workflow from a file
func (p *Parser) ParseFile(path string) (*Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.Parse(data)
}

// Parse parses a workflow from YAML data
func (p *Parser) Parse(data []byte) (*Workflow, error) {
	var workflow Workflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := p.validate(&workflow); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &workflow, nil
}

// ParseDirectory parses all workflow files in a directory
func (p *Parser) ParseDirectory(dir string) ([]*Workflow, error) {
	var workflows []*Workflow

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var parseErrors []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		workflow, err := p.ParseFile(path)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("%s: %v", entry.Name(), err))
			continue
		}

		workflows = append(workflows, workflow)
	}

	if len(parseErrors) > 0 {
		return workflows, fmt.Errorf("failed to parse %d file(s): %s", len(parseErrors), strings.Join(parseErrors, "; "))
	}

	return workflows, nil
}

// validate validates a workflow definition
func (p *Parser) validate(w *Workflow) error {
	if w.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	if len(w.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	for i, step := range w.Steps {
		if step.Name == "" {
			return fmt.Errorf("step %d: name is required", i+1)
		}
		if step.Action == "" {
			return fmt.Errorf("step %d (%s): action is required", i+1, step.Name)
		}
		if !p.isValidAction(step.Action) {
			return fmt.Errorf("step %d (%s): unknown action '%s'", i+1, step.Name, step.Action)
		}
	}

	return nil
}

// isValidAction checks if an action type is valid
func (p *Parser) isValidAction(action string) bool {
	validActions := map[string]bool{
		"kubectl":   true,
		"shell":     true,
		"wait":      true,
		"confirm":   true,
		"log":       true,
		"scale":     true,
		"restart":   true,
		"delete":    true,
		"apply":     true,
		"describe":  true,
		"logs":      true,
		"exec":      true,
		"rollout":   true,
		"port-forward": true,
	}
	return validActions[action]
}

// Serialize serializes a workflow to YAML
func (p *Parser) Serialize(w *Workflow) ([]byte, error) {
	return yaml.Marshal(w)
}

// SaveToFile saves a workflow to a file
func (p *Parser) SaveToFile(w *Workflow, path string) error {
	data, err := p.Serialize(w)
	if err != nil {
		return fmt.Errorf("failed to serialize: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Example returns an example workflow definition
func (p *Parser) Example() string {
	return `# Example workflow: Restart deployment
name: restart-deployment
description: Safely restart a deployment with health checks
author: kview
version: "1.0"
tags:
  - deployment
  - restart
  - rollout

variables:
  namespace: default
  deployment: ""

steps:
  - name: Verify deployment exists
    action: describe
    args:
      resource: deployment
      name: "{{ .deployment }}"
      namespace: "{{ .namespace }}"

  - name: Scale down
    action: scale
    confirm: true
    args:
      resource: deployment
      name: "{{ .deployment }}"
      namespace: "{{ .namespace }}"
      replicas: "0"

  - name: Wait for pods to terminate
    action: wait
    args:
      duration: "10s"

  - name: Scale up
    action: scale
    args:
      resource: deployment
      name: "{{ .deployment }}"
      namespace: "{{ .namespace }}"
      replicas: "1"

  - name: Wait for rollout
    action: rollout
    args:
      resource: deployment
      name: "{{ .deployment }}"
      namespace: "{{ .namespace }}"
      timeout: "300s"

  - name: Verify health
    action: kubectl
    args:
      command: "get pods -n {{ .namespace }} -l app={{ .deployment }} -o wide"
`
}
