package k8s

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/muesli/cancelreader"
	"golang.org/x/term"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// ShellExecCmd implements tea.ExecCommand for interactive shell sessions.
// It connects to a container via SPDY and streams stdin/stdout/stderr.
type ShellExecCmd struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
	namespace string
	pod       string
	container string
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
}

// NewShellExecCmd creates a new ShellExecCmd for the given pod/container.
func NewShellExecCmd(clientset *kubernetes.Clientset, config *rest.Config, namespace, pod, container string) *ShellExecCmd {
	return &ShellExecCmd{
		clientset: clientset,
		config:    config,
		namespace: namespace,
		pod:       pod,
		container: container,
	}
}

func (c *ShellExecCmd) SetStdin(r io.Reader) { c.stdin = r }

// SetStdout ignores the provided writer (which is the syncWriter) and uses
// raw os.Stdout instead. The syncWriter wraps every Write() in DEC 2026
// synchronized output sequences, which corrupts shell I/O when the SPDY
// transport splits escape sequences across multiple writes.
func (c *ShellExecCmd) SetStdout(w io.Writer) { c.stdout = os.Stdout }

// SetStderr uses raw os.Stderr to bypass the syncWriter for the same reason.
func (c *ShellExecCmd) SetStderr(w io.Writer) { c.stderr = os.Stderr }

// Run executes the shell session. It probes for bash, falls back to sh,
// puts the terminal in raw mode, and streams via SPDY.
func (c *ShellExecCmd) Run() error {
	shell := c.detectShell()

	// Clear screen and show banner using raw os.Stdout (before raw mode so \n works normally).
	// We write directly to os.Stdout to avoid the syncWriter's DEC 2026 wrapping.
	fmt.Fprintf(os.Stdout, "\033[2J\033[H")
	fmt.Fprintf(os.Stdout, "<<kview-Shell>> Pod: %s/%s | Container: %s\n", c.namespace, c.pod, c.container)

	// Get the file descriptor for raw mode
	fd, ok := c.getFd()
	if !ok {
		return fmt.Errorf("stdin is not a terminal")
	}

	// Put terminal in raw mode
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("failed to set terminal raw mode: %w", err)
	}
	defer term.Restore(fd, oldState)

	// Set up terminal size queue
	tsq := newTerminalSizeQueue(fd)
	defer tsq.stop()

	// Build exec request
	req := c.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(c.pod).
		Namespace(c.namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: c.container,
			Command:   []string{shell},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Wrap stdin in a cancelreader so we can kill the SPDY stdin copier
	// goroutine after the shell exits. Without this, the SPDY goroutine
	// remains blocked on os.Stdin.Read() and races with Bubble Tea's new
	// readLoop for the first keystroke — causing it to be silently consumed
	// and discarded. The cancelreader uses epoll (Linux) / select (BSD) to
	// wait on stdin, so Cancel() interrupts it without consuming any data.
	stdinReader := c.stdin
	cr, crErr := cancelreader.NewReader(c.stdin)
	if crErr == nil {
		stdinReader = cr
	}

	streamErr := exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:             stdinReader,
		Stdout:            os.Stdout,
		Stderr:            os.Stderr,
		Tty:               true,
		TerminalSizeQueue: tsq,
	})

	// Kill the leaked SPDY stdin copier goroutine before returning.
	// This must happen before Bubble Tea's RestoreTerminal() creates its
	// new readLoop, so nothing competes for os.Stdin.
	if crErr == nil {
		cr.Cancel()
		cr.Close()
	}

	// No post-shell screen clear needed. Bubble Tea's RestoreTerminal() enters
	// a fresh alt-screen buffer (ESC[?1049h + ESC[2J + ESC[H) automatically.

	// EOF is the normal SPDY stream termination signal when the shell exits.
	// context.Canceled covers external cancellation. Both are benign.
	if errors.Is(streamErr, io.EOF) || errors.Is(streamErr, context.Canceled) {
		return nil
	}
	return streamErr
}

// detectShell probes for /bin/bash, falls back to /bin/sh.
func (c *ShellExecCmd) detectShell() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := c.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(c.pod).
		Namespace(c.namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: c.container,
			Command:   []string{"/bin/bash", "-c", "exit 0"},
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		return "/bin/sh"
	}

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	if err != nil {
		return "/bin/sh"
	}
	return "/bin/bash"
}

// getFd returns the file descriptor from stdin if it's an *os.File.
func (c *ShellExecCmd) getFd() (int, bool) {
	if f, ok := c.stdin.(*os.File); ok {
		return int(f.Fd()), true
	}
	return int(os.Stdin.Fd()), true
}

// terminalSizeQueue implements remotecommand.TerminalSizeQueue.
// It listens for SIGWINCH and sends terminal size updates.
type terminalSizeQueue struct {
	fd      int
	sizeCh  chan *remotecommand.TerminalSize
	doneCh  chan struct{}
	sigCh   chan os.Signal
}

func newTerminalSizeQueue(fd int) *terminalSizeQueue {
	tsq := &terminalSizeQueue{
		fd:     fd,
		sizeCh: make(chan *remotecommand.TerminalSize, 1),
		doneCh: make(chan struct{}),
		sigCh:  make(chan os.Signal, 1),
	}

	// Send initial size
	if size := tsq.getSize(); size != nil {
		tsq.sizeCh <- size
	}

	// Watch for SIGWINCH
	signal.Notify(tsq.sigCh, syscall.SIGWINCH)
	go tsq.watch()

	return tsq
}

func (tsq *terminalSizeQueue) watch() {
	for {
		select {
		case <-tsq.doneCh:
			return
		case <-tsq.sigCh:
			if size := tsq.getSize(); size != nil {
				select {
				case tsq.sizeCh <- size:
				default:
					// Drop if channel is full (next resize will catch up)
				}
			}
		}
	}
}

func (tsq *terminalSizeQueue) getSize() *remotecommand.TerminalSize {
	w, h, err := term.GetSize(tsq.fd)
	if err != nil {
		return nil
	}
	return &remotecommand.TerminalSize{
		Width:  uint16(w),
		Height: uint16(h),
	}
}

// Next returns the next terminal size. It blocks until a size is available
// or the queue is stopped (returns nil).
func (tsq *terminalSizeQueue) Next() *remotecommand.TerminalSize {
	select {
	case size := <-tsq.sizeCh:
		return size
	case <-tsq.doneCh:
		return nil
	}
}

func (tsq *terminalSizeQueue) stop() {
	signal.Stop(tsq.sigCh)
	close(tsq.doneCh)
}
