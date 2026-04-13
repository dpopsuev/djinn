package execution

import "testing"

func TestDefaultSlotTable_HasGenSec(t *testing.T) {
	table := DefaultSlotTable()
	if len(table.Slots) != 1 {
		t.Fatalf("slots = %d, want 1", len(table.Slots))
	}
	if table.Slots[0].Role != "gensec" {
		t.Fatalf("role = %q, want gensec", table.Slots[0].Role)
	}
	if table.Slots[0].Priority != 100 {
		t.Fatalf("priority = %d, want 100", table.Slots[0].Priority)
	}
	if table.Capacity != 1 {
		t.Fatalf("capacity = %d, want 1", table.Capacity)
	}
}

func TestSlotTable_ForRole(t *testing.T) {
	table := &SlotTable{
		Slots: []AgentSlot{
			{Role: "coder", Priority: 50},
			{Role: "reviewer", Priority: 30},
		},
	}

	s := table.ForRole("coder")
	if s == nil {
		t.Fatal("ForRole(coder) returned nil")
	}
	if s.Priority != 50 {
		t.Fatalf("priority = %d, want 50", s.Priority)
	}

	if table.ForRole("nonexistent") != nil {
		t.Fatal("ForRole(nonexistent) should return nil")
	}
}

func TestSlotTable_KillOrder(t *testing.T) {
	table := &SlotTable{
		Slots: []AgentSlot{
			{Role: "gensec", Priority: 100},
			{Role: "coder", Priority: 50},
			{Role: "reviewer", Priority: 30},
			{Role: "linter", Priority: 10},
		},
	}

	// Threshold 100: kill everyone except gensec.
	order := table.KillOrder(100)
	if len(order) != 3 {
		t.Fatalf("killable = %d, want 3", len(order))
	}
	// Lowest priority first.
	if order[0] != "linter" {
		t.Fatalf("first to kill = %q, want linter", order[0])
	}
	if order[1] != "reviewer" {
		t.Fatalf("second to kill = %q, want reviewer", order[1])
	}
	if order[2] != "coder" {
		t.Fatalf("third to kill = %q, want coder", order[2])
	}

	// Threshold 50: only kill < 50.
	order = table.KillOrder(50)
	if len(order) != 2 {
		t.Fatalf("killable = %d, want 2", len(order))
	}
	if order[0] != "linter" || order[1] != "reviewer" {
		t.Fatalf("kill order = %v, want [linter, reviewer]", order)
	}

	// Threshold 0: nothing killable.
	order = table.KillOrder(0)
	if len(order) != 0 {
		t.Fatalf("killable = %d, want 0", len(order))
	}
}

func TestSlotTable_SlotsForSignal(t *testing.T) {
	table := &SlotTable{
		Slots: []AgentSlot{
			{Role: "coder", SpawnOn: []string{"tasks_planned"}},
			{Role: "reviewer", SpawnOn: []string{"gate_passed"}},
			{Role: "gensec", SpawnOn: []string{}}, // manual only
		},
	}

	slots := table.SlotsForSignal("tasks_planned")
	if len(slots) != 1 || slots[0].Role != "coder" {
		t.Fatalf("SlotsForSignal(tasks_planned) = %v, want [coder]", slots)
	}

	slots = table.SlotsForSignal("gate_passed")
	if len(slots) != 1 || slots[0].Role != "reviewer" {
		t.Fatalf("SlotsForSignal(gate_passed) = %v, want [reviewer]", slots)
	}

	slots = table.SlotsForSignal("unknown")
	if len(slots) != 0 {
		t.Fatalf("SlotsForSignal(unknown) = %v, want []", slots)
	}
}

func TestSlotTable_Roles(t *testing.T) {
	table := &SlotTable{
		Slots: []AgentSlot{
			{Role: "gensec", Priority: 100},
			{Role: "coder", Priority: 50},
		},
	}

	roles := table.Roles()
	if len(roles) != 2 || roles[0] != "gensec" || roles[1] != "coder" {
		t.Fatalf("Roles() = %v, want [gensec, coder]", roles)
	}
}

func TestSlotTable_CustomYAMLConfig(t *testing.T) {
	// Simulates what an operator would write in YAML.
	table := &SlotTable{
		Capacity: 3,
		Slots: []AgentSlot{
			{
				Role:         "gensec",
				Model:        "opus",
				Priority:     100,
				Capabilities: []string{"WorkTracking", "SignalBroadcasting"},
			},
			{
				Role:         "coder",
				Model:        "sonnet",
				Priority:     50,
				Capabilities: []string{"FileEditing", "ShellExecution"},
				Budget:       SlotBudget{MaxTokens: 200000, MaxCost: 1.0},
				SpawnOn:      []string{"tasks_planned"},
			},
			{
				Role:         "reviewer",
				Model:        "haiku",
				Priority:     30,
				Capabilities: []string{"FileEditing", "QualityGating"},
				Budget:       SlotBudget{MaxTokens: 100000, MaxCost: 0.5},
				SpawnOn:      []string{"gate_passed"},
			},
		},
	}

	if table.Capacity != 3 {
		t.Fatalf("capacity = %d, want 3", table.Capacity)
	}
	if len(table.Slots) != 3 {
		t.Fatalf("slots = %d, want 3", len(table.Slots))
	}

	// Gensec is never killed.
	killOrder := table.KillOrder(100)
	for _, r := range killOrder {
		if r == "gensec" {
			t.Fatal("gensec should never be in kill order (priority 100)")
		}
	}

	// Coder spawns on tasks_planned.
	coderSlots := table.SlotsForSignal("tasks_planned")
	if len(coderSlots) != 1 || coderSlots[0].Model != "sonnet" {
		t.Fatalf("coder slot model = %v", coderSlots)
	}
}
