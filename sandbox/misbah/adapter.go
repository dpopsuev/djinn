// adapter.go — MisbahSandbox implements sandbox.Sandbox using the Misbah daemon.
// Bridges the broker.SandboxPort (tier-scoped) to sandbox.Sandbox (level+repos).
// Fails fast if the Misbah daemon is unreachable at startup.
package misbah

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/dpopsuev/djinn/sandbox"
	"github.com/dpopsuev/djinn/tier"
)

// DefaultSocketPath is the standard Misbah daemon socket location.
const DefaultSocketPath = "/run/misbah/permission.sock"

// MisbahSandbox implements sandbox.Sandbox backed by a Misbah daemon.
type MisbahSandbox struct {
	port *SandboxPort
}

// NewMisbahSandbox creates a sandbox backed by the Misbah daemon.
// Fails fast if the daemon is unreachable (security: never degrade silently).
func NewMisbahSandbox(socketPath, workspace string) (*MisbahSandbox, error) {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("misbah daemon unreachable at %s: %w", socketPath, err)
	}
	conn.Close()
	return &MisbahSandbox{port: New(socketPath, workspace)}, nil
}

func (m *MisbahSandbox) Create(ctx context.Context, level string, repos []string) (sandbox.Handle, error) {
	name := "workspace"
	if len(repos) > 0 {
		name = repos[0]
	}
	scope := tier.Scope{
		Level: tierLevelFromString(level),
		Name:  name,
	}
	id, err := m.port.Create(ctx, scope)
	if err != nil {
		return "", err
	}
	return sandbox.Handle(id), nil
}

func (m *MisbahSandbox) Destroy(ctx context.Context, handle sandbox.Handle) error {
	return m.port.Destroy(ctx, string(handle))
}

func (m *MisbahSandbox) Exec(ctx context.Context, handle sandbox.Handle, cmd []string, timeout int64) (sandbox.ExecResult, error) {
	result, err := m.port.Exec(ctx, string(handle), cmd, timeout)
	if err != nil {
		return sandbox.ExecResult{}, err
	}
	return sandbox.ExecResult{
		ExitCode: result.ExitCode,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
	}, nil
}

func (m *MisbahSandbox) Name() string { return "misbah" }

func (m *MisbahSandbox) Close() {
	m.port.Close()
}

// tierLevelFromString converts a sandbox level string to a tier.TierLevel.
func tierLevelFromString(level string) tier.TierLevel {
	switch level {
	case sandbox.LevelNamespace:
		return tier.Sys
	case sandbox.LevelContainer:
		return tier.Com
	case sandbox.LevelKata:
		return tier.Mod
	default:
		return tier.Eco
	}
}

// Ensure interface compliance.
var _ sandbox.Sandbox = (*MisbahSandbox)(nil)

// init registers the Misbah backend in the sandbox registry.
// The factory tries to connect; if the daemon is down, Get() returns an error.
func init() {
	sandbox.Register("misbah", func() (sandbox.Sandbox, error) {
		return NewMisbahSandbox(DefaultSocketPath, "")
	})
}
