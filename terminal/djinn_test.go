package terminal

import (
	"context"
	"testing"
)

func TestDjinn_ImplementsTerminal(t *testing.T) {
	var _ Terminal = (*Djinn)(nil)
	var _ Controller = (*Djinn)(nil)
	var _ Viewer = (*Djinn)(nil)
}

func TestDjinn_Command_Op(t *testing.T) {
	d := NewDjinn()
	ctx := context.Background()

	// Query current.
	out, err := d.Command(ctx, "op", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "operation: agent" {
		t.Errorf("op = %q, want 'operation: agent'", out)
	}

	// Set to ask.
	out, err = d.Command(ctx, "op", []string{"ask"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "operation → ask" {
		t.Errorf("op set = %q", out)
	}

	// Invalid.
	_, err = d.Command(ctx, "op", []string{"bogus"})
	if err == nil {
		t.Error("expected error for invalid operation")
	}
}

func TestDjinn_Command_Capacity(t *testing.T) {
	d := NewDjinn()
	ctx := context.Background()

	// Query.
	out, err := d.Command(ctx, "ac", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "agents: 0/1" {
		t.Errorf("ac = %q, want 'agents: 0/1'", out)
	}

	// Increment.
	out, err = d.Command(ctx, "ac+", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "capacity → 0/2" {
		t.Errorf("ac+ = %q", out)
	}

	// Set.
	out, err = d.Command(ctx, "ac", []string{"5"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "capacity → 0/5" {
		t.Errorf("ac 5 = %q", out)
	}

	// Decrement.
	out, err = d.Command(ctx, "ac-", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "capacity → 0/4" {
		t.Errorf("ac- = %q", out)
	}
}

func TestDjinn_Command_Envelope(t *testing.T) {
	d := NewDjinn()
	ctx := context.Background()

	// Query.
	out, err := d.Command(ctx, "envelope", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Error("envelope query should return status")
	}

	// Off.
	out, err = d.Command(ctx, "envelope", []string{"off"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "envelope → off" {
		t.Errorf("envelope off = %q", out)
	}

	// On.
	out, err = d.Command(ctx, "envelope", []string{"on"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "envelope → on" {
		t.Errorf("envelope on = %q", out)
	}

	// Every N.
	out, err = d.Command(ctx, "envelope", []string{"every", "5"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "envelope checkpoint every 5 tasks" {
		t.Errorf("envelope every = %q", out)
	}
}

func TestDjinn_Command_Unknown_DelegatesToHandler(t *testing.T) {
	d := NewDjinn()
	d.OnCommand = func(_ context.Context, name string, args []string) (string, error) {
		return "handled: " + name, nil
	}

	out, err := d.Command(context.Background(), "custom", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "handled: custom" {
		t.Errorf("custom = %q", out)
	}
}

func TestDjinn_Command_Unknown_NoHandler(t *testing.T) {
	d := NewDjinn()
	_, err := d.Command(context.Background(), "bogus", nil)
	if err == nil {
		t.Error("expected error for unknown command without handler")
	}
}

func TestDjinn_Status(t *testing.T) {
	d := NewDjinn()
	d.SetTokens(100, 50)
	d.SetTurns(3)
	d.SetActiveRole("executor")
	d.SetAgentCount(2)
	d.SetStreaming(true)

	s := d.Status()
	if s.Operation != "agent" {
		t.Errorf("Operation = %q", s.Operation)
	}
	if s.TokensIn != 100 || s.TokensOut != 50 {
		t.Errorf("Tokens = %d/%d", s.TokensIn, s.TokensOut)
	}
	if s.Turns != 3 {
		t.Errorf("Turns = %d", s.Turns)
	}
	if s.ActiveRole != "executor" {
		t.Errorf("ActiveRole = %q", s.ActiveRole)
	}
	if s.AgentCount != 2 {
		t.Errorf("AgentCount = %d", s.AgentCount)
	}
	if !s.IsStreaming {
		t.Error("should be streaming")
	}
	if s.AgentCap != 1 {
		t.Errorf("AgentCap = %d, want 1", s.AgentCap)
	}
	if s.ScopePath != "/" {
		t.Errorf("ScopePath = %q, want /", s.ScopePath)
	}
}

func TestDjinn_Subscribe_Emit(t *testing.T) {
	d := NewDjinn()
	ch := make(chan ViewEvent, 10)
	d.Subscribe(ch)

	d.Emit(ViewEvent{Kind: EventOutput, Text: "hello"})
	d.Emit(ViewEvent{Kind: EventToolCall, Tool: "Read"})

	if len(ch) != 2 {
		t.Fatalf("expected 2 events, got %d", len(ch))
	}

	ev1 := <-ch
	if ev1.Kind != EventOutput || ev1.Text != "hello" {
		t.Errorf("event 1: %+v", ev1)
	}

	ev2 := <-ch
	if ev2.Kind != EventToolCall || ev2.Tool != "Read" {
		t.Errorf("event 2: %+v", ev2)
	}
}

func TestDjinn_Unsubscribe(t *testing.T) {
	d := NewDjinn()
	ch := make(chan ViewEvent, 10)
	d.Subscribe(ch)
	d.Unsubscribe(ch)

	d.Emit(ViewEvent{Kind: EventOutput, Text: "should not arrive"})

	if len(ch) != 0 {
		t.Error("unsubscribed channel should not receive events")
	}
}

func TestDjinn_NavigateScope(t *testing.T) {
	d := NewDjinn()

	var gotPath, gotType string
	d.OnNavigate = func(path, scopeType string) error {
		gotPath = path
		gotType = scopeType
		return nil
	}

	err := d.NavigateScope("/aeon/djinn", "system")
	if err != nil {
		t.Fatal(err)
	}

	s := d.Status()
	if s.ScopePath != "/aeon/djinn" {
		t.Errorf("ScopePath = %q", s.ScopePath)
	}
	if s.ScopeType != "system" {
		t.Errorf("ScopeType = %q", s.ScopeType)
	}
	if gotPath != "/aeon/djinn" || gotType != "system" {
		t.Errorf("handler got %q %q", gotPath, gotType)
	}
}

func TestDjinn_Introspect(t *testing.T) {
	d := NewDjinn()
	_ = d.Start(context.Background())
	d.SetTokens(500, 200)

	report := d.Introspect()
	if report.Operation != "agent" {
		t.Errorf("Operation = %q", report.Operation)
	}
	if report.TokensIn != 500 {
		t.Errorf("TokensIn = %d", report.TokensIn)
	}
	if report.Uptime <= 0 {
		t.Error("Uptime should be positive after Start")
	}
}

func TestDjinn_Lifecycle(t *testing.T) {
	d := NewDjinn()
	ch := make(chan ViewEvent, 10)
	d.Subscribe(ch)

	_ = d.Start(context.Background())
	d.Stop()

	// Stop should emit EventDone.
	if len(ch) == 0 {
		t.Fatal("Stop should emit EventDone")
	}
	ev := <-ch
	if ev.Kind != EventDone {
		t.Errorf("Stop event kind = %s, want done", ev.Kind)
	}
}
