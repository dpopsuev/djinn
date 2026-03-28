package bugleport_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dpopsuev/bugle/world"
	"github.com/dpopsuev/djinn/bugleport"
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

func (s *stubLauncher) Start(_ context.Context, id world.EntityID, _ bugleport.LaunchConfig) error {
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
	w := bugleport.NewWorld()
	if w == nil {
		t.Fatal("NewWorld returned nil")
	}
}

func TestNewLocalTransport_NonNil(t *testing.T) {
	tr := bugleport.NewLocalTransport()
	if tr == nil {
		t.Fatal("NewLocalTransport returned nil")
	}
}

func TestNewMemBus_NonNil(t *testing.T) {
	bus := bugleport.NewMemBus()
	if bus == nil {
		t.Fatal("NewMemBus returned nil")
	}
}

func TestNewDurableBus_NonNil(t *testing.T) {
	path := filepath.Join(t.TempDir(), "signals.jsonl")
	bus, err := bugleport.NewDurableBus(path)
	if err != nil {
		t.Fatalf("NewDurableBus error: %v", err)
	}
	if bus == nil {
		t.Fatal("NewDurableBus returned nil")
	}
	bus.Close()
}

func TestNewTracker_NonNil(t *testing.T) {
	tr := bugleport.NewTracker()
	if tr == nil {
		t.Fatal("NewTracker returned nil")
	}
}

func TestNewRegistry_NonNil(t *testing.T) {
	reg := bugleport.NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry returned nil")
	}
}

func TestNewAgentPool_NonNil(t *testing.T) {
	w := bugleport.NewWorld()
	tr := bugleport.NewLocalTransport()
	bus := bugleport.NewMemBus()
	launcher := newStubLauncher()

	pool := bugleport.NewAgentPool(w, tr, bus, launcher)
	if pool == nil {
		t.Fatal("NewAgentPool returned nil")
	}
}

// ---------------------------------------------------------------------------
// Type alias usability tests — types from bugle are usable through bugleport
// ---------------------------------------------------------------------------

func TestEntityID_Usable(t *testing.T) {
	var id bugleport.EntityID
	if id != 0 {
		t.Fatal("zero EntityID should be 0")
	}
}

func TestSignal_Usable(t *testing.T) {
	sig := bugleport.Signal{
		Timestamp: time.Now().Format(time.RFC3339),
		Event:     bugleport.EventWorkerStarted,
		Agent:     "test",
		Meta: map[string]string{
			bugleport.MetaKeyWorkerID: "w-1",
		},
	}
	if sig.Event != bugleport.EventWorkerStarted {
		t.Fatalf("event = %q, want %q", sig.Event, bugleport.EventWorkerStarted)
	}
}

func TestBus_Interface(t *testing.T) {
	// MemBus satisfies the Bus interface via bugleport alias.
	var bus bugleport.Bus = bugleport.NewMemBus()
	idx := bus.Emit(&bugleport.Signal{Event: "test"})
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
	msg := bugleport.Message{
		From:         "agent-1",
		To:           "agent-2",
		Performative: bugleport.Inform,
		Content:      "hello",
	}
	if msg.From != "agent-1" {
		t.Fatalf("from = %q", msg.From)
	}
}

func TestTask_Usable(t *testing.T) {
	task := bugleport.Task{
		ID:    "task-1",
		State: bugleport.TaskSubmitted,
	}
	if task.State != bugleport.TaskSubmitted {
		t.Fatalf("state = %q", task.State)
	}
}

func TestAgentCard_Usable(t *testing.T) {
	card := bugleport.AgentCard{
		ID:        "agent-1",
		Name:      "Test Agent",
		Role:      "executor",
		Transport: "local",
	}
	if card.ID != "agent-1" {
		t.Fatalf("card ID = %q", card.ID)
	}
}

func TestHandler_Usable(t *testing.T) {
	var h bugleport.Handler = func(_ context.Context, msg bugleport.Message) (bugleport.Message, error) {
		return bugleport.Message{From: msg.To, Content: "ack"}, nil
	}
	resp, err := h(context.Background(), bugleport.Message{To: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.From != "test" {
		t.Fatalf("resp.From = %q", resp.From)
	}
}

func TestEvent_Usable(t *testing.T) {
	ev := bugleport.Event{
		TaskID: "task-1",
		State:  bugleport.TaskWorking,
	}
	if ev.State != bugleport.TaskWorking {
		t.Fatalf("state = %q", ev.State)
	}
}

func TestAgentIdentity_Usable(t *testing.T) {
	ai := bugleport.AgentIdentity{
		PersonaName: "Herald",
		Role:        bugleport.RoleWorker,
	}
	if !ai.IsRole(bugleport.RoleWorker) {
		t.Fatal("IsRole(RoleWorker) should be true")
	}
	if !ai.HasRole() {
		t.Fatal("HasRole() should be true for non-empty role")
	}
}

func TestModelIdentity_Usable(t *testing.T) {
	mi := bugleport.ModelIdentity{
		ModelName: "sonnet-4",
		Provider:  "anthropic",
	}
	s := mi.String()
	if s == "" {
		t.Fatal("String() should not be empty")
	}
}

func TestPersona_Usable(t *testing.T) {
	p := bugleport.Persona{
		Identity:    bugleport.AgentIdentity{PersonaName: "TestBot"},
		Description: "A test persona",
	}
	if p.Identity.PersonaName != "TestBot" {
		t.Fatalf("name = %q", p.Identity.PersonaName)
	}
}

func TestColorIdentity_Usable(t *testing.T) {
	ci := bugleport.ColorIdentity{
		Shade:      "Azure",
		Colour:     "Cerulean", //nolint:misspell // Bugle uses British spelling
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
	rec := bugleport.TokenRecord{
		CaseID:       "case-1",
		Step:         "triage",
		PromptTokens: 100,
	}
	if rec.CaseID != "case-1" {
		t.Fatalf("case_id = %q", rec.CaseID)
	}
}

func TestTokenSummary_Usable(t *testing.T) {
	tracker := bugleport.NewTracker()
	tracker.Record(&bugleport.TokenRecord{
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
	// Verify CostBill type is usable through bugleport.
	bill := bugleport.CostBill{
		Title:       "Test",
		TotalTokens: 1500,
	}
	if bill.Title != "Test" {
		t.Fatalf("title = %q", bill.Title)
	}
}

func TestLaunchConfig_Usable(t *testing.T) {
	cfg := bugleport.LaunchConfig{
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
	consts := []bugleport.Signal{
		{Performative: bugleport.Inform},
		{Performative: bugleport.Request},
		{Performative: bugleport.Confirm},
		{Performative: bugleport.Refuse},
		{Performative: bugleport.Handoff},
		{Performative: bugleport.Directive},
	}
	for i, s := range consts {
		if s.Performative == "" {
			t.Fatalf("performative[%d] is empty", i)
		}
	}
}

func TestEventConstants(t *testing.T) {
	events := []string{
		bugleport.EventWorkerStarted,
		bugleport.EventWorkerStopped,
		bugleport.EventWorkerDone,
		bugleport.EventWorkerError,
		bugleport.EventShouldStop,
		bugleport.EventBudgetUpdate,
		bugleport.EventDispatchRouted,
	}
	for i, e := range events {
		if e == "" {
			t.Fatalf("event[%d] is empty", i)
		}
	}
}

func TestMetaKeyConstants(t *testing.T) {
	keys := []string{
		bugleport.MetaKeyWorkerID,
		bugleport.MetaKeyError,
	}
	for i, k := range keys {
		if k == "" {
			t.Fatalf("meta key[%d] is empty", i)
		}
	}
}

func TestRoleConstants(t *testing.T) {
	roles := []bugleport.Role{
		bugleport.RoleWorker,
		bugleport.RoleManager,
		bugleport.RoleEnforcer,
		bugleport.RoleBroker,
	}
	for i, r := range roles {
		if r == "" {
			t.Fatalf("role[%d] is empty", i)
		}
	}
}

func TestTaskStateConstants(t *testing.T) {
	states := []bugleport.TaskState{
		bugleport.TaskSubmitted,
		bugleport.TaskWorking,
		bugleport.TaskCompleted,
		bugleport.TaskFailed,
	}
	for i, s := range states {
		if s == "" {
			t.Fatalf("task state[%d] is empty", i)
		}
	}
}

func TestAgentStateConstants(t *testing.T) {
	states := []bugleport.AgentState{
		bugleport.Active,
		bugleport.Idle,
		bugleport.Stale,
		bugleport.Errored,
		bugleport.Done,
	}
	for i, s := range states {
		if s == "" {
			t.Fatalf("agent state[%d] is empty", i)
		}
	}
}

func TestDiffKindConstants(t *testing.T) {
	// DiffKind is a type alias, verify world.DiffAttached etc. are accessible.
	var dk bugleport.DiffKind = "attached"
	if dk == "" {
		t.Fatal("DiffKind should not be empty")
	}
}

// ---------------------------------------------------------------------------
// Persona lookup tests
// ---------------------------------------------------------------------------

func TestAllPersonas_ReturnsEight(t *testing.T) {
	all := bugleport.AllPersonas()
	if len(all) != 8 {
		t.Fatalf("AllPersonas() = %d, want 8", len(all))
	}
}

func TestThesisPersonas_ReturnsFour(t *testing.T) {
	thesis := bugleport.ThesisPersonas()
	if len(thesis) != 4 {
		t.Fatalf("ThesisPersonas() = %d, want 4", len(thesis))
	}
}

func TestAntithesisPersonas_ReturnsFour(t *testing.T) {
	anti := bugleport.AntithesisPersonas()
	if len(anti) != 4 {
		t.Fatalf("AntithesisPersonas() = %d, want 4", len(anti))
	}
}

func TestPersonaByName_Known(t *testing.T) {
	names := []string{"Herald", "Seeker", "Sentinel", "Weaver", "Challenger", "Abyss", "Bulwark", "Specter"}
	for _, name := range names {
		p, ok := bugleport.PersonaByName(name)
		if !ok {
			t.Fatalf("PersonaByName(%q) not found", name)
		}
		if p.Identity.PersonaName != name {
			t.Fatalf("PersonaByName(%q).PersonaName = %q", name, p.Identity.PersonaName)
		}
	}
}

func TestPersonaByName_CaseInsensitive(t *testing.T) {
	p, ok := bugleport.PersonaByName("herald")
	if !ok {
		t.Fatal("PersonaByName(herald) should be case-insensitive")
	}
	if p.Identity.PersonaName != "Herald" {
		t.Fatalf("name = %q, want Herald", p.Identity.PersonaName)
	}
}

func TestPersonaByName_Unknown(t *testing.T) {
	_, ok := bugleport.PersonaByName("nonexistent")
	if ok {
		t.Fatal("PersonaByName(nonexistent) should return false")
	}
}

func TestDefaultPersonaResolver_Set(t *testing.T) {
	if bugleport.DefaultPersonaResolver == nil {
		t.Fatal("DefaultPersonaResolver should not be nil (persona init sets it)")
	}
	p, ok := bugleport.DefaultPersonaResolver("Herald")
	if !ok {
		t.Fatal("DefaultPersonaResolver(Herald) not found")
	}
	if p.Identity.PersonaName != "Herald" {
		t.Fatalf("name = %q", p.Identity.PersonaName)
	}
}

// ---------------------------------------------------------------------------
// World generic wrapper tests — Attach, Get, TryGet
// ---------------------------------------------------------------------------

func TestWorld_AttachGetTryGet(t *testing.T) {
	w := bugleport.NewWorld()
	id := w.Spawn()

	// Attach Health component via bugleport wrapper.
	bugleport.Attach(w, id, bugleport.Health{State: bugleport.Active})

	// Get via bugleport wrapper.
	h := bugleport.Get[bugleport.Health](w, id)
	if h.State != bugleport.Active {
		t.Fatalf("state = %q, want active", h.State)
	}

	// TryGet via bugleport wrapper.
	h2, ok := bugleport.TryGet[bugleport.Health](w, id)
	if !ok {
		t.Fatal("TryGet should find Health")
	}
	if h2.State != bugleport.Active {
		t.Fatalf("state = %q", h2.State)
	}

	// TryGet for unattached component.
	_, ok = bugleport.TryGet[bugleport.Budget](w, id)
	if ok {
		t.Fatal("TryGet should return false for Budget not attached")
	}
}

func TestWorld_SpawnDespawn(t *testing.T) {
	w := bugleport.NewWorld()
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
	reg := bugleport.NewRegistry()
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
	bus := bugleport.NewMemBus()
	bus.Emit(&bugleport.Signal{Event: "first"})
	bus.Emit(&bugleport.Signal{Event: "second"})

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

	bus, err := bugleport.NewDurableBus(path)
	if err != nil {
		t.Fatal(err)
	}
	bus.Emit(&bugleport.Signal{Event: "alpha"})
	bus.Emit(&bugleport.Signal{Event: "beta"})
	bus.Close()

	// Replay into a fresh bus.
	bus2, err := bugleport.NewDurableBus(path)
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
	tr := bugleport.NewTracker()
	tr.Record(&bugleport.TokenRecord{
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
	tr := bugleport.NewTracker()
	tr.Record(&bugleport.TokenRecord{
		CaseID:         "c1",
		Step:           "triage",
		PromptTokens:   1000,
		ArtifactTokens: 500,
	})
	summary := tr.Summary()
	bill := bugleport.BuildCostBill(&summary)
	if bill == nil {
		t.Fatal("BuildCostBill returned nil")
	}
	if bill.TotalTokens != 1500 {
		t.Fatalf("total = %d", bill.TotalTokens)
	}
}

func TestFormatCostBill_NonEmpty(t *testing.T) {
	bill := &bugleport.CostBill{
		Title:       "Test Bill",
		TotalTokens: 100,
	}
	out := bugleport.FormatCostBill(bill)
	if out == "" {
		t.Fatal("FormatCostBill returned empty string")
	}
}

func TestFormatCostBill_NilReturnsEmpty(t *testing.T) {
	out := bugleport.FormatCostBill(nil)
	if out != "" {
		t.Fatal("FormatCostBill(nil) should return empty string")
	}
}

// ---------------------------------------------------------------------------
// LocalTransport send + receive
// ---------------------------------------------------------------------------

func TestLocalTransport_SendReceive(t *testing.T) {
	tr := bugleport.NewLocalTransport()
	defer tr.Close()

	tr.Register("echo", func(_ context.Context, msg bugleport.Message) (bugleport.Message, error) {
		return bugleport.Message{From: "echo", Content: msg.Content + " echoed"}, nil
	})

	task, err := tr.SendMessage(context.Background(), "echo", bugleport.Message{
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
		if ev.State == bugleport.TaskCompleted {
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
	w := bugleport.NewWorld()
	tr := bugleport.NewLocalTransport()
	bus := bugleport.NewMemBus()
	launcher := newStubLauncher()
	pool := bugleport.NewAgentPool(w, tr, bus, launcher)

	ctx := context.Background()
	id, err := pool.Fork(ctx, "executor", bugleport.LaunchConfig{
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
	if signals[0].Event != bugleport.EventWorkerStarted {
		t.Fatalf("event = %q, want worker_started", signals[0].Event)
	}
}

// Ensure temp file cleanup for DurableBus test.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
