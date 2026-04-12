package substrate

import (
	"context"
	"sync"

	"github.com/dpopsuev/battery/middleware"
	"github.com/dpopsuev/battery/service"
	"github.com/dpopsuev/battery/tool"
	djinncache "github.com/dpopsuev/djinn/cache"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/vessel"
	"github.com/dpopsuev/troupe/signal"
)

// StubSubstrate implements Substrate for testing.
// Records all observations and spawn/kill calls. Returns configurable tools.
type StubSubstrate struct {
	mu           sync.Mutex
	tools        tool.Executor
	envelope     *middleware.Envelope
	eventLog     signal.EventLog
	l2           djinncache.Cache
	health       service.HealthReport
	workDir      string // workspace directory for Vessel creation
	Observations []Observation
	Spawned      []SpawnConfig
	Killed       []string
}

var _ Substrate = (*StubSubstrate)(nil)

// NewStubSubstrate creates a test Substrate with the given tool executor and event log.
func NewStubSubstrate(tools tool.Executor, log signal.EventLog) *StubSubstrate {
	return &StubSubstrate{
		tools:    tools,
		eventLog: log,
		l2:       djinncache.NewMemCache(),
		health:   service.HealthReport{Status: service.Healthy},
	}
}

func (s *StubSubstrate) Tools() tool.Executor           { return s.tools }
func (s *StubSubstrate) Envelope() *middleware.Envelope { return s.envelope }
func (s *StubSubstrate) EventLog() signal.EventLog      { return s.eventLog }

func (s *StubSubstrate) Observe(_ context.Context, obs Observation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Observations = append(s.Observations, obs)
}

func (s *StubSubstrate) Spawn(_ context.Context, cfg SpawnConfig) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := cfg.Role + "-0"
	s.Spawned = append(s.Spawned, cfg)
	return id, nil
}

func (s *StubSubstrate) Kill(_ context.Context, agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Killed = append(s.Killed, agentID)
	return nil
}

func (s *StubSubstrate) L2() djinncache.Cache { return s.l2 }

func (s *StubSubstrate) Vessel(cfg SpawnConfig) vessel.Vessel {
	workDir := s.workDir
	if workDir == "" {
		workDir = "/tmp/workspace"
	}
	// Create workspace-rooted tools so relative paths resolve correctly.
	reg := builtin.NewRegistryWithWorkDir(workDir)
	return vessel.NewWorkspaceVessel(workDir, reg, s.eventLog)
}

// SetWorkDir configures the workspace directory for Vessel creation.
func (s *StubSubstrate) SetWorkDir(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workDir = dir
}

func (s *StubSubstrate) Health() service.HealthReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.health
}

// SetHealth configures the health report for testing.
func (s *StubSubstrate) SetHealth(h service.HealthReport) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health = h
}

// SetEnvelope configures the envelope for testing.
func (s *StubSubstrate) SetEnvelope(e *middleware.Envelope) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.envelope = e
}
