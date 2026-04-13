// real.go — RealSubstrate: production implementation with functional options.
//
// New(workDir, ...Option) constructs a fully-wired Substrate.
// DefaultServices() bundles sane defaults. TestServices() for testkit.
// ISP: consumers take Spawner, HealthChecker, or EventLogger interfaces.
//
// DJN-TSK-1064, GOL-159
package substrate

import (
	"context"
	"log/slog"
	"sync"

	"github.com/dpopsuev/battery/middleware"
	"github.com/dpopsuev/battery/service"
	"github.com/dpopsuev/battery/tool"
	djinncache "github.com/dpopsuev/djinn/cache"
	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/vessel"
	"github.com/dpopsuev/troupe/director"
	"github.com/dpopsuev/troupe/signal"
)

var _ Substrate = (*RealSubstrate)(nil)

// RealSubstrate is the production Substrate implementation.
// Constructed via New() with functional options.
type RealSubstrate struct {
	mu       sync.Mutex
	workDir  string
	tools    tool.Executor
	envelope *middleware.Envelope
	eventLog signal.EventLog
	l2       djinncache.Cache
	log      *slog.Logger

	// Orchestration (GOL-162).
	director director.Director          // troupe Director interface — LocalDirector or external
	ring     *telemetry.TraceProjection // trace ring with TraceID propagation
	recorder *ToolEventRecorder         // bridges tool calls → EventLog

	// Lifecycle tracking.
	observations []Observation
	spawned      []SpawnConfig
	killed       []string
}

// Option configures a RealSubstrate.
type Option func(*RealSubstrate)

// New creates a production Substrate with functional options.
// workDir is the primary workspace directory (required).
// Use DefaultServices() for batteries-included, or compose your own.
func New(workDir string, opts ...Option) *RealSubstrate {
	s := &RealSubstrate{
		workDir: workDir,
		l2:      djinncache.NewMemCache(),
		log:     telemetry.Nop(),
	}
	for _, opt := range opts {
		opt(s)
	}
	// Ensure EventLog exists (safe default).
	if s.eventLog == nil {
		s.eventLog = signal.NewMemLog()
	}
	// Ensure tools exist (workspace-rooted).
	if s.tools == nil {
		reg := builtin.NewRegistryWithWorkDir(workDir)
		builtin.RegisterBuiltinTools(reg, workDir, workDir)
		s.tools = reg
	}
	return s
}

// --- Functional Options ---

// WithEventLog sets the unified event log.
func WithEventLog(log signal.EventLog) Option {
	return func(s *RealSubstrate) { s.eventLog = log }
}

// WithL2Cache sets the shared L2 cache.
func WithL2Cache(c djinncache.Cache) Option {
	return func(s *RealSubstrate) { s.l2 = c }
}

// WithTools sets the tool executor.
func WithTools(t tool.Executor) Option {
	return func(s *RealSubstrate) { s.tools = t }
}

// WithEnvelopeMiddleware sets the Gate/Enrich/Execute/Record pipeline.
func WithEnvelopeMiddleware(e *middleware.Envelope) Option {
	return func(s *RealSubstrate) { s.envelope = e }
}

// WithSubstrateLogger sets the structured logger.
func WithSubstrateLogger(l *slog.Logger) Option {
	return func(s *RealSubstrate) { s.log = l }
}

// WithDirector sets the orchestration Director (troupe/director.Director).
// LocalDirector for built-in pipeline, CircuitDirector for Origami, etc.
func WithDirector(d director.Director) Option {
	return func(s *RealSubstrate) { s.director = d }
}

// WithTraceProjection sets the trace ring with EventLog bridge and TraceID propagation.
func WithTraceProjection(r *telemetry.TraceProjection) Option {
	return func(s *RealSubstrate) { s.ring = r }
}

// WithToolRecorder sets the tool event recorder and registers it as the
// Battery default recorder (middleware.SetDefaultRecorder).
func WithToolRecorder(r *ToolEventRecorder) Option {
	return func(s *RealSubstrate) {
		s.recorder = r
		middleware.SetDefaultRecorder(r)
	}
}

// --- Module Grouping ---

// DefaultServices returns options for a batteries-included Substrate.
// EventLog, L2 cache, workspace-rooted tools — all with sane defaults.
func DefaultServices() []Option {
	return []Option{
		WithEventLog(signal.NewMemLog()),
		WithL2Cache(djinncache.NewMemCache()),
	}
}

// --- Substrate Interface ---

func (s *RealSubstrate) Tools() tool.Executor           { return s.tools }
func (s *RealSubstrate) Envelope() *middleware.Envelope { return s.envelope }
func (s *RealSubstrate) EventLog() signal.EventLog      { return s.eventLog }
func (s *RealSubstrate) L2() djinncache.Cache           { return s.l2 }

// WorkDir returns the workspace directory.
func (s *RealSubstrate) WorkDir() string { return s.workDir }

// Director returns the orchestration Director (nil if not configured).
func (s *RealSubstrate) Director() director.Director { return s.director }

// TraceProjection returns the trace ring (nil if not configured).
func (s *RealSubstrate) TraceProjection() *telemetry.TraceProjection { return s.ring }

// ToolRecorder returns the tool event recorder (nil if not configured).
func (s *RealSubstrate) ToolRecorder() *ToolEventRecorder { return s.recorder }

func (s *RealSubstrate) Observe(_ context.Context, obs Observation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observations = append(s.observations, obs)
}

func (s *RealSubstrate) Vessel(cfg SpawnConfig) vessel.Vessel {
	reg := builtin.NewRegistryWithWorkDir(s.workDir)
	builtin.RegisterBuiltinTools(reg, s.workDir, s.workDir)
	return vessel.NewWorkspaceVessel(s.workDir, reg, s.eventLog)
}

func (s *RealSubstrate) Spawn(_ context.Context, cfg SpawnConfig) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := cfg.Role + "-0"
	s.spawned = append(s.spawned, cfg)
	return id, nil
}

func (s *RealSubstrate) Kill(_ context.Context, agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.killed = append(s.killed, agentID)
	return nil
}

func (s *RealSubstrate) Health() service.HealthReport {
	return service.HealthReport{Status: service.Healthy}
}
