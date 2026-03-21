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
		return []string{"bind", "ro"}
	case tier.Sys:
		return []string{"bind", "ro"}
	case tier.Com:
		return []string{"bind", "rw"}
	case tier.Mod:
		return []string{"bind", "rw"}
	default:
		return []string{"bind", "ro"}
	}
}
