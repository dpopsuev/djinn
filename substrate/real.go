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
	canonPkg "github.com/dpopsuev/djinn/canon"
	"github.com/dpopsuev/djinn/hook"
	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/vessel"
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

	// Orchestration (GOL-162, GOL-163).
	scheduler  Scheduler                  // role routing: signal → next role
	ring       *telemetry.TraceProjection // trace ring with TraceID propagation
	recorder   *ToolEventRecorder         // bridges tool calls → EventLog
	dispatcher *hook.EventDispatcher      // unified hook runtime (GOL-161)

	// L2 Cache Services (GOL-174, NED-55).
	canon canonPkg.Canon // VCS state cache

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

	// ORANGE: warn about missing integration wiring (GOL-162).
	if s.scheduler == nil {
		s.log.WarnContext(context.Background(), "substrate: no Scheduler configured — role routing unavailable",
			slog.String(telemetry.KeyComponent, "substrate"),
		)
	}
	if s.recorder == nil {
		s.log.WarnContext(context.Background(), "substrate: no ToolRecorder configured — tool calls not traced",
			slog.String(telemetry.KeyComponent, "substrate"),
		)
	}
	if s.ring == nil {
		s.log.WarnContext(context.Background(), "substrate: no TraceProjection configured — TraceID propagation disabled",
			slog.String(telemetry.KeyComponent, "substrate"),
		)
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

// WithScheduler sets the role routing function (signal → next role).
func WithScheduler(sched Scheduler) Option {
	return func(s *RealSubstrate) { s.scheduler = sched }
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

// WithHookDispatcher sets the unified hook runtime (GOL-161).
func WithHookDispatcher(d *hook.EventDispatcher) Option {
	return func(s *RealSubstrate) { s.dispatcher = d }
}

// WithCanon sets the VCS cache service (GOL-174).
func WithCanon(c canonPkg.Canon) Option {
	return func(s *RealSubstrate) { s.canon = c }
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

// IntegrationServices returns options for the full trace + recorder + director
// stack (GOL-162). Builds on DefaultServices and adds TraceProjection,
// ToolEventRecorder, and LocalDirector. Pass a logger for YELLOW summaries.
// eventLogPath: when non-empty, uses DurableJSONLines (events survive restarts).
// When empty, uses in-memory MemLog (tests, ephemeral sessions).
func IntegrationServices(log *slog.Logger, eventLogPath string) []Option {
	eventLog := createEventLog(eventLogPath, log)

	ring := telemetry.NewTraceProjection(1000).WithEventLog(eventLog) //nolint:mnd // sensible default
	if log != nil {
		ring.WithLogger(log)
	}
	recorder := NewToolEventRecorder(eventLog, ring.TraceID)

	return []Option{
		WithEventLog(eventLog),
		WithL2Cache(djinncache.NewMemCache()),
		WithTraceProjection(ring),
		WithToolRecorder(recorder),
		WithScheduler(DefaultScheduler()),
	}
}

// createEventLog builds the EventLog: durable if path is set, in-memory otherwise.
func createEventLog(path string, log *slog.Logger) signal.EventLog {
	if path == "" {
		return signal.NewMemLog()
	}
	durable, err := signal.NewDurableJSONLines(path)
	if err != nil {
		if log != nil {
			log.WarnContext(context.Background(), "durable event log failed, falling back to memory",
				slog.String(telemetry.KeyPath, path),
				slog.String(telemetry.KeyError, err.Error()),
			)
		}
		return signal.NewMemLog()
	}
	if count, rErr := durable.Replay(); rErr == nil && log != nil && count > 0 {
		log.InfoContext(context.Background(), "event log replayed",
			slog.Int(telemetry.KeyCount, count),
		)
	}
	return durable
}

// --- Substrate Interface ---

func (s *RealSubstrate) Tools() tool.Executor           { return s.tools }
func (s *RealSubstrate) Envelope() *middleware.Envelope { return s.envelope }
func (s *RealSubstrate) EventLog() signal.EventLog      { return s.eventLog }
func (s *RealSubstrate) L2() djinncache.Cache           { return s.l2 }

// WorkDir returns the workspace directory.
func (s *RealSubstrate) WorkDir() string { return s.workDir }

// Scheduler returns the role routing function (nil if not configured).
func (s *RealSubstrate) SchedulerRef() Scheduler { return s.scheduler }

// TraceProjection returns the trace ring (nil if not configured).
func (s *RealSubstrate) TraceProjection() *telemetry.TraceProjection { return s.ring }

// ToolRecorder returns the tool event recorder (nil if not configured).
func (s *RealSubstrate) ToolRecorder() *ToolEventRecorder { return s.recorder }

// HookDispatcher returns the unified hook runtime (nil if not configured).
func (s *RealSubstrate) HookDispatcher() *hook.EventDispatcher { return s.dispatcher }

// Canon returns the VCS cache service (nil if not configured).
func (s *RealSubstrate) Canon() canonPkg.Canon { return s.canon }

// SetHookDispatcher sets the hook dispatcher after construction (late binding).
func (s *RealSubstrate) SetHookDispatcher(d *hook.EventDispatcher) { s.dispatcher = d }

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
