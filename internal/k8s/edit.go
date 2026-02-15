package k8s

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

const editHeader = `# Please edit the object below. Lines beginning with a '#' will be ignored,
# and an empty file will abort the edit. If an error occurs while saving this file will be
# reopened with the relevant failures.
#
`

// EditExecCmd implements tea.ExecCommand for editing a Kubernetes resource
// in the user's preferred editor. It fetches the resource YAML, opens it
// in an editor, and applies changes on save via the dynamic client Update API.
type EditExecCmd struct {
	client    *K8sClient
	kind      string
	namespace string
	name      string
	applied   bool
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
}

// NewEditExecCmd creates a new EditExecCmd for the given resource.
func NewEditExecCmd(client *K8sClient, kind, namespace, name string) *EditExecCmd {
	return &EditExecCmd{
		client:    client,
		kind:      kind,
		namespace: namespace,
		name:      name,
	}
}

// Applied returns true if changes were applied to the cluster.
func (c *EditExecCmd) Applied() bool {
	return c.applied
}

func (c *EditExecCmd) SetStdin(r io.Reader)  { c.stdin = r }
func (c *EditExecCmd) SetStdout(w io.Writer) { c.stdout = os.Stdout }
func (c *EditExecCmd) SetStderr(w io.Writer) { c.stderr = os.Stderr }

// Run fetches the resource YAML, opens it in the editor, and applies changes.
func (c *EditExecCmd) Run() error {
	editor, err := resolveEditor()
	if err != nil {
		return err
	}

	gvr, err := c.client.getGVR(c.kind)
	if err != nil {
		return fmt.Errorf("unknown resource type %q: %w", c.kind, err)
	}

	clusterScoped := c.client.isClusterScoped(c.kind)

	// Fetch resource
	ctx := context.Background()
	var original *unstructured.Unstructured
	if clusterScoped || c.namespace == "" {
		original, err = c.client.dynamicClient.Resource(gvr).Get(ctx, c.name, metav1.GetOptions{})
	} else {
		original, err = c.client.dynamicClient.Resource(gvr).Namespace(c.namespace).Get(ctx, c.name, metav1.GetOptions{})
	}
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	// Strip managedFields for cleaner editing (kubectl edit does this too)
	obj := original.DeepCopy()
	unstructured.RemoveNestedField(obj.Object, "metadata", "managedFields")

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(obj.Object)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write to temp file with instructional header
	tmpFile, err := os.CreateTemp("", "kview-edit-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(editHeader); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if _, err := tmpFile.Write(yamlBytes); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Edit loop: reopen on save error so the user can fix mistakes
	for {
		editorCmd := exec.Command(editor, tmpPath)
		editorCmd.Stdin = c.stdin
		editorCmd.Stdout = c.stdout
		editorCmd.Stderr = c.stderr

		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		// Read modified file and strip comment lines
		modifiedBytes, err := readAndStripComments(tmpPath)
		if err != nil {
			return fmt.Errorf("failed to read modified file: %w", err)
		}

		// Empty file aborts the edit
		if len(bytes.TrimSpace(modifiedBytes)) == 0 {
			c.applied = false
			return nil
		}

		// Unmarshal modified YAML
		var modifiedObj map[string]interface{}
		if err := yaml.Unmarshal(modifiedBytes, &modifiedObj); err != nil {
			return fmt.Errorf("failed to parse modified YAML: %w", err)
		}

		// Compare: strip managedFields from original for fair comparison
		originalForCompare := original.DeepCopy()
		unstructured.RemoveNestedField(originalForCompare.Object, "metadata", "managedFields")
		if reflect.DeepEqual(originalForCompare.Object, modifiedObj) {
			c.applied = false
			return nil
		}

		// Apply changes
		modified := &unstructured.Unstructured{Object: modifiedObj}
		if clusterScoped || c.namespace == "" {
			_, err = c.client.dynamicClient.Resource(gvr).Update(ctx, modified, metav1.UpdateOptions{})
		} else {
			_, err = c.client.dynamicClient.Resource(gvr).Namespace(c.namespace).Update(ctx, modified, metav1.UpdateOptions{})
		}
		if err != nil {
			// Rewrite file with error prepended so the user can fix and retry
			errComment := fmt.Sprintf("# Error saving resource: %s\n#\n", err.Error())
			contentWithErr := errComment + string(modifiedBytes)
			if writeErr := os.WriteFile(tmpPath, []byte(contentWithErr), 0600); writeErr != nil {
				return fmt.Errorf("failed to update resource: %w", err)
			}
			continue // reopen editor
		}

		c.applied = true
		return nil
	}
}

// readAndStripComments reads a file and returns its contents with comment lines removed.
func readAndStripComments(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var buf bytes.Buffer
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// resolveEditor determines the editor to use, checking environment variables
// and falling back to vi.
func resolveEditor() (string, error) {
	for _, env := range []string{"KUBE_EDITOR", "EDITOR", "VISUAL"} {
		if v := os.Getenv(env); v != "" {
			return v, nil
		}
	}
	if path, err := exec.LookPath("vi"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("no editor found: set $EDITOR or $KUBE_EDITOR")
}
