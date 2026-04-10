package daemon

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/artifact"
	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/djinn/tools"
)

// --- TSK-508: PlanHub → Artifact mediation integration tests ---

func TestPlanHub_AddSegment_Integration(t *testing.T) {
	// Wire real infrastructure: Graph + Ring + SignalBus.
	ring := telemetry.NewTraceProjection(100)
	bus := telemetry.NewSignalBus()
	spy := &spyDisplay{}
	core := HubCore{
		Tracer:  ring.For(telemetry.ComponentTool),
		Signals: bus,
		Display: spy,
	}
	graph := artifact.NewGraph("integration-test", artifact.DefaultRegistry())
	ph := NewPlanHub(core, graph)

	// Subscribe before adding the segment — verify signal delivery.
	var received []telemetry.Signal
	var mu sync.Mutex
	bus.OnSignal(func(s telemetry.Signal) {
		mu.Lock()
		received = append(received, s)
		mu.Unlock()
	})

	// Act: add a segment through PlanHub.
	id := ph.AddSegment(artifact.Artifact{Title: "implement auth"})

	// 1. Segment appears in graph.
	seg, err := graph.Get(id)
	if err != nil {
		t.Fatalf("graph.Get(%q) error: %v", id, err)
	}
	if seg.Title != "implement auth" {
		t.Errorf("segment title = %q, want %q", seg.Title, "implement auth")
	}
	if seg.Kind != artifact.KindPlanSegment {
		t.Errorf("segment kind = %q, want %q", seg.Kind, artifact.KindPlanSegment)
	}

	// 2. Trace ring has the event.
	events := ring.Last(10)
	foundTrace := false
	for _, e := range events {
		if e.Action == "segment-add" && e.Detail == "implement auth" {
			foundTrace = true
			break
		}
	}
	if !foundTrace {
		t.Error("trace ring missing 'segment-add' event")
	}

	// 3. Signal bus received emission (both stored and subscriber callback).
	signals := bus.Signals()
	if len(signals) == 0 {
		t.Fatal("signal bus has no signals after AddSegment")
	}
	if signals[0].Category != "plan" || signals[0].Level != telemetry.Green {
		t.Errorf("signal = {category:%q, level:%v}, want {plan, green}", signals[0].Category, signals[0].Level)
	}

	mu.Lock()
	subCount := len(received)
	mu.Unlock()
	if subCount == 0 {
		t.Error("subscriber callback was not invoked")
	}

	// 4. Display received a message.
	if len(spy.msgs) == 0 {
		t.Fatal("display received no messages")
	}
	pe, ok := spy.msgs[0].Content.(PlanEvent)
	if !ok {
		t.Fatalf("display content type = %T, want PlanEvent", spy.msgs[0].Content)
	}
	if pe.Action != "add" || pe.SegmentID != id {
		t.Errorf("PlanEvent = {action:%q, id:%q}, want {add, %q}", pe.Action, pe.SegmentID, id)
	}
}

func TestPlanHub_UpdateStatus_Integration(t *testing.T) {
	ring := telemetry.NewTraceProjection(100)
	bus := telemetry.NewSignalBus()
	core := HubCore{
		Tracer:  ring.For(telemetry.ComponentTool),
		Signals: bus,
		Display: NopDisplaySender{},
	}
	graph := artifact.NewGraph("status-test", artifact.DefaultRegistry())
	ph := NewPlanHub(core, graph)

	// Add segment via PlanHub (starts as draft).
	id := ph.AddSegment(artifact.Artifact{Title: "status lifecycle"})

	// Verify initial status is draft.
	seg, _ := graph.Get(id)
	if seg.Status != artifact.StatusDraft {
		t.Fatalf("initial status = %q, want %q", seg.Status, artifact.StatusDraft)
	}

	// Update status to ready via Graph.UpdateStatus (PlanHub delegates to Graph).
	if err := graph.UpdateStatus(id, artifact.StatusReady); err != nil {
		t.Fatalf("UpdateStatus to ready: %v", err)
	}

	// Verify graph reflects new status.
	seg, _ = graph.Get(id)
	if seg.Status != artifact.StatusReady {
		t.Errorf("status after update = %q, want %q", seg.Status, artifact.StatusReady)
	}

	// Claim and start via Graph, then complete via PlanHub.
	if err := graph.Claim(id, "executor-1"); err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if err := graph.Start(id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Complete via PlanHub — verifies mediation for the full lifecycle.
	if err := ph.Complete(id); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	seg, _ = graph.Get(id)
	if seg.Status != artifact.StatusComplete {
		t.Errorf("final status = %q, want %q", seg.Status, artifact.StatusComplete)
	}

	// Verify trace has complete event.
	events := ring.Last(20)
	foundComplete := false
	for _, e := range events {
		if e.Action == "segment-complete" {
			foundComplete = true
			break
		}
	}
	if !foundComplete {
		t.Error("trace ring missing 'segment-complete' event")
	}

	// Verify signals: at least AddSegment signal + Complete signal.
	signals := bus.Signals()
	if len(signals) < 2 { //nolint:mnd // add + complete
		t.Errorf("signal count = %d, want >= 2 (add + complete)", len(signals))
	}
}

func TestPlanHub_ClaimSegment_Integration(t *testing.T) {
	ring := telemetry.NewTraceProjection(100)
	bus := telemetry.NewSignalBus()
	spy := &spyDisplay{}
	core := HubCore{
		Tracer:  ring.For(telemetry.ComponentTool),
		Signals: bus,
		Display: spy,
	}
	graph := artifact.NewGraph("claim-test", artifact.DefaultRegistry())
	ph := NewPlanHub(core, graph)

	// Add via hub and move to ready for claiming.
	id := ph.AddSegment(artifact.Artifact{Title: "claimable segment"})
	if err := graph.UpdateStatus(id, artifact.StatusReady); err != nil {
		t.Fatalf("UpdateStatus to ready: %v", err)
	}

	// Claim via PlanHub.
	if err := ph.Claim(id, "executor-1"); err != nil {
		t.Fatalf("Claim: %v", err)
	}

	// Verify graph state.
	seg, _ := graph.Get(id)
	if seg.Status != artifact.StatusClaimed {
		t.Errorf("status = %q, want %q", seg.Status, artifact.StatusClaimed)
	}
	if seg.Owner != "executor-1" {
		t.Errorf("owner = %q, want %q", seg.Owner, "executor-1")
	}

	// Verify trace has claim event.
	events := ring.Last(20)
	foundClaim := false
	for _, e := range events {
		if e.Action == "segment-claim" {
			foundClaim = true
			break
		}
	}
	if !foundClaim {
		t.Error("trace ring missing 'segment-claim' event")
	}

	// Verify signal emitted for claim.
	signals := bus.Signals()
	foundClaimSignal := false
	for _, s := range signals {
		if s.Category == "plan" && s.Message == "segment claimed: "+id+" by executor-1" {
			foundClaimSignal = true
			break
		}
	}
	if !foundClaimSignal {
		t.Error("signal bus missing claim signal")
	}

	// Verify display has claim event.
	foundClaimDisplay := false
	for _, msg := range spy.msgs {
		if pe, ok := msg.Content.(PlanEvent); ok && pe.Action == "claim" {
			foundClaimDisplay = true
			break
		}
	}
	if !foundClaimDisplay {
		t.Error("display missing claim PlanEvent")
	}
}

func TestPlanHub_FullLifecycle_Integration(t *testing.T) {
	// End-to-end: add → ready → claim → start → complete.
	ring := telemetry.NewTraceProjection(100)
	bus := telemetry.NewSignalBus()
	core := HubCore{
		Tracer:  ring.For(telemetry.ComponentTool),
		Signals: bus,
		Display: NopDisplaySender{},
	}
	graph := artifact.NewGraph("lifecycle-test", artifact.DefaultRegistry())
	ph := NewPlanHub(core, graph)

	// 1. Add segment.
	id := ph.AddSegment(artifact.Artifact{Title: "full lifecycle"})

	// 2. Transition to ready.
	if err := graph.UpdateStatus(id, artifact.StatusReady); err != nil {
		t.Fatalf("to ready: %v", err)
	}

	// Verify segment appears in Ready() list.
	ready := ph.Ready()
	if len(ready) != 1 {
		t.Fatalf("Ready() = %d, want 1", len(ready))
	}
	if ready[0].ID != id {
		t.Errorf("Ready()[0].ID = %q, want %q", ready[0].ID, id)
	}

	// 3. Claim.
	if err := ph.Claim(id, "agent-1"); err != nil {
		t.Fatalf("Claim: %v", err)
	}

	// After claim, Ready() should be empty.
	if len(ph.Ready()) != 0 {
		t.Error("Ready() should be empty after claim")
	}

	// 4. Start.
	if err := graph.Start(id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// 5. Complete.
	if err := ph.Complete(id); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Verify final state.
	seg, _ := graph.Get(id)
	if seg.Status != artifact.StatusComplete {
		t.Errorf("final status = %q, want %q", seg.Status, artifact.StatusComplete)
	}

	// Verify all trace events recorded (add, claim, complete).
	events := ring.Last(50)
	actions := make(map[string]bool)
	for _, e := range events {
		actions[e.Action] = true
	}
	for _, want := range []string{"segment-add", "segment-claim", "segment-complete"} {
		if !actions[want] {
			t.Errorf("trace missing action %q", want)
		}
	}

	// Verify signals: add + claim + complete = at least 3.
	signals := bus.Signals()
	if len(signals) < 3 { //nolint:mnd // add + claim + complete
		t.Errorf("signal count = %d, want >= 3", len(signals))
	}
}

func TestPlanHub_MultipleSegments_Integration(t *testing.T) {
	ring := telemetry.NewTraceProjection(100)
	bus := telemetry.NewSignalBus()
	core := HubCore{
		Tracer:  ring.For(telemetry.ComponentTool),
		Signals: bus,
		Display: NopDisplaySender{},
	}
	graph := artifact.NewGraph("multi-segment", artifact.DefaultRegistry())
	ph := NewPlanHub(core, graph)

	// Add three segments.
	id1 := ph.AddSegment(artifact.Artifact{Title: "segment 1"})
	id2 := ph.AddSegment(artifact.Artifact{Title: "segment 2"})
	id3 := ph.AddSegment(artifact.Artifact{Title: "segment 3"})

	// All should be in graph.
	all := graph.All()
	if len(all) != 3 { //nolint:mnd // 3 segments
		t.Fatalf("graph.All() = %d, want 3", len(all))
	}

	// IDs should be unique.
	ids := map[string]bool{id1: true, id2: true, id3: true}
	if len(ids) != 3 { //nolint:mnd // 3 unique IDs
		t.Errorf("IDs not unique: %q, %q, %q", id1, id2, id3)
	}

	// Signal count should match segment count.
	signals := bus.Signals()
	if len(signals) != 3 { //nolint:mnd // one per segment
		t.Errorf("signal count = %d, want 3", len(signals))
	}
}

// --- TSK-509: ToolHub SLA breach integration tests ---

func TestToolHub_SLABreach_Integration(t *testing.T) {
	ring := telemetry.NewTraceProjection(100)
	bus := telemetry.NewSignalBus()
	core := HubCore{
		Tracer:  ring.For(telemetry.ComponentTool),
		Signals: bus,
		Display: NopDisplaySender{},
	}

	// Slow executor: 200ms delay breaches plan SLA (P95=50ms).
	executor := &slowExecutor{delay: 200 * time.Millisecond, result: "done"} //nolint:mnd // intentionally slow
	tracker := tools.NewToolLatencyTracker()
	th := NewToolHub(core, executor, tracker)

	// Subscribe to verify signals are delivered to subscribers.
	var breachSignals []telemetry.Signal
	var mu sync.Mutex
	bus.OnSignal(func(s telemetry.Signal) {
		if s.Level == telemetry.Yellow && s.Category == toolHubName {
			mu.Lock()
			breachSignals = append(breachSignals, s)
			mu.Unlock()
		}
	})

	// Execute enough times for P50/P95 to stabilize.
	const execCount = 3
	for range execCount {
		result, err := th.Execute(context.Background(), "plan", json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if result != "done" {
			t.Errorf("result = %q, want %q", result, "done")
		}
	}

	// Verify latency recorded in tracker.
	if tracker.Count("plan") != execCount {
		t.Errorf("tracker.Count(plan) = %d, want %d", tracker.Count("plan"), execCount)
	}

	// P50 and P95 should be >= 200ms (all samples are slow).
	p50 := tracker.P50("plan")
	p95 := tracker.P95("plan")
	if p50 < 200*time.Millisecond { //nolint:mnd // SLA threshold
		t.Errorf("P50 = %v, want >= 200ms", p50)
	}
	if p95 < 200*time.Millisecond { //nolint:mnd // SLA threshold
		t.Errorf("P95 = %v, want >= 200ms", p95)
	}

	// Verify SLA breach signals emitted.
	mu.Lock()
	breachCount := len(breachSignals)
	mu.Unlock()
	if breachCount == 0 {
		t.Error("no SLA breach signals emitted to subscriber")
	}

	// Also check stored signals.
	allSignals := bus.Signals()
	foundBreach := false
	for _, s := range allSignals {
		if s.Level == telemetry.Yellow && s.Category == toolHubName {
			foundBreach = true
			break
		}
	}
	if !foundBreach {
		t.Error("signal bus has no stored SLA breach signal")
	}

	// Verify trace ring recorded tool-exec events.
	events := ring.Last(50)
	toolExecCount := 0
	for _, e := range events {
		if e.Action == "tool-exec" || e.Action == "tool-exec_done" {
			toolExecCount++
		}
	}
	if toolExecCount == 0 {
		t.Error("trace ring missing tool-exec events")
	}
}

func TestToolHub_NoSLABreach_FastExecution(t *testing.T) {
	bus := telemetry.NewSignalBus()
	core := HubCore{
		Tracer:  telemetry.NewTraceProjection(100).For(telemetry.ComponentTool),
		Signals: bus,
		Display: NopDisplaySender{},
	}

	// Fast executor: no delay, well within SLA.
	executor := &stubExecutor{result: "ok"}
	tracker := tools.NewToolLatencyTracker()
	th := NewToolHub(core, executor, tracker)

	for range 3 { //nolint:mnd // warm up tracker
		th.Execute(context.Background(), "plan", json.RawMessage(`{}`)) //nolint:errcheck // test
	}

	// No SLA breach signals should exist.
	signals := bus.Signals()
	for _, s := range signals {
		if s.Level == telemetry.Yellow && s.Category == toolHubName {
			t.Errorf("unexpected SLA breach signal: %q", s.Message)
		}
	}
}

func TestToolHub_TracksMultipleTools(t *testing.T) {
	ring := telemetry.NewTraceProjection(100)
	bus := telemetry.NewSignalBus()
	core := HubCore{
		Tracer:  ring.For(telemetry.ComponentTool),
		Signals: bus,
		Display: NopDisplaySender{},
	}

	executor := &stubExecutor{result: "ok"}
	tracker := tools.NewToolLatencyTracker()
	th := NewToolHub(core, executor, tracker)

	// Execute different tools.
	th.Execute(context.Background(), "plan", json.RawMessage(`{}`)) //nolint:errcheck // test
	th.Execute(context.Background(), "test", json.RawMessage(`{}`)) //nolint:errcheck // test
	th.Execute(context.Background(), "git", json.RawMessage(`{}`))  //nolint:errcheck // test

	// Verify all three recorded.
	if tracker.Count("plan") != 1 {
		t.Errorf("tracker.Count(plan) = %d, want 1", tracker.Count("plan"))
	}
	if tracker.Count("test") != 1 {
		t.Errorf("tracker.Count(test) = %d, want 1", tracker.Count("test"))
	}
	if tracker.Count("git") != 1 {
		t.Errorf("tracker.Count(git) = %d, want 1", tracker.Count("git"))
	}

	// Verify trace ring has events for all three.
	events := ring.Last(50)
	toolNames := make(map[string]bool)
	for _, e := range events {
		if e.Action == "tool-exec" {
			toolNames[e.Detail] = true
		}
	}
	for _, want := range []string{"plan", "test", "git"} {
		if !toolNames[want] {
			t.Errorf("trace ring missing tool-exec event for %q", want)
		}
	}
}

// --- Cross-hub integration: HubRegistry with real hubs ---

func TestHubRegistry_RealHubs_Integration(t *testing.T) {
	ring := telemetry.NewTraceProjection(100)
	bus := telemetry.NewSignalBus()
	core := HubCore{
		Tracer:  ring.For(telemetry.ComponentTool),
		Signals: bus,
		Display: NopDisplaySender{},
	}

	// Create real hubs.
	graph := artifact.NewGraph("registry-test", artifact.DefaultRegistry())
	planHub := NewPlanHub(core, graph)
	toolHub := NewToolHub(core, &stubExecutor{result: "ok"}, tools.NewToolLatencyTracker())
	analysisHub := NewAnalysisHub(core, "/tmp/test")

	// Register all.
	reg := NewRegistry()
	reg.Register(planHub)
	reg.Register(toolHub)
	reg.Register(analysisHub)

	// Lookup by name.
	h, ok := reg.Get("plan")
	if !ok {
		t.Fatal("plan hub not found")
	}
	if h.Phase() != "plan" {
		t.Errorf("plan hub phase = %q, want %q", h.Phase(), "plan")
	}

	h, ok = reg.Get("tool")
	if !ok {
		t.Fatal("tool hub not found")
	}
	if h.Phase() != "execute" {
		t.Errorf("tool hub phase = %q, want %q", h.Phase(), "execute")
	}

	// Lookup by phase.
	execHubs := reg.ByPhase("execute")
	if len(execHubs) != 1 {
		t.Errorf("ByPhase(execute) = %d, want 1", len(execHubs))
	}

	// All names.
	names := reg.Names()
	if len(names) != 3 { //nolint:mnd // plan + tool + analysis
		t.Errorf("Names() = %v, want 3 entries", names)
	}

	// Operate through the plan hub retrieved from registry.
	planH, _ := reg.Get("plan")
	realPH, ok := planH.(*PlanHub)
	if !ok {
		t.Fatal("plan hub type assertion failed")
	}
	id := realPH.AddSegment(artifact.Artifact{Title: "via registry"})
	if id == "" {
		t.Error("AddSegment through registry-retrieved hub returned empty ID")
	}

	// Verify graph was mutated.
	seg, err := graph.Get(id)
	if err != nil {
		t.Fatalf("segment not in graph: %v", err)
	}
	if seg.Title != "via registry" {
		t.Errorf("title = %q, want %q", seg.Title, "via registry")
	}
}
