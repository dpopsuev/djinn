// Package sandbox defines the Strategy interface for execution isolation.
// Backends (Misbah, bubblewrap, podman, etc.) implement this interface.
// The workspace manifest declares the desired backend + level.
package sandbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
)

// Handle identifies a running sandbox instance.
type Handle string

// ExecResult holds the output of a command executed inside a sandbox.
type ExecResult struct {
	ExitCode int32
	Stdout   string
	Stderr   string
}

// Sandbox is the Strategy interface for execution isolation.
type Sandbox interface {
	Create(ctx context.Context, level string, repos []string) (Handle, error)
	Destroy(ctx context.Context, handle Handle) error
	Exec(ctx context.Context, handle Handle, cmd []string, timeout int64) (ExecResult, error)
	Name() string
}

// Levels.
const (
	LevelNone      = "none"
	LevelNamespace = "namespace"
	LevelContainer = "container"
	LevelKata      = "kata"
)

// Sentinel errors.
var (
	ErrBackendNotFound = errors.New("sandbox backend not found")
	ErrBackendFailed   = errors.New("sandbox backend failed to start")
)

// Registry of sandbox backends.
var (
	mu       sync.RWMutex
	backends = make(map[string]func() (Sandbox, error))
)

// Register adds a sandbox backend factory.
func Register(name string, factory func() (Sandbox, error)) {
	mu.Lock()
	defer mu.Unlock()
	backends[name] = factory
}

// Get returns a sandbox backend by name.
func Get(name string) (Sandbox, error) {
	mu.RLock()
	factory, ok := backends[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %q (available: %v)", ErrBackendNotFound, name, Available())
	}
	return factory()
}

// Available returns registered backend names.
func Available() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(backends))
	for name := range backends {
		names = append(names, name)
	}
	return names
}

// LoggingSandbox wraps a Sandbox with structured logging.
// All operations are logged; the underlying sandbox does the real work.
type LoggingSandbox struct {
	inner Sandbox
	log   *slog.Logger
}

// WithLogging wraps a Sandbox with structured logging.
// Pass nil for log to discard all output.
func WithLogging(sb Sandbox, log *slog.Logger) *LoggingSandbox {
	if log == nil {
		log = telemetry.Nop()
	}
	return &LoggingSandbox{inner: sb, log: log}
}

func (s *LoggingSandbox) Name() string { return s.inner.Name() }

func (s *LoggingSandbox) Create(ctx context.Context, level string, repos []string) (Handle, error) {
	start := time.Now()
	handle, err := s.inner.Create(ctx, level, repos)
	if err != nil {
		// Orange: sandbox creation failure
		s.log.WarnContext(ctx, "sandbox create failed",
			slog.String(telemetry.KeyAction, "create"),
			slog.String(telemetry.KeyBackend, s.inner.Name()),
			slog.String(telemetry.KeyLevel, level),
			slog.String(telemetry.KeyError, err.Error()),
		)
		return handle, err
	}
	// Yellow: sandbox created
	s.log.InfoContext(ctx, "sandbox created",
		slog.String(telemetry.KeyAction, "create"),
		slog.String(telemetry.KeyBackend, s.inner.Name()),
		slog.String(telemetry.KeyLevel, level),
		slog.Duration(telemetry.KeyDuration, time.Since(start)),
	)
	return handle, nil
}

func (s *LoggingSandbox) Destroy(ctx context.Context, handle Handle) error {
	err := s.inner.Destroy(ctx, handle)
	if err != nil {
		// Orange: destroy failure
		s.log.WarnContext(ctx, "sandbox destroy failed",
			slog.String(telemetry.KeyAction, "destroy"),
			slog.String(telemetry.KeyBackend, s.inner.Name()),
			slog.String(telemetry.KeyError, err.Error()),
		)
		return err
	}
	// Yellow: sandbox destroyed
	s.log.InfoContext(ctx, "sandbox destroyed",
		slog.String(telemetry.KeyAction, "destroy"),
		slog.String(telemetry.KeyBackend, s.inner.Name()),
	)
	return nil
}

func (s *LoggingSandbox) Exec(ctx context.Context, handle Handle, cmd []string, timeout int64) (ExecResult, error) {
	start := time.Now()
	// Yellow: exec started
	s.log.DebugContext(ctx, "sandbox exec started",
		slog.String(telemetry.KeyAction, "exec"),
		slog.String(telemetry.KeyBackend, s.inner.Name()),
	)
	result, err := s.inner.Exec(ctx, handle, cmd, timeout)
	if err != nil {
		// Orange: exec failure
		s.log.WarnContext(ctx, "sandbox exec failed",
			slog.String(telemetry.KeyAction, "exec"),
			slog.String(telemetry.KeyBackend, s.inner.Name()),
			slog.String(telemetry.KeyError, err.Error()),
			slog.Duration(telemetry.KeyDuration, time.Since(start)),
		)
		return result, err
	}
	if result.ExitCode != 0 {
		// Orange: non-zero exit code
		s.log.WarnContext(ctx, "sandbox exec non-zero exit",
			slog.String(telemetry.KeyAction, "exec"),
			slog.String(telemetry.KeyBackend, s.inner.Name()),
			slog.Int(telemetry.KeyExitCode, int(result.ExitCode)),
			slog.Duration(telemetry.KeyDuration, time.Since(start)),
		)
	} else {
		// Yellow: exec completed
		s.log.DebugContext(ctx, "sandbox exec completed",
			slog.String(telemetry.KeyAction, "exec"),
			slog.String(telemetry.KeyBackend, s.inner.Name()),
			slog.Int(telemetry.KeyExitCode, int(result.ExitCode)),
			slog.Duration(telemetry.KeyDuration, time.Since(start)),
		)
	}
	return result, nil
}

// Ensure interface compliance.
var _ Sandbox = (*LoggingSandbox)(nil)
