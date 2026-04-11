// app_hub.go — djinn hub: the GenSec daemon for seamless hot-swap.
//
// Usage:
//
//	djinn hub                    start hub on default socket (~/.djinn/hub.sock)
//	djinn hub --socket <path>    start hub on custom socket
//
// The hub is the persistent process. Both shell (TUI) and backend (LLM)
// connect to it as clients. Either side can restart independently.
//
// Typical dogfooding workflow:
//
//	Terminal 1:  djinn hub                     # stays alive
//	Terminal 2:  djinn                         # auto-connects as shell
//	Terminal 3:  djinn backend --socket <path> # auto-connects as backend
//	[rebuild]    go build && djinn             # shell reconnects, session preserved
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dpopsuev/djinn/hotswap"
	"github.com/dpopsuev/djinn/telemetry"
)

// DefaultHubSocket returns the default hub socket path.
func DefaultHubSocket() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".djinn")
	os.MkdirAll(dir, 0o700) //nolint:errcheck // best-effort directory creation
	return filepath.Join(dir, "hub.sock")
}

// RunHub starts the GenSec hub substrate.
func RunHub(args []string, stderr io.Writer) error {
	var socketPath string
	var spawnBackend bool
	// Simple arg parsing — no flag set needed for 2 flags.
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--socket":
			if i+1 < len(args) {
				i++
				socketPath = args[i]
			}
		case "--spawn-backend":
			spawnBackend = true
		}
	}
	if socketPath == "" {
		socketPath = DefaultHubSocket()
	}

	logResult := telemetry.Setup(telemetry.Options{Verbose: true})
	log := telemetry.For(logResult.Logger, "hub")

	h, err := hotswap.NewHub(socketPath, telemetry.For(log, "clutch"))
	if err != nil {
		return fmt.Errorf("start hub: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Info("hub started", "socket", socketPath)
	fmt.Fprintf(stderr, "djinn hub: listening on %s\n", socketPath)
	fmt.Fprintf(stderr, "djinn hub: connect shell:   djinn\n")
	fmt.Fprintf(stderr, "djinn hub: connect backend: djinn backend --socket %s\n", socketPath)

	// Auto-spawn backend if requested.
	if spawnBackend {
		go autoSpawnBackend(ctx, socketPath, log)
	}

	err = h.Run(ctx)
	h.Close()

	// Clean up socket file.
	os.Remove(socketPath) //nolint:errcheck // best-effort cleanup
	log.Info("hub stopped")
	return err
}

// autoSpawnBackend spawns `djinn backend` as a child and restarts on crash.
func autoSpawnBackend(ctx context.Context, socketPath string, log *slog.Logger) {
	for {
		if ctx.Err() != nil {
			return
		}

		// Small delay to let hub start accepting.
		time.Sleep(200 * time.Millisecond)

		exe, _ := os.Executable()
		cmd := exec.CommandContext(ctx, exe, "backend", "--socket", socketPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		log.Info("spawning backend", "cmd", cmd.String())
		if err := cmd.Run(); err != nil {
			if ctx.Err() != nil {
				return // graceful shutdown
			}
			log.Warn("backend exited", "error", err)
			log.Info("restarting backend in 1s...")
			time.Sleep(1 * time.Second)
		}
	}
}

// HubSocketExists checks if a hub is running on the default socket.
func HubSocketExists() (string, bool) {
	path := DefaultHubSocket()
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	// Try connecting to verify it's alive (not a stale socket file).
	conn, err := net.DialTimeout("unix", path, 500*time.Millisecond)
	if err != nil {
		// Stale socket — clean it up.
		os.Remove(path) //nolint:errcheck // best-effort cleanup
		return "", false
	}
	conn.Close()
	return path, true
}

// connectToHubAs connects to the hub and registers with the given role.
func connectToHubAs(socketPath, role string) (*hotswap.SocketTransport, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("connect to hub: %w", err)
	}
	enc := json.NewEncoder(conn)
	if err := enc.Encode(hotswap.RegisterMsg{Role: role}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("register as %s: %w", role, err)
	}
	return hotswap.WrapConn(conn), nil
}

// ConnectToHub connects to the hub as a shell client.
func ConnectToHub(socketPath string) (*hotswap.SocketTransport, error) {
	return connectToHubAs(socketPath, "shell")
}

// ConnectToHubAsBackend connects to the hub as a backend client.
func ConnectToHubAsBackend(socketPath string) (*hotswap.SocketTransport, error) {
	return connectToHubAs(socketPath, "backend")
}
