package miraged

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dpopsuev/battery/service"
	"github.com/dpopsuev/battery/tool"
	"github.com/dpopsuev/troupe/testkit"
)

// stubExecutor is a minimal tool.Executor for testing.
type stubExecutor struct{}

func (stubExecutor) Execute(_ context.Context, _ string, _ json.RawMessage) (string, error) {
	return "ok", nil
}
func (stubExecutor) All() []tool.Tool { return nil }
func (stubExecutor) Names() []string  { return []string{"stub"} }

func TestStubSubstrate_Tools(t *testing.T) {
	exec := stubExecutor{}
	s := NewStubSubstrate(exec, testkit.NewStubEventLog())

	if s.Tools() == nil {
		t.Fatal("Tools() should return executor")
	}
	names := s.Tools().Names()
	if len(names) != 1 || names[0] != "stub" {
		t.Fatalf("Names = %v, want [stub]", names)
	}
}

func TestStubSubstrate_Observe(t *testing.T) {
	s := NewStubSubstrate(stubExecutor{}, testkit.NewStubEventLog())
	ctx := context.Background()

	s.Observe(ctx, Observation{AgentID: "exec-0", Tool: "Read", Duration: 42})
	s.Observe(ctx, Observation{AgentID: "exec-0", Tool: "Write", Duration: 100})

	if len(s.Observations) != 2 {
		t.Fatalf("observations = %d, want 2", len(s.Observations))
	}
	if s.Observations[0].Tool != "Read" {
		t.Fatalf("first tool = %q, want Read", s.Observations[0].Tool)
	}
}

func TestStubSubstrate_SpawnAndKill(t *testing.T) {
	s := NewStubSubstrate(stubExecutor{}, testkit.NewStubEventLog())
	ctx := context.Background()

	id, err := s.Spawn(ctx, SpawnConfig{Role: "executor", Model: "haiku"})
	if err != nil {
		t.Fatal(err)
	}
	if id != "executor-0" {
		t.Fatalf("id = %q, want executor-0", id)
	}
	if len(s.Spawned) != 1 {
		t.Fatalf("spawned = %d, want 1", len(s.Spawned))
	}

	if err := s.Kill(ctx, id); err != nil {
		t.Fatal(err)
	}
	if len(s.Killed) != 1 || s.Killed[0] != "executor-0" {
		t.Fatalf("killed = %v", s.Killed)
	}
}

func TestStubSubstrate_Health(t *testing.T) {
	s := NewStubSubstrate(stubExecutor{}, testkit.NewStubEventLog())

	h := s.Health()
	if h.Status != service.Healthy {
		t.Fatalf("health = %q, want healthy", h.Status)
	}

	s.SetHealth(service.HealthReport{Status: service.Degraded, Message: "high load"})
	h = s.Health()
	if h.Status != service.Degraded {
		t.Fatalf("health = %q, want degraded", h.Status)
	}
}
