// Package misbah implements broker.SandboxPort using the Misbah daemon.
// Wraps daemon.Client with Agent Space operations (Diff, Commit, Events, Logs).
package misbah

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/dpopsuev/misbah/daemon"
	"github.com/dpopsuev/misbah/model"

	"github.com/dpopsuev/djinn/tier"
)

// Default container settings.
const (
	specVersion     = "1.0"
	containerPrefix = "djinn-"
	defaultCwd      = "/workspace"
)

// DiffEntry re-exports Misbah's overlay diff entry type.
type DiffEntry = daemon.DiffEntry

// ContainerEvent re-exports Misbah's container lifecycle event type.
type ContainerEvent = daemon.ContainerEvent

// SandboxPort implements broker.SandboxPort using a Misbah daemon client.
type SandboxPort struct {
	client    *daemon.Client
	workspace string // host workspace path to mount
	counter   atomic.Int64
}

// New creates a SandboxPort connected to the Misbah daemon at the given socket.
func New(socketPath, workspace string) *SandboxPort {
	client := daemon.NewClient(socketPath, nil)
	return &SandboxPort{
		client:    client,
		workspace: workspace,
	}
}

// Create creates a Misbah container with tier-appropriate mounts.
// The daemon's tier system generates overlay mounts for writable paths —
// agent writes are captured in the overlay, real workspace untouched.
func (s *SandboxPort) Create(ctx context.Context, scope tier.Scope) (string, error) {
	name := fmt.Sprintf("%s%s-%d", containerPrefix, scope.Name, s.counter.Add(1))

	spec := &model.ContainerSpec{
		Version: specVersion,
		Metadata: model.ContainerMetadata{
			Name: name,
		},
		Process: model.ProcessSpec{
			Command: []string{"sleep", "infinity"},
			Cwd:     defaultCwd,
		},
		Namespaces: model.NamespaceSpec{
			User:  true,
			Mount: true,
		},
		Mounts: []model.MountSpec{
			{Type: "tmpfs", Destination: "/tmp"},
		},
	}

	// Mount workspace read-only as overlay base. TierConfig tells Misbah
	// which paths are writable — the daemon generates overlay mounts for those.
	spec.Mounts = append(spec.Mounts, model.MountSpec{
		Type:        "bind",
		Source:      s.workspace,
		Destination: defaultCwd,
		Options:     []string{"ro", "rbind"},
	})

	if scope.Name != "" {
		spec.TierConfig = &model.TierConfig{
			Tier:          scope.Level.String(),
			WritablePaths: []string{scope.Name},
		}
	}

	if err := spec.Validate(); err != nil {
		return "", fmt.Errorf("invalid container spec: %w", err)
	}

	resp, err := s.client.ContainerStart(ctx, spec)
	if err != nil {
		return "", fmt.Errorf("container start: %w", err)
	}

	return resp.ID, nil
}

// Destroy stops and removes a Misbah container.
func (s *SandboxPort) Destroy(ctx context.Context, sandboxID string) error {
	if err := s.client.ContainerStop(ctx, sandboxID, false); err != nil {
		_ = s.client.ContainerStop(ctx, sandboxID, true)
	}
	return s.client.ContainerDestroy(ctx, sandboxID)
}

// List returns all Djinn-managed containers.
func (s *SandboxPort) List(ctx context.Context) ([]*model.ContainerInfo, error) {
	resp, err := s.client.ContainerList(ctx)
	if err != nil {
		return nil, fmt.Errorf("container list: %w", err)
	}
	return resp.Containers, nil
}

// Status returns the status of a single container.
func (s *SandboxPort) Status(ctx context.Context, name string) (*model.ContainerInfo, error) {
	info, err := s.client.ContainerStatus(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("container status: %w", err)
	}
	return info, nil
}

// Exec runs a command inside a running container.
func (s *SandboxPort) Exec(ctx context.Context, name string, cmd []string, timeout int64) (ExecResult, error) {
	resp, err := s.client.ContainerExec(ctx, name, cmd, timeout)
	if err != nil {
		return ExecResult{}, fmt.Errorf("container exec: %w", err)
	}
	return ExecResult{
		ExitCode: resp.ExitCode,
		Stdout:   resp.Stdout,
		Stderr:   resp.Stderr,
	}, nil
}

// ExecResult holds the output of a command executed inside a container.
type ExecResult struct {
	ExitCode int32
	Stdout   string
	Stderr   string
}

// --- Agent Space operations ---

// Diff returns files changed by the agent in the container's overlay.
func (s *SandboxPort) Diff(ctx context.Context, name string) ([]DiffEntry, error) {
	resp, err := s.client.ContainerDiff(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("container diff: %w", err)
	}
	return resp.Entries, nil
}

// Commit promotes selected files from the overlay to the real workspace.
func (s *SandboxPort) Commit(ctx context.Context, name string, paths []string) error {
	if err := s.client.ContainerCommit(ctx, name, paths); err != nil {
		return fmt.Errorf("container commit: %w", err)
	}
	return nil
}

// Events subscribes to container lifecycle events via SSE.
// Returns a channel that receives events. Close ctx to stop.
func (s *SandboxPort) Events(ctx context.Context, name string) (<-chan ContainerEvent, error) {
	return s.client.ContainerEvents(ctx, name)
}

// Logs returns captured stdout/stderr for a container.
func (s *SandboxPort) Logs(ctx context.Context, name string) (stdout, stderr string, err error) {
	resp, err := s.client.ContainerLogs(ctx, name)
	if err != nil {
		return "", "", fmt.Errorf("container logs: %w", err)
	}
	return resp.Stdout, resp.Stderr, nil
}

// Close releases the underlying daemon client.
func (s *SandboxPort) Close() {
	s.client.Close()
}
