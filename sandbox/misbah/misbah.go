// Package misbah implements broker.SandboxPort using the Misbah daemon.
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
	specVersion       = "1.0"
	containerPrefix   = "djinn-"
	defaultCwd        = "/workspace"
	defaultCommand    = "/bin/sh"
	mountTypeBind     = "bind"
	mountTypeTmpfs    = "tmpfs"
	mountOptionRO     = "ro"
	mountOptionRW     = "rw"
)

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
func (s *SandboxPort) Create(ctx context.Context, scope tier.Scope) (string, error) {
	name := fmt.Sprintf("%s%s-%d", containerPrefix, scope.Name, s.counter.Add(1))

	spec := s.buildSpec(name, scope)
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
		// Best-effort stop before destroy
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

// Close releases the underlying daemon client.
func (s *SandboxPort) Close() {
	s.client.Close()
}

func (s *SandboxPort) buildSpec(name string, scope tier.Scope) *model.ContainerSpec {
	spec := &model.ContainerSpec{
		Version: specVersion,
		Metadata: model.ContainerMetadata{
			Name: name,
		},
		Process: model.ProcessSpec{
			Command: []string{defaultCommand},
			Cwd:     defaultCwd,
		},
		Namespaces: model.NamespaceSpec{
			User:  true,
			Mount: true,
		},
		Mounts: []model.MountSpec{
			{
				Type:        mountTypeBind,
				Source:      s.workspace,
				Destination: defaultCwd,
				Options:     mountOptionsForTier(scope.Level),
			},
			{
				Type:        mountTypeTmpfs,
				Destination: "/tmp",
			},
		},
	}

	// Set tier config for Misbah's built-in tier isolation
	if scope.Name != "" {
		spec.TierConfig = &model.TierConfig{
			Tier:          scope.Level.String(),
			WritablePaths: []string{scope.Name},
		}
	}

	return spec
}

func mountOptionsForTier(level tier.TierLevel) []string {
	switch level {
	case tier.Eco:
		return []string{mountTypeBind, mountOptionRO}
	case tier.Sys:
		return []string{mountTypeBind, mountOptionRO}
	case tier.Com:
		return []string{mountTypeBind, mountOptionRW}
	case tier.Mod:
		return []string{mountTypeBind, mountOptionRW}
	default:
		return []string{mountTypeBind, mountOptionRO}
	}
}
