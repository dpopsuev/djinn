// swap.go — Binary hot-swap via hub backend restart (TSK-492).
//
// SwapBackend kills the current backend and starts a new binary.
// The hub queues messages during the swap. The new backend connects
// to the hub and receives queued messages. Session is saved before
// swap and loaded with --resume-session by the new binary.
package clutch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// SwapConfig describes how to swap the backend binary.
type SwapConfig struct {
	NewBinaryPath string   // path to the new compiled binary
	SessionFile   string   // path to saved session JSON
	SocketPath    string   // hub socket path for reconnection
	ExtraArgs     []string // additional args for the new binary
}

// SwapBackend replaces the backend process with a new binary.
// The hub stays alive — only the backend restarts.
//
// Process:
//  1. Disconnect current backend (hub queues messages)
//  2. Start new binary with --resume-session pointing to saved session
//  3. New binary connects to hub as backend
//  4. Hub drains queued messages to new backend
//
// Returns the started process. Caller is responsible for waiting on it.
func SwapBackend(ctx context.Context, hub *Hub, cfg SwapConfig) (*os.Process, error) {
	// Verify new binary exists.
	if _, err := os.Stat(cfg.NewBinaryPath); err != nil {
		return nil, fmt.Errorf("new binary not found: %w", err)
	}

	// Disconnect current backend — hub will queue messages.
	hub.mu.Lock()
	if hub.backend != nil {
		hub.backend.Close()
		hub.backend = nil
	}
	if hub.backendRelay != nil {
		hub.backendRelay()
		hub.backendRelay = nil
	}
	hub.mu.Unlock()

	// Start new binary.
	args := []string{
		"backend",
		"--socket", cfg.SocketPath,
		"--resume-session", cfg.SessionFile,
	}
	args = append(args, cfg.ExtraArgs...)

	cmd := exec.CommandContext(ctx, cfg.NewBinaryPath, args...) //nolint:gosec // binary path from trusted self-heal
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start new backend: %w", err)
	}

	return cmd.Process, nil
}

// KeepRollback copies the current binary to a rollback path.
func KeepRollback(currentBinary, rollbackPath string) error {
	src, err := os.ReadFile(currentBinary) //nolint:gosec // trusted path
	if err != nil {
		return fmt.Errorf("read current binary: %w", err)
	}
	return os.WriteFile(rollbackPath, src, 0o755) //nolint:gosec // executable permission intentional
}
