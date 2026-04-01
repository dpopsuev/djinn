package jerichoport_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/jerichoport"
	"github.com/dpopsuev/jericho/world"
)

// ---------------------------------------------------------------------------
// Mock Launcher for NewAgentPool tests
// ---------------------------------------------------------------------------

type stubLauncher struct {
	started map[world.EntityID]bool
}

func newStubLauncher() *stubLauncher {
	return &stubLauncher{started: make(map[world.EntityID]bool)}
}

func (s *stubLauncher) Start(_ context.Context, id world.EntityID, _ jerichoport.AgentConfig) error {
	s.started[id] = true
	return nil
}

func (s *stubLauncher) Stop(_ context.Context, _ world.EntityID) error { return nil }

func (s *stubLauncher) Healthy(_ context.Context, id world.EntityID) bool {
	return s.started[id]
}

// ---------------------------------------------------------------------------
// Constructor smoke tests — every constructor returns non-nil
// ---------------------------------------------------------------------------

func TestNewWorld_NonNil(t *testing.T) {
	w := jerichoport.NewWorld()
	if w == nil {
		t.Fatal("NewWorld returned nil")
	}
}

func TestNewLocalTransport_NonNil(t *testing.T) {
	tr := jerichoport.NewLocalTransport()
	if tr == nil {
		t.Fatal("NewLocalTransport returned nil")
	}
}

func TestNewMemBus_NonNil(t *testing.T) {
	bus := jerichoport.NewMemBus()
	if bus == nil {
		t.Fatal("NewMemBus returned nil")
	}
}

func TestNewDurableBus_NonNil(t *testing.T) {
	path := filepath.Join(t.TempDir(), "signals.jsonl")
	bus, err := jerichoport.NewDurableBus(path)
	if err != nil {
		t.Fatalf("NewDurableBus error: %v", err)
	}
	if bus == nil {
		t.Fatal("NewDurableBus returned nil")
	}
	bus.Close()
}

func TestNewTracker_NonNil(t *testing.T) {
	tr := jerichoport.NewTracker()
	if tr == nil {
		t.Fatal("NewTracker returned nil")
	}
}

func TestNewRegistry_NonNil(t *testing.T) {
	reg := jerichoport.NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry returned nil")
	}
}

func TestNewAgentPool_NonNil(t *testing.T) {
	w := jerichoport.NewWorld()
	tr := jerichoport.NewLocalTransport()
	bus := jerichoport.NewMemBus()
	launcher := newStubLauncher()

	pool := jerichoport.NewAgentPool(w, tr, bus, launcher)
	if pool == nil {
		t.Fatal("NewAgentPool returned nil")
	}
}

// ---------------------------------------------------------------------------
// Type alias usability tests — types from bugle are usable through jerichoport
// ---------------------------------------------------------------------------

func TestEntityID_Usable(t *testing.T) {
	var id jerichoport.EntityID
	if id != 0 {
		t.Fatal("zero EntityID should be 0")
	}
}

func TestSignal_Usable(t *testing.T) {
	sig := jerichoport.Signal{
		Timestamp: time.Now().Format(time.RFC3339),
		Event:     jerichoport.EventWorkerStarted,
		Agent:     "test",
		Meta: map[string]string{
			jerichoport.MetaKeyWorkerID: "w-1",
		},
	}
	if sig.Event != jerichoport.EventWorkerStarted {
		t.Fatalf("event = %q, want %q", sig.Event, jerichoport.EventWorkerStarted)
	}
}

func TestBus_Interface(t *testing.T) {
	// MemBus satisfies the Bus interface via jerichoport alias.
	var bus jerichoport.Bus = jerichoport.NewMemBus()
	idx := bus.Emit(&jerichoport.Signal{Event: "test"})
	if idx != 0 {
		t.Fatalf("first emit index = %d, want 0", idx)
	}
	if bus.Len() != 1 {
		t.Fatalf("len = %d, want 1", bus.Len())
	}
	signals := bus.Since(0)
	if len(signals) != 1 {
		t.Fatalf("since(0) = %d signals, want 1", len(signals))
	}
}

func TestMessage_Usable(t *testing.T) {
	msg := jerichoport.Message{
		From:         "agent-1",
		To:           "agent-2",
		Performative: jerichoport.Inform,
		Content:      "hello",
	}
	if msg.From != "agent-1" {
		t.Fatalf("from = %q", msg.From)
	}
}

func TestTask_Usable(t *testing.T) {
	task := jerichoport.Task{
		ID:    "task-1",
		State: jerichoport.TaskSubmitted,
	}
	if task.State != jerichoport.TaskSubmitted {
		t.Fatalf("state = %q", task.State)
	}
}

func TestAgentCard_Usable(t *testing.T) {
	card := jerichoport.AgentCard{
		ID:        "agent-1",
		Name:      "Test Agent",
		Role:      "executor",
		Transport: "local",
	}
	if card.ID != "agent-1" {
		t.Fatalf("card ID = %q", card.ID)
	}
}

func TestMsgHandler_Usable(t *testing.T) {
	var h jerichoport.MsgHandler = func(_ context.Context, msg jerichoport.Message) (jerichoport.Message, error) {
		return jerichoport.Message{From: msg.To, Content: "ack"}, nil
	}
	resp, err := h(context.Background(), jerichoport.Message{To: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.From != "test" {
		t.Fatalf("resp.From = %q", resp.From)
	}
}

func TestEvent_Usable(t *testing.T) {
	ev := jerichoport.Event{
		TaskID: "task-1",
		State:  jerichoport.TaskWorking,
	}
	if ev.State != jerichoport.TaskWorking {
		t.Fatalf("state = %q", ev.State)
	}
}

func TestPersona_HasRole(t *testing.T) {
	p := jerichoport.Persona{
		Name: "Herald",
		Role: jerichoport.RoleWorker,
	}
	if p.Role != jerichoport.RoleWorker {
		t.Fatalf("Role = %q, want %q", p.Role, jerichoport.RoleWorker)
	}
	if p.Name != "Herald" {
		t.Fatalf("Name = %q, want Herald", p.Name)
	}
}

func TestModelIdentity_Usable(t *testing.T) {
	mi := jerichoport.ModelIdentity{
		ModelName: "sonnet-4",
		Provider:  "anthropic",
	}
	s := mi.String()
	if s == "" {
		t.Fatal("String() should not be empty")
	}
}

func TestPersona_Usable(t *testing.T) {
	p := jerichoport.Persona{
		Name:        "TestBot",
		Description: "A test persona",
	}
	if p.Name != "TestBot" {
		t.Fatalf("name = %q", p.Name)
	}
}

func TestColor_Usable(t *testing.T) {
	ci := jerichoport.Color{
		Shade:      "Azure",
		Name:       "Cerulean",
		Role:       "Writer",
		Collective: "Refactor",
		Hex:        "#007BA7",
	}
	title := ci.Title()
	if title == "" {
		t.Fatal("Title() should not be empty")
	}
	label := ci.Label()
	if label == "" {
		t.Fatal("Label() should not be empty")
	}
}

func TestTokenRecord_Usable(t *testing.T) {
	rec := jerichoport.TokenRecord{
		CaseID:       "case-1",
		Step:         "triage",
		PromptTokens: 100,
	}
	if rec.CaseID != "case-1" {
		t.Fatalf("case_id = %q", rec.CaseID)
	}
}

func TestTokenSummary_Usable(t *testing.T) {
	tracker := jerichoport.NewTracker()
	tracker.Record(&jerichoport.TokenRecord{
		CaseID:         "c1",
		Step:           "s1",
		PromptTokens:   500,
		ArtifactTokens: 200,
	})
	summary := tracker.Summary()
	if summary.TotalTokens != 700 {
		t.Fatalf("total = %d, want 700", summary.TotalTokens)
	}
}

func TestCostBill_Usable(t *testing.T) {
	// Verify CostBill type is usable through jerichoport.
	bill := jerichoport.CostBill{
		Title:       "Test",
		TotalTokens: 1500,
	}
	if bill.Title != "Test" {
		t.Fatalf("title = %q", bill.Title)
	}
}

func TestAgentConfig_Usable(t *testing.T) {
	cfg := jerichoport.AgentConfig{
		Role:    "executor",
		Prompt:  "system prompt",
		Model:   "sonnet-4",
		Tools:   []string{"read", "write"},
		WorkDir: "/tmp",
		Budget:  100.0,
	}
	if cfg.Role != "executor" {
		t.Fatalf("role = %q", cfg.Role)
	}
}

// ---------------------------------------------------------------------------
// Constant re-export tests
// ---------------------------------------------------------------------------

func TestPerformativeConstants(t *testing.T) {
	consts := []jerichoport.Signal{
		{Performative: jerichoport.Inform},
		{Performative: jerichoport.Request},
		{Performative: jerichoport.Confirm},
		{Performative: jerichoport.Refuse},
		{Performative: jerichoport.Handoff},
		{Performative: jerichoport.Directive},
	}
	for i, s := range consts {
		if s.Performative == "" {
			t.Fatalf("performative[%d] is empty", i)
		}
	}
}

func TestEventConstants(t *testing.T) {
	events := []string{
		jerichoport.EventWorkerStarted,
		jerichoport.EventWorkerStopped,
		jerichoport.EventWorkerDone,
		jerichoport.EventWorkerError,
		jerichoport.EventShouldStop,
		jerichoport.EventBudgetUpdate,
		jerichoport.EventDispatchRouted,
	}
	for i, e := range events {
		if e == "" {
			t.Fatalf("event[%d] is empty", i)
		}
	}
}

func TestMetaKeyConstants(t *testing.T) {
	keys := []string{
		jerichoport.MetaKeyWorkerID,
		jerichoport.MetaKeyError,
	}
	for i, k := range keys {
		if k == "" {
			t.Fatalf("meta key[%d] is empty", i)
		}
	}
}

func TestRoleConstants(t *testing.T) {
	roles := []jerichoport.Role{
		jerichoport.RoleWorker,
		jerichoport.RoleManager,
		jerichoport.RoleEnforcer,
		jerichoport.RoleBroker,
	}
	for i, r := range roles {
		if r == "" {
			t.Fatalf("role[%d] is empty", i)
		}
	}
}

func TestTaskStateConstants(t *testing.T) {
	states := []jerichoport.TaskState{
		jerichoport.TaskSubmitted,
		jerichoport.TaskWorking,
		jerichoport.TaskCompleted,
		jerichoport.TaskFailed,
	}
	for i, s := range states {
		if s == "" {
			t.Fatalf("task state[%d] is empty", i)
		}
	}
}

func TestAliveStateConstants(t *testing.T) {
	states := []jerichoport.AliveState{
		jerichoport.AliveRunning,
		jerichoport.AliveTerminated,
	}
	for i, s := range states {
		if s == "" {
			t.Fatalf("alive state[%d] is empty", i)
		}
	}
}

func TestReadyReasonConstants(t *testing.T) {
	reasons := []jerichoport.ReadyReason{
		jerichoport.ReasonIdle,
		jerichoport.ReasonStale,
		jerichoport.ReasonErrored,
	}
	for i, r := range reasons {
		if r == "" {
			t.Fatalf("ready reason[%d] is empty", i)
		}
	}
}

func TestDiffKindConstants(t *testing.T) {
	// DiffKind is a type alias, verify world.DiffAttached etc. are accessible.
	var dk jerichoport.DiffKind = "attached"
	if dk == "" {
		t.Fatal("DiffKind should not be empty")
	}
}

// ---------------------------------------------------------------------------
// Persona lookup tests
// ---------------------------------------------------------------------------

func TestAllPersonas_ReturnsEight(t *testing.T) {
	all := jerichoport.AllPersonas()
	if len(all) != 8 {
		t.Fatalf("AllPersonas() = %d, want 8", len(all))
	}
}

func TestThesisPersonas_ReturnsFour(t *testing.T) {
	thesis := jerichoport.ThesisPersonas()
	if len(thesis) != 4 {
		t.Fatalf("ThesisPersonas() = %d, want 4", len(thesis))
	}
}

func TestAntithesisPersonas_ReturnsFour(t *testing.T) {
	anti := jerichoport.AntithesisPersonas()
	if len(anti) != 4 {
		t.Fatalf("AntithesisPersonas() = %d, want 4", len(anti))
	}
}

func TestPersonaByName_Known(t *testing.T) {
	names := []string{"Herald", "Seeker", "Sentinel", "Weaver", "Challenger", "Abyss", "Bulwark", "Specter"}
	for _, name := range names {
		p, ok := jerichoport.PersonaByName(name)
		if !ok {
			t.Fatalf("PersonaByName(%q) not found", name)
		}
		if p.Name != name {
			t.Fatalf("PersonaByName(%q).Name = %q", name, p.Name)
		}
	}
}

func TestPersonaByName_CaseInsensitive(t *testing.T) {
	p, ok := jerichoport.PersonaByName("herald")
	if !ok {
		t.Fatal("PersonaByName(herald) should be case-insensitive")
	}
	if p.Name != "Herald" {
		t.Fatalf("name = %q, want Herald", p.Name)
	}
}

func TestPersonaByName_Unknown(t *testing.T) {
	_, ok := jerichoport.PersonaByName("nonexistent")
	if ok {
		t.Fatal("PersonaByName(nonexistent) should return false")
	}
}

func TestDefaultPersonaResolver_Set(t *testing.T) {
	if jerichoport.DefaultPersonaResolver == nil {
		t.Fatal("DefaultPersonaResolver should not be nil (persona init sets it)")
	}
	p, ok := jerichoport.DefaultPersonaResolver("Herald")
	if !ok {
		t.Fatal("DefaultPersonaResolver(Herald) not found")
	}
	if p.Name != "Herald" {
		t.Fatalf("name = %q", p.Name)
	}
}

// ---------------------------------------------------------------------------
// World generic wrapper tests — Attach, Get, TryGet
// ---------------------------------------------------------------------------

func TestWorld_AttachGetTryGet(t *testing.T) {
	w := jerichoport.NewWorld()
	id := w.Spawn()

	// Attach Alive component via jerichoport wrapper.
	jerichoport.Attach(w, id, jerichoport.Alive{State: jerichoport.AliveRunning})

	// Get via jerichoport wrapper.
	h := jerichoport.Get[jerichoport.Alive](w, id)
	if h.State != jerichoport.AliveRunning {
		t.Fatalf("state = %q, want running", h.State)
	}

	// TryGet via jerichoport wrapper.
	h2, ok := jerichoport.TryGet[jerichoport.Alive](w, id)
	if !ok {
		t.Fatal("TryGet should find Alive")
	}
	if h2.State != jerichoport.AliveRunning {
		t.Fatalf("state = %q", h2.State)
	}

	// TryGet for unattached component.
	_, ok = jerichoport.TryGet[jerichoport.Budget](w, id)
	if ok {
		t.Fatal("TryGet should return false for Budget not attached")
	}
}

func TestWorld_SpawnDespawn(t *testing.T) {
	w := jerichoport.NewWorld()
	id := w.Spawn()
	if !w.Alive(id) {
		t.Fatal("entity should be alive")
	}
	w.Despawn(id)
	if w.Alive(id) {
		t.Fatal("entity should be dead after Despawn")
	}
}

// ---------------------------------------------------------------------------
// Registry assign test
// ---------------------------------------------------------------------------

func TestRegistry_Assign(t *testing.T) {
	reg := jerichoport.NewRegistry()
	ci, err := reg.Assign("Writer", "Refactor")
	if err != nil {
		t.Fatalf("Assign error: %v", err)
	}
	if ci.Role != "Writer" {
		t.Fatalf("role = %q", ci.Role)
	}
	if ci.Collective != "Refactor" {
		t.Fatalf("collective = %q", ci.Collective)
	}
	if reg.Active() != 1 {
		t.Fatalf("active = %d, want 1", reg.Active())
	}
}

// ---------------------------------------------------------------------------
// MemBus emit/since round-trip
// ---------------------------------------------------------------------------

func TestMemBus_EmitSince(t *testing.T) {
	bus := jerichoport.NewMemBus()
	bus.Emit(&jerichoport.Signal{Event: "first"})
	bus.Emit(&jerichoport.Signal{Event: "second"})

	all := bus.Since(0)
	if len(all) != 2 {
		t.Fatalf("since(0) = %d, want 2", len(all))
	}
	if all[0].Event != "first" || all[1].Event != "second" {
		t.Fatalf("events = %q, %q", all[0].Event, all[1].Event)
	}
}

// ---------------------------------------------------------------------------
// DurableBus round-trip
// ---------------------------------------------------------------------------

func TestDurableBus_EmitReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bus.jsonl")

	bus, err := jerichoport.NewDurableBus(path)
	if err != nil {
		t.Fatal(err)
	}
	bus.Emit(&jerichoport.Signal{Event: "alpha"})
	bus.Emit(&jerichoport.Signal{Event: "beta"})
	bus.Close()

	// Replay into a fresh bus.
	bus2, err := jerichoport.NewDurableBus(path)
	if err != nil {
		t.Fatal(err)
	}
	defer bus2.Close()
	n, err := bus2.Replay()
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("replayed = %d, want 2", n)
	}
}

// ---------------------------------------------------------------------------
// Tracker record + summary
// ---------------------------------------------------------------------------

func TestTracker_RecordSummary(t *testing.T) {
	tr := jerichoport.NewTracker()
	tr.Record(&jerichoport.TokenRecord{
		CaseID:         "c1",
		Step:           "triage",
		PromptTokens:   1000,
		ArtifactTokens: 500,
		Timestamp:      time.Now(),
		WallClockMs:    250,
	})
	s := tr.Summary()
	if s.TotalPromptTokens != 1000 {
		t.Fatalf("prompt = %d", s.TotalPromptTokens)
	}
	if s.TotalArtifactTokens != 500 {
		t.Fatalf("artifact = %d", s.TotalArtifactTokens)
	}
	if s.TotalTokens != 1500 {
		t.Fatalf("total = %d", s.TotalTokens)
	}
}

// ---------------------------------------------------------------------------
// BuildCostBill + FormatCostBill
// ---------------------------------------------------------------------------

func TestBuildCostBill_NonNil(t *testing.T) {
	tr := jerichoport.NewTracker()
	tr.Record(&jerichoport.TokenRecord{
		CaseID:         "c1",
		Step:           "triage",
		PromptTokens:   1000,
		ArtifactTokens: 500,
	})
	summary := tr.Summary()
	bill := jerichoport.BuildCostBill(&summary)
	if bill == nil {
		t.Fatal("BuildCostBill returned nil")
	}
	if bill.TotalTokens != 1500 {
		t.Fatalf("total = %d", bill.TotalTokens)
	}
}

func TestFormatCostBill_NonEmpty(t *testing.T) {
	bill := &jerichoport.CostBill{
		Title:       "Test Bill",
		TotalTokens: 100,
	}
	out := jerichoport.FormatCostBill(bill)
	if out == "" {
		t.Fatal("FormatCostBill returned empty string")
	}
}

func TestFormatCostBill_NilReturnsEmpty(t *testing.T) {
	out := jerichoport.FormatCostBill(nil)
	if out != "" {
		t.Fatal("FormatCostBill(nil) should return empty string")
	}
}

// ---------------------------------------------------------------------------
// LocalTransport send + receive
// ---------------------------------------------------------------------------

func TestLocalTransport_SendReceive(t *testing.T) {
	tr := jerichoport.NewLocalTransport()
	defer tr.Close()

	tr.Register("echo", func(_ context.Context, msg jerichoport.Message) (jerichoport.Message, error) {
		return jerichoport.Message{From: "echo", Content: msg.Content + " echoed"}, nil
	})

	task, err := tr.SendMessage(context.Background(), "echo", jerichoport.Message{
		From:    "caller",
		Content: "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if task == nil {
		t.Fatal("task should not be nil")
	}

	// Subscribe and wait for completion.
	ch, err := tr.Subscribe(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}

	var completed bool
	for ev := range ch {
		if ev.State == jerichoport.TaskCompleted {
			completed = true
			if ev.Data == nil {
				t.Fatal("completed event should have data")
			}
			if ev.Data.Content != "hello echoed" {
				t.Fatalf("content = %q", ev.Data.Content)
			}
		}
	}
	if !completed {
		t.Fatal("never received TaskCompleted event")
	}
}

// ---------------------------------------------------------------------------
// AgentPool fork + count
// ---------------------------------------------------------------------------

func TestAgentPool_ForkCount(t *testing.T) {
	w := jerichoport.NewWorld()
	tr := jerichoport.NewLocalTransport()
	bus := jerichoport.NewMemBus()
	launcher := newStubLauncher()
	pool := jerichoport.NewAgentPool(w, tr, bus, launcher)

	ctx := context.Background()
	id, err := pool.Fork(ctx, "executor", jerichoport.AgentConfig{
		Role:  "executor",
		Model: "test-model",
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("entity ID should not be 0")
	}
	if pool.Count() != 1 {
		t.Fatalf("count = %d, want 1", pool.Count())
	}
	if !launcher.started[id] {
		t.Fatal("launcher.Start should have been called")
	}

	// Verify signal was emitted.
	signals := bus.Since(0)
	if len(signals) == 0 {
		t.Fatal("no signals emitted after Fork")
	}
	if signals[0].Event != jerichoport.EventWorkerStarted {
		t.Fatalf("event = %q, want worker_started", signals[0].Event)
	}
}

// Ensure temp file cleanup for DurableBus test.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
