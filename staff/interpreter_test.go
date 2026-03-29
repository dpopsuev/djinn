package staff

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/signal"
)

// mockGenSec implements GenSecAgent for testing.
type mockGenSec struct {
	mu        sync.Mutex
	calls     []string
	responses map[string]string // category → JSON response
}

func newMockGenSec() *mockGenSec {
	return &mockGenSec{
		responses: map[string]string{
			signal.CategoryBudget:      `{"action":"throttle","reason":"budget at 85%, downshift","confidence":0.8}`,
			signal.CategoryDrift:       `{"action":"re_scope","reason":"structure violations detected","confidence":0.7}`,
			signal.CategoryPerformance: `{"action":"continue","reason":"transient spike","confidence":0.9}`,
		},
	}
}

func (m *mockGenSec) Ask(_ context.Context, content string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, content)

	// Match by category keyword in prompt.
	for cat, resp := range m.responses {
		if contains(content, cat) {
			return resp, nil
		}
	}
	return `{"action":"continue","reason":"default","confidence":0.5}`, nil
}

func (m *mockGenSec) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestGreenZoneNoLLMCalls(t *testing.T) {
	bus := signal.NewSignalBus()
	mock := newMockGenSec()
	interp := NewSignalInterpreter(bus, mock)
	interp.Start(context.Background())
	defer interp.Stop()

	// Green signal should NOT trigger GenSec.
	bus.Emit(signal.Signal{
		Workstream: "test",
		Level:      signal.Green,
		Category:   signal.CategoryBudget,
		Source:     "budget-watchdog",
		Message:    "budget healthy",
	})

	time.Sleep(50 * time.Millisecond)

	if mock.callCount() != 0 {
		t.Errorf("Green zone should cost zero LLM calls, got %d", mock.callCount())
	}
	if len(interp.AuditEntries()) != 0 {
		t.Errorf("Green zone should not produce audit entries, got %d", len(interp.AuditEntries()))
	}
}

func TestYellowZoneCallsGenSec(t *testing.T) {
	bus := signal.NewSignalBus()
	mock := newMockGenSec()
	interp := NewSignalInterpreter(bus, mock)
	interp.SetContextProvider(func() SignalContext {
		return SignalContext{
			TasksTotal: 5,
			TasksDone:  3,
			BudgetPct:  0.85,
			ActiveGear: GearE1,
		}
	})

	var notified bool
	interp.OnDecision(func(entry AuditEntry) {
		notified = true
	})

	interp.Start(context.Background())
	defer interp.Stop()

	bus.Emit(signal.Signal{
		Workstream: "test",
		Level:      signal.Yellow,
		Category:   signal.CategoryBudget,
		Source:     "budget-watchdog",
		Message:    "budget at 85%",
	})

	time.Sleep(50 * time.Millisecond)

	if mock.callCount() != 1 {
		t.Fatalf("Yellow zone should call GenSec once, got %d", mock.callCount())
	}

	entries := interp.AuditEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Zone != ZoneYellow {
		t.Errorf("expected ZoneYellow, got %s", entry.Zone)
	}
	if entry.Decision.Action != ActionThrottle {
		t.Errorf("expected ActionThrottle, got %s", entry.Decision.Action)
	}
	if entry.Decision.Pillar != signal.CategoryBudget {
		t.Errorf("expected pillar %q, got %q", signal.CategoryBudget, entry.Decision.Pillar)
	}
	if !notified {
		t.Error("OnDecision callback was not called")
	}
}

func TestBlackZoneEmitsCordon(t *testing.T) {
	bus := signal.NewSignalBus()
	mock := newMockGenSec()
	interp := NewSignalInterpreter(bus, mock)
	interp.Start(context.Background())
	defer interp.Stop()

	// Black = ZoneRed = operator must approve = cordon emitted.
	bus.Emit(signal.Signal{
		Workstream: "test",
		Level:      signal.Black,
		Category:   signal.CategoryDrift,
		Source:     "drift-watchdog",
		Message:    "unrecoverable failure",
	})

	time.Sleep(50 * time.Millisecond)

	if mock.callCount() != 1 {
		t.Fatalf("ZoneRed should call GenSec, got %d calls", mock.callCount())
	}

	// Check bus for cordon signal emitted by interpreter.
	var cordonFound bool
	for _, s := range bus.Signals() {
		if s.Source == "signal-interpreter" && s.Level == signal.Red {
			cordonFound = true
			break
		}
	}
	if !cordonFound {
		t.Error("ZoneRed (Black signal) should emit cordon signal on bus")
	}
}

func TestOrangeZoneAppliesDecision(t *testing.T) {
	bus := signal.NewSignalBus()
	mock := newMockGenSec()
	interp := NewSignalInterpreter(bus, mock)
	interp.Start(context.Background())
	defer interp.Stop()

	// Red signal → ZoneOrange → apply decision + notify operator.
	bus.Emit(signal.Signal{
		Workstream: "test",
		Level:      signal.Red,
		Category:   signal.CategoryDrift,
		Source:     "drift-watchdog",
		Message:    "structure score critical",
	})

	time.Sleep(50 * time.Millisecond)

	if mock.callCount() != 1 {
		t.Fatalf("ZoneOrange should call GenSec, got %d calls", mock.callCount())
	}

	entries := interp.AuditEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	if entries[0].Zone != ZoneOrange {
		t.Errorf("expected ZoneOrange, got %s", entries[0].Zone)
	}
}

func TestAuditLogRecordsFullChain(t *testing.T) {
	bus := signal.NewSignalBus()
	mock := newMockGenSec()
	interp := NewSignalInterpreter(bus, mock)
	interp.Start(context.Background())
	defer interp.Stop()

	bus.Emit(signal.Signal{
		Workstream: "test",
		Level:      signal.Yellow,
		Category:   signal.CategoryPerformance,
		Source:     "bottleneck-detector",
		Message:    "tool latency spike",
	})

	time.Sleep(50 * time.Millisecond)

	entries := interp.AuditEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}

	entry := entries[0]
	// Verify full chain is recorded.
	if entry.Signal.Category != signal.CategoryPerformance {
		t.Errorf("signal category: got %q, want %q", entry.Signal.Category, signal.CategoryPerformance)
	}
	if entry.Decision.Action != ActionContinue {
		t.Errorf("decision action: got %q, want %q", entry.Decision.Action, ActionContinue)
	}
	if entry.Decision.Confidence != 0.9 {
		t.Errorf("decision confidence: got %f, want 0.9", entry.Decision.Confidence)
	}
	if entry.Timestamp.IsZero() {
		t.Error("audit entry should have a timestamp")
	}
}

func TestDecisionParsing(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		wantAction Action
		wantErr    bool
	}{
		{
			name:       "valid JSON",
			response:   `{"action":"abort","reason":"critical","confidence":0.95}`,
			wantAction: ActionAbort,
		},
		{
			name:       "with markdown fences",
			response:   "```json\n{\"action\":\"skip\",\"reason\":\"low value\",\"confidence\":0.6}\n```",
			wantAction: ActionSkip,
		},
		{
			name:       "invalid JSON falls back to continue",
			response:   "I think we should continue because...",
			wantAction: ActionContinue,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := ParseDecision(tt.response, "test")
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDecision() error = %v, wantErr %v", err, tt.wantErr)
			}
			if d.Action != tt.wantAction {
				t.Errorf("action = %q, want %q", d.Action, tt.wantAction)
			}
		})
	}
}

func TestZoneFromLevel(t *testing.T) {
	tests := []struct {
		level signal.FlagLevel
		want  Zone
	}{
		{signal.Green, ZoneGreen},
		{signal.Yellow, ZoneYellow},
		{signal.Red, ZoneOrange},
		{signal.Black, ZoneRed},
	}
	for _, tt := range tests {
		got := ZoneFromLevel(tt.level)
		if got != tt.want {
			t.Errorf("ZoneFromLevel(%s) = %s, want %s", tt.level, got, tt.want)
		}
	}
}

func TestFormatSignalPrompt(t *testing.T) {
	s := signal.Signal{
		Category: signal.CategoryBudget,
		Level:    signal.Yellow,
		Source:   "budget-watchdog",
		Message:  "budget at 80%",
	}
	ctx := SignalContext{
		TasksTotal: 10,
		TasksDone:  7,
		BudgetPct:  0.80,
		ActiveGear: GearE2,
	}

	prompt := FormatSignalPrompt(s, ctx)

	// Verify prompt contains key elements.
	for _, want := range []string{"budget", "80%", "7/10", "E2", "continue", "skip", "throttle", "abort", "JSON"} {
		if !contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestActionJSONRoundTrip(t *testing.T) {
	d := Decision{
		Action:     ActionReScope,
		Reason:     "too broad",
		Confidence: 0.75,
		Pillar:     signal.CategoryDrift,
	}

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}

	var got Decision
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	if got.Action != d.Action || got.Reason != d.Reason || got.Confidence != d.Confidence {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, d)
	}
}
