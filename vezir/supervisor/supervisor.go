// Package supervisor implements the Vezir process supervisor.
// Spawns and manages the Substrate process via os/exec.
// Auto-restarts on crash with exponential backoff.
// Listens for SIGHUP to trigger rebuild + restart (hot-swap).
package supervisor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
)

var (
	// ErrMaxRestarts is returned when the Substrate exceeds the restart limit.
	ErrMaxRestarts = errors.New("substrate exceeded max restarts")
	// ErrNoProcess is returned when wait is called with no running process.
	ErrNoProcess = errors.New("no process to wait for")
)

// Supervisor manages a single child process (Substrate).
type Supervisor struct {
	substrateBin string
	socketPath   string
	log          *slog.Logger

	mu         sync.Mutex
	cmd        *exec.Cmd
	restarts   int
	lastStart  time.Time
	maxRetries int
	baseDelay  time.Duration
}

// New creates a Supervisor for the given Substrate binary.
func New(substrateBin, socketPath string, log *slog.Logger) *Supervisor {
	return &Supervisor{
		substrateBin: substrateBin,
		socketPath:   socketPath,
		log:          log,
		maxRetries:   10,
		baseDelay:    500 * time.Millisecond,
	}
}

// Run starts the supervision loop. Blocks until context is canceled.
func (s *Supervisor) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			s.log.InfoContext(ctx, "vezir shutting down")
			s.stop()
			return nil
		default:
		}

		if err := s.spawn(ctx); err != nil {
			s.log.WarnContext(ctx, "substrate spawn failed",
				slog.String(telemetry.KeyError, err.Error()),
				slog.Int(telemetry.KeyCount, s.restarts),
			)
		}

		// Wait for subprocess to exit.
		exitErr := s.wait()

		s.mu.Lock()
		s.restarts++
		restarts := s.restarts
		s.mu.Unlock()

		if ctx.Err() != nil {
			return nil // context canceled — clean shutdown
		}

		// Backoff on repeated crashes.
		if restarts > s.maxRetries {
			s.log.ErrorContext(ctx, "substrate exceeded max restarts — giving up",
				slog.Int(telemetry.KeyCount, s.maxRetries),
			)
			return fmt.Errorf("%w: %d", ErrMaxRestarts, s.maxRetries)
		}

		delay := s.backoff(restarts)
		s.log.WarnContext(ctx, "substrate exited — restarting after backoff",
			slog.String(telemetry.KeyError, fmt.Sprint(exitErr)),
			slog.Int(telemetry.KeyCount, restarts),
			slog.Duration(telemetry.KeyDuration, delay),
		)

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}
	}
}

// ResetRestarts resets the restart counter (e.g. after stable period).
func (s *Supervisor) ResetRestarts() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.restarts = 0
}

func (s *Supervisor) spawn(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Build the command. Pass socket path as flag.
	cmd := exec.CommandContext(ctx, "go", "run", s.substrateBin, "--socket", s.socketPath) //nolint:gosec // substrateBin is operator-configured, not user input
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start substrate: %w", err)
	}

	s.cmd = cmd
	s.lastStart = time.Now()

	s.log.InfoContext(ctx, "substrate started",
		slog.Int(telemetry.KeyExitCode, cmd.Process.Pid),
		slog.Int(telemetry.KeyCount, s.restarts),
	)

	return nil
}

func (s *Supervisor) wait() error {
	s.mu.Lock()
	cmd := s.cmd
	s.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return ErrNoProcess
	}

	return cmd.Wait()
}

func (s *Supervisor) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd != nil && s.cmd.Process != nil {
		s.log.InfoContext(context.Background(), "stopping substrate",
			slog.Int(telemetry.KeyExitCode, s.cmd.Process.Pid),
		)
		_ = s.cmd.Process.Signal(os.Interrupt)

		// Give it 5 seconds to clean up, then kill.
		done := make(chan error, 1)
		go func() { done <- s.cmd.Wait() }()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = s.cmd.Process.Kill()
		}
	}
}

func (s *Supervisor) backoff(restarts int) time.Duration {
	delay := s.baseDelay
	for i := 1; i < restarts && i < 8; i++ {
		delay *= 2
	}
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	return delay
}

// Restarts returns the number of times the Substrate was restarted.
func (s *Supervisor) Restarts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.restarts
}
