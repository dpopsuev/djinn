package quality

import (
	"testing"
	"time"
)

func TestWasteKind_AllConstants(t *testing.T) {
	t.Parallel()

	kinds := AllWasteKinds()
	if len(kinds) != 7 {
		t.Fatalf("expected 7 waste kinds, got %d", len(kinds))
	}

	// Verify all expected values are present.
	expected := map[WasteKind]bool{
		WasteOverproduction: true,
		WasteWaiting:        true,
		WasteTransportation: true,
		WasteOverProcessing: true,
		WasteInventory:      true,
		WasteMotion:         true,
		WasteDefect:         true,
	}

	for _, k := range kinds {
		if !expected[k] {
			t.Errorf("unexpected waste kind: %q", k)
		}
		delete(expected, k)
	}
	for k := range expected {
		t.Errorf("missing waste kind: %q", k)
	}

	// Verify string values are lowercase with hyphens (Lean naming convention).
	for _, k := range kinds {
		s := string(k)
		if s == "" {
			t.Error("empty waste kind string")
		}
		for _, c := range s {
			if c >= 'A' && c <= 'Z' {
				t.Errorf("waste kind %q contains uppercase", s)
			}
		}
	}
}

func TestWasteClassifier_DuplicateRead_Transportation(t *testing.T) {
	t.Parallel()

	wc := NewWasteClassifier(nil)

	// First read of a file — no waste.
	r := wc.ClassifyCall("Read", `{"file_path": "/home/user/foo.go"}`, "file contents", false, 10*time.Millisecond)
	if r != nil {
		t.Fatalf("first read should not be waste, got %+v", r)
	}

	// Second read of same file — transportation waste.
	r = wc.ClassifyCall("Read", `{"file_path": "/home/user/foo.go"}`, "file contents", false, 10*time.Millisecond)
	if r == nil {
		t.Fatal("duplicate read should be classified as waste")
	}
	if r.Kind != WasteTransportation {
		t.Errorf("expected WasteTransportation, got %q", r.Kind)
	}
	if r.Tool != "Read" {
		t.Errorf("expected tool=Read, got %q", r.Tool)
	}
}

func TestWasteClassifier_ErrorResult_Defect(t *testing.T) {
	t.Parallel()

	wc := NewWasteClassifier(nil)

	r := wc.ClassifyCall("Bash", `{"command": "go build ./..."}`, "compilation failed", true, 5*time.Second)
	if r == nil {
		t.Fatal("error result should be classified as waste")
	}
	if r.Kind != WasteDefect {
		t.Errorf("expected WasteDefect, got %q", r.Kind)
	}
	if r.Tool != "Bash" {
		t.Errorf("expected tool=Bash, got %q", r.Tool)
	}
}

func TestWasteClassifier_LongIdle_Waiting(t *testing.T) {
	t.Parallel()

	wc := NewWasteClassifier(nil)

	// First call establishes the baseline.
	wc.ClassifyCall("Read", `{"file_path": "/a/b.go"}`, "ok", false, 10*time.Millisecond)

	// Simulate long idle by setting lastCall far in the past.
	wc.mu.Lock()
	wc.lastCall = time.Now().Add(-10 * time.Second)
	wc.mu.Unlock()

	// Second call after long idle — waiting waste.
	// elapsed=10ms, but the gap since lastCall is ~10s.
	r := wc.ClassifyCall("Bash", `{"command": "ls"}`, "output", false, 10*time.Millisecond)
	if r == nil {
		t.Fatal("long idle should be classified as waste")
	}
	if r.Kind != WasteWaiting {
		t.Errorf("expected WasteWaiting, got %q", r.Kind)
	}
}

func TestWasteClassifier_PackageSwitch_Motion(t *testing.T) {
	t.Parallel()

	wc := NewWasteClassifier(nil)

	// First file call — no waste (no previous package).
	r := wc.ClassifyCall("Read", `{"file_path": "/home/user/agent/loop.go"}`, "ok", false, 10*time.Millisecond)
	if r != nil {
		t.Fatalf("first call should not be waste, got %+v", r)
	}

	// Same package — no waste.
	r = wc.ClassifyCall("Read", `{"file_path": "/home/user/agent/mode.go"}`, "ok", false, 10*time.Millisecond)
	if r != nil {
		t.Fatalf("same package should not be waste, got %+v", r)
	}

	// Different package — motion waste.
	r = wc.ClassifyCall("Read", `{"file_path": "/home/user/tui/messages.go"}`, "ok", false, 10*time.Millisecond)
	if r == nil {
		t.Fatal("package switch should be classified as waste")
	}
	if r.Kind != WasteMotion {
		t.Errorf("expected WasteMotion, got %q", r.Kind)
	}
}

func TestWasteClassifier_NoWaste_Returns_Nil(t *testing.T) {
	t.Parallel()

	wc := NewWasteClassifier(nil)

	// Clean call: no error, first read, short elapsed, no package switch.
	r := wc.ClassifyCall("Read", `{"file_path": "/a/b.go"}`, "contents", false, 10*time.Millisecond)
	if r != nil {
		t.Fatalf("clean call should return nil, got %+v", r)
	}

	// Non-file tool calls don't trigger transportation or motion.
	r = wc.ClassifyCall("Bash", `{"command": "echo hello"}`, "hello", false, 10*time.Millisecond)
	if r != nil {
		t.Fatalf("bash call should return nil, got %+v", r)
	}
}

func TestWasteClassifier_Metrics(t *testing.T) {
	t.Parallel()

	wc := NewWasteClassifier(nil)

	// Clean call.
	wc.ClassifyCall("Read", `{"file_path": "/a/b.go"}`, "ok", false, 10*time.Millisecond)

	// Defect.
	wc.ClassifyCall("Bash", `{"command": "go build"}`, "fail", true, 1*time.Second)

	// Transportation (duplicate read).
	wc.ClassifyCall("Read", `{"file_path": "/a/b.go"}`, "ok", false, 10*time.Millisecond)

	// Another clean call.
	wc.ClassifyCall("Bash", `{"command": "echo ok"}`, "ok", false, 10*time.Millisecond)

	m := wc.Metrics()

	if m.Total != 2 {
		t.Errorf("expected total=2, got %d", m.Total)
	}
	if m.ByKind[WasteDefect] != 1 {
		t.Errorf("expected 1 defect, got %d", m.ByKind[WasteDefect])
	}
	if m.ByKind[WasteTransportation] != 1 {
		t.Errorf("expected 1 transportation, got %d", m.ByKind[WasteTransportation])
	}
	// 2 waste / 4 total = 0.5
	if m.WasteRate < 0.49 || m.WasteRate > 0.51 {
		t.Errorf("expected waste rate ~0.5, got %f", m.WasteRate)
	}
}

func TestWasteClassifier_Overproduction_Stub(t *testing.T) {
	t.Parallel()

	wc := NewWasteClassifier(nil)
	r := wc.ClassifyOverproduction("long output that nobody reads", "agent response")
	if r != nil {
		t.Fatal("overproduction stub should return nil")
	}
}

func TestWasteClassifier_Inventory_Stub(t *testing.T) {
	t.Parallel()

	wc := NewWasteClassifier(nil)
	r := wc.ClassifyInventory(100000, 50)
	if r != nil {
		t.Fatal("inventory stub should return nil")
	}
}

func TestWasteClassifier_Records(t *testing.T) {
	t.Parallel()

	wc := NewWasteClassifier(nil)

	// Generate a defect.
	wc.ClassifyCall("Bash", `{"command": "false"}`, "error", true, 10*time.Millisecond)

	records := wc.Records()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Kind != WasteDefect {
		t.Errorf("expected defect record, got %q", records[0].Kind)
	}

	// Verify it's a copy — modifying shouldn't affect internal state.
	records[0].Kind = WasteMotion
	original := wc.Records()
	if original[0].Kind != WasteDefect {
		t.Error("Records() should return a copy, not a reference")
	}
}

func TestExtractPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "file_path key",
			input: `{"file_path": "/home/user/foo.go"}`,
			want:  "/home/user/foo.go",
		},
		{
			name:  "path key",
			input: `{"path": "/home/user/bar", "pattern": "*.go"}`,
			want:  "/home/user/bar",
		},
		{
			name:  "no path",
			input: `{"command": "ls -la"}`,
			want:  "",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractPath(tt.input)
			if got != tt.want {
				t.Errorf("extractPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWasteClassifier_GlobGrepTransportation(t *testing.T) {
	t.Parallel()

	wc := NewWasteClassifier(nil)

	// Glob a path.
	r := wc.ClassifyCall("Glob", `{"path": "/home/user/agent"}`, "loop.go\nmode.go", false, 10*time.Millisecond)
	if r != nil {
		t.Fatalf("first glob should not be waste, got %+v", r)
	}

	// Grep the same path.
	r = wc.ClassifyCall("Grep", `{"path": "/home/user/agent"}`, "matches", false, 10*time.Millisecond)
	if r == nil {
		t.Fatal("duplicate path access via Grep should be classified as transportation")
	}
	if r.Kind != WasteTransportation {
		t.Errorf("expected WasteTransportation, got %q", r.Kind)
	}
}

func TestWasteClassifier_DefectPriority(t *testing.T) {
	t.Parallel()

	wc := NewWasteClassifier(nil)

	// Read a file.
	wc.ClassifyCall("Read", `{"file_path": "/a/b.go"}`, "ok", false, 10*time.Millisecond)

	// Read same file but with error — defect should take priority over transportation.
	r := wc.ClassifyCall("Read", `{"file_path": "/a/b.go"}`, "permission denied", true, 10*time.Millisecond)
	if r == nil {
		t.Fatal("expected waste record")
	}
	if r.Kind != WasteDefect {
		t.Errorf("defect should take priority over transportation, got %q", r.Kind)
	}
}
