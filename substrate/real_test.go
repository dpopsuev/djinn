package substrate

import (
	"context"
	"testing"

	"github.com/dpopsuev/battery/service"
	"github.com/dpopsuev/troupe/signal"
)

// --- ISP interfaces ---

// Spawner is the narrow interface for agent spawning.
// Consumers that only need to spawn agents take this.
type Spawner interface {
	Spawn(ctx context.Context, cfg SpawnConfig) (string, error)
}

// HealthChecker is the narrow interface for health queries.
type HealthChecker interface {
	Health() service.HealthReport
}

// EventLogger is the narrow interface for event log access.
type EventLogger interface {
	EventLog() signal.EventLog
}

// --- RED tests: must fail until GREEN ---

func TestNew_WithOptions(t *testing.T) {
	log := &signal.MemLog{}
	sub := New("/tmp/test",
		WithEventLog(log),
	)

	if sub.EventLog() != log {
		t.Fatal("EventLog should be the one we provided")
	}
	if sub.WorkDir() != "/tmp/test" {
		t.Fatalf("WorkDir = %q, want /tmp/test", sub.WorkDir())
	}
}

func TestNew_DefaultServices(t *testing.T) {
	sub := New("/tmp/test", DefaultServices()...)

	if sub.EventLog() == nil {
		t.Fatal("DefaultServices should provide EventLog")
	}
	if sub.L2() == nil {
		t.Fatal("DefaultServices should provide L2 cache")
	}
	if sub.Health().Status != service.Healthy {
		t.Fatalf("health = %q, want healthy", sub.Health().Status)
	}
}

func TestNew_VesselCreatesWorkspaceRooted(t *testing.T) {
	workDir := t.TempDir()
	sub := New(workDir, DefaultServices()...)

	v := sub.Vessel(SpawnConfig{Role: "test"})
	defer v.Close(context.Background()) //nolint:errcheck // test cleanup

	if v.WorkDir() != workDir {
		t.Fatalf("Vessel.WorkDir() = %q, want %q", v.WorkDir(), workDir)
	}
}

func TestNew_ISP_Spawner(t *testing.T) {
	sub := New("/tmp/test", DefaultServices()...)
	var s Spawner = sub // compile-time ISP check
	id, err := s.Spawn(context.Background(), SpawnConfig{Role: "test"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if id == "" {
		t.Fatal("Spawn should return non-empty ID")
	}
}

func TestNew_ISP_HealthChecker(t *testing.T) {
	sub := New("/tmp/test", DefaultServices()...)
	var h HealthChecker = sub // compile-time ISP check
	if h.Health().Status != service.Healthy {
		t.Fatalf("health = %q", h.Health().Status)
	}
}

func TestNew_ISP_EventLogger(t *testing.T) {
	sub := New("/tmp/test", DefaultServices()...)
	var e EventLogger = sub // compile-time ISP check
	if e.EventLog() == nil {
		t.Fatal("EventLog should not be nil")
	}
}
