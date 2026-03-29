//go:build linux

package namespace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dpopsuev/djinn/sandbox"
	"github.com/dpopsuev/mirage"
)

// Sentinel errors.
var (
	ErrUnknownHandle = errors.New("namespace sandbox: unknown handle")
	ErrEmptyCommand  = errors.New("namespace sandbox: empty command")
)

// NamespaceSandbox implements sandbox.Sandbox using mirage overlay spaces.
// Each Create() produces an independent overlay on the workspace.
// Agent operations are captured in the overlay — real workspace untouched.
type NamespaceSandbox struct {
	workDir string
	builder mirage.Builder
	mu      sync.Mutex
	spaces  map[sandbox.Handle]mirage.Space
	counter atomic.Int64
}

// New creates a NamespaceSandbox rooted at the given workspace directory.
func New(workDir string) *NamespaceSandbox {
	return &NamespaceSandbox{
		workDir: workDir,
		builder: mirage.NewOverlayBuilder(),
		spaces:  make(map[sandbox.Handle]mirage.Space),
	}
}

// BackendName is the sandbox backend identifier for namespace sandboxes.
const BackendName = "namespace"

func (s *NamespaceSandbox) Name() string { return BackendName }

// Create mounts a new overlay on the workspace. The level and repos params
// are accepted for interface compatibility but ignored — all overlays use
// the same workspace root.
func (s *NamespaceSandbox) Create(_ context.Context, _ string, _ []string) (sandbox.Handle, error) {
	space, err := s.builder.Create(s.workDir)
	if err != nil {
		return "", fmt.Errorf("namespace sandbox: create: %w", err)
	}

	id := sandbox.Handle(fmt.Sprintf("ns-%d", s.counter.Add(1)))

	s.mu.Lock()
	s.spaces[id] = space
	s.mu.Unlock()

	return id, nil
}

// Destroy unmounts the overlay and cleans up. Real workspace untouched.
func (s *NamespaceSandbox) Destroy(_ context.Context, handle sandbox.Handle) error {
	s.mu.Lock()
	space, ok := s.spaces[handle]
	if ok {
		delete(s.spaces, handle)
	}
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownHandle, handle)
	}
	return space.Destroy()
}

// Exec runs a command inside the overlay's merged view. The command sees
// the overlay filesystem — reads pass through, writes are captured.
func (s *NamespaceSandbox) Exec(ctx context.Context, handle sandbox.Handle, cmd []string, timeoutSec int64) (sandbox.ExecResult, error) {
	s.mu.Lock()
	space, ok := s.spaces[handle]
	s.mu.Unlock()

	if !ok {
		return sandbox.ExecResult{}, fmt.Errorf("%w: %s", ErrUnknownHandle, handle)
	}

	if len(cmd) == 0 {
		return sandbox.ExecResult{}, ErrEmptyCommand
	}

	if timeoutSec > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		defer cancel()
	}

	// Run the command with cwd set to the merged overlay directory.
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...) //nolint:gosec // command is agent-controlled within sandbox
	c.Dir = space.WorkDir()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()

	exitCode := int32(0)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = int32(exitErr.ExitCode()) //nolint:gosec // exit codes are 0-255, safe for int32
		} else {
			return sandbox.ExecResult{}, fmt.Errorf("namespace sandbox: exec: %w", err)
		}
	}

	return sandbox.ExecResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

// GetSpace returns the mirage Space for a handle. Used for Diff/Commit
// operations that extend beyond the base Sandbox interface.
func (s *NamespaceSandbox) GetSpace(handle sandbox.Handle) (mirage.Space, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	space, ok := s.spaces[handle]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownHandle, handle)
	}
	return space, nil
}

// Diff returns files changed by the agent in the given sandbox.
func (s *NamespaceSandbox) Diff(handle sandbox.Handle) ([]mirage.Change, error) {
	space, err := s.GetSpace(handle)
	if err != nil {
		return nil, err
	}
	return space.Diff()
}

// Commit promotes selected files from the overlay to the real workspace.
func (s *NamespaceSandbox) Commit(handle sandbox.Handle, paths []string) error {
	space, err := s.GetSpace(handle)
	if err != nil {
		return err
	}
	return space.Commit(paths)
}

func init() {
	sandbox.Register(BackendName, func() (sandbox.Sandbox, error) {
		return New("."), nil
	})
}
