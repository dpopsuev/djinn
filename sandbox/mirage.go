// mirage.go — MirageSandbox adapter: implements sandbox.Sandbox via mirage.Space.
//
// Mirage provides filesystem isolation (overlay). Exec runs commands
// inside the overlay's WorkDir via os/exec. This is the Day 1 backend —
// no daemon, no containers, just fuse-overlayfs.
package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/dpopsuev/mirage"
)

// Mirage-specific sentinel errors.
var (
	ErrNoRepos       = errors.New("mirage: at least one repo path required")
	ErrUnknownHandle = errors.New("mirage: unknown handle")
	ErrEmptyCommand  = errors.New("mirage: empty command")
)

// MirageSandbox implements Sandbox using Mirage overlay spaces.
type MirageSandbox struct {
	mu     sync.Mutex
	spaces map[Handle]mirage.Space
	nextID int
}

var _ Sandbox = (*MirageSandbox)(nil)

// NewMirageSandbox creates a Mirage-backed sandbox.
func NewMirageSandbox() *MirageSandbox {
	return &MirageSandbox{
		spaces: make(map[Handle]mirage.Space),
	}
}

func (m *MirageSandbox) Name() string { return "mirage" }

func (m *MirageSandbox) Create(_ context.Context, _ string, repos []string) (Handle, error) {
	if len(repos) == 0 {
		return "", ErrNoRepos
	}

	spec := mirage.Spec{
		Workspace: repos[0],
		Backend:   mirage.Overlay,
	}

	space, err := mirage.Create(spec)
	if err != nil {
		return "", fmt.Errorf("mirage: create space: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	handle := Handle(fmt.Sprintf("mirage-%d", m.nextID))
	m.spaces[handle] = space

	return handle, nil
}

func (m *MirageSandbox) Destroy(_ context.Context, handle Handle) error {
	m.mu.Lock()
	space, ok := m.spaces[handle]
	if ok {
		delete(m.spaces, handle)
	}
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownHandle, handle)
	}
	return space.Destroy()
}

func (m *MirageSandbox) Exec(ctx context.Context, handle Handle, cmd []string, timeout int64) (ExecResult, error) {
	m.mu.Lock()
	space, ok := m.spaces[handle]
	m.mu.Unlock()

	if !ok {
		return ExecResult{ExitCode: 1}, fmt.Errorf("%w: %q", ErrUnknownHandle, handle)
	}

	if len(cmd) == 0 {
		return ExecResult{ExitCode: 1}, ErrEmptyCommand
	}

	// Apply timeout if specified
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	// Run command inside the overlay WorkDir
	execCmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...) //nolint:gosec // sandbox-controlled command
	execCmd.Dir = space.WorkDir()

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	err := execCmd.Run()
	exitCode := extractExitCode(err)
	if err != nil && exitCode == 0 {
		// Non-exit error (e.g. binary not found)
		return ExecResult{ExitCode: 1, Stderr: err.Error()}, nil
	}

	return ExecResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

// Space returns the underlying Mirage Space for a handle.
// Used by the Arena and testkit to access Diff/Commit/Reset.
func (m *MirageSandbox) Space(handle Handle) (mirage.Space, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.spaces[handle]
	return s, ok
}

func extractExitCode(err error) int32 {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code := exitErr.ExitCode()
		if code > 0 && code < 256 {
			return int32(code) //nolint:gosec // exit codes are 0-255
		}
		return 1
	}
	return 0 // not an exit error
}

func init() {
	Register("mirage", func() (Sandbox, error) {
		return NewMirageSandbox(), nil
	})
}
