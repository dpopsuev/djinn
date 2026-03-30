package trace

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExportImportRoundTrip(t *testing.T) {
	r := NewRing(100)
	tr := r.For(ComponentMCP)

	// Seed ring with diverse events.
	rt1 := tr.Begin("call", "scan repo").WithServer("locus").WithTool("codograph.scan")
	rt1.End()
	rt2 := tr.Begin("call", "list artifacts").WithServer("scribe").WithTool("artifact.list")
	rt2.EndWithError()
	tr.Event("emit", "budget signal")

	original := r.Last(100)
	if len(original) == 0 {
		t.Fatal("ring should have events")
	}

	// Export and import into a fresh ring.
	archive := Export(r, "")
	if len(archive.Events) != len(original) {
		t.Fatalf("archive events = %d, want %d", len(archive.Events), len(original))
	}

	r2 := NewRing(100)
	Import(archive, r2)

	imported := r2.Last(100)
	if len(imported) != len(original) {
		t.Fatalf("imported events = %d, want %d", len(imported), len(original))
	}

	// Verify content matches (IDs will differ because Import re-assigns them).
	for i := range original {
		if imported[i].Detail != original[i].Detail {
			t.Errorf("event %d: detail = %q, want %q", i, imported[i].Detail, original[i].Detail)
		}
		if imported[i].Component != original[i].Component {
			t.Errorf("event %d: component = %q, want %q", i, imported[i].Component, original[i].Component)
		}
		if imported[i].Server != original[i].Server {
			t.Errorf("event %d: server = %q, want %q", i, imported[i].Server, original[i].Server)
		}
		if imported[i].Tool != original[i].Tool {
			t.Errorf("event %d: tool = %q, want %q", i, imported[i].Tool, original[i].Tool)
		}
		if imported[i].Error != original[i].Error {
			t.Errorf("event %d: error = %v, want %v", i, imported[i].Error, original[i].Error)
		}
	}
}

func TestExportWithComponentFilter(t *testing.T) {
	r := NewRing(100)

	r.Append(TraceEvent{Component: ComponentMCP, Detail: "mcp1"})
	r.Append(TraceEvent{Component: ComponentAgent, Detail: "agent1"})
	r.Append(TraceEvent{Component: ComponentMCP, Detail: "mcp2"})

	archive := Export(r, ComponentMCP)
	if len(archive.Events) != 2 {
		t.Fatalf("filtered archive events = %d, want 2", len(archive.Events))
	}
	if archive.Filter != string(ComponentMCP) {
		t.Errorf("filter = %q, want %q", archive.Filter, ComponentMCP)
	}
	for _, e := range archive.Events {
		if e.Component != ComponentMCP {
			t.Errorf("unexpected component %q in filtered export", e.Component)
		}
	}
}

func TestSaveLoadJSON(t *testing.T) {
	r := NewRing(100)
	tr := r.For(ComponentMCP)

	rt := tr.Begin("call", "scan").WithServer("locus").WithTool("codograph.scan")
	rt.End()
	tr.Event("emit", "signal fired")

	archive := Export(r, "")
	archive.SessionID = "test-session-42"

	// Save to temp file.
	path := filepath.Join(t.TempDir(), "trace.json")
	if err := archive.SaveJSON(path); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	// Load back.
	loaded, err := LoadArchive(path)
	if err != nil {
		t.Fatalf("LoadArchive: %v", err)
	}

	if loaded.SessionID != archive.SessionID {
		t.Errorf("session_id = %q, want %q", loaded.SessionID, archive.SessionID)
	}
	if len(loaded.Events) != len(archive.Events) {
		t.Fatalf("loaded events = %d, want %d", len(loaded.Events), len(archive.Events))
	}
	for i := range archive.Events {
		if loaded.Events[i].Detail != archive.Events[i].Detail {
			t.Errorf("event %d: detail = %q, want %q", i, loaded.Events[i].Detail, archive.Events[i].Detail)
		}
	}
}

func TestSaveLoadJSONErrors(t *testing.T) {
	// Load from non-existent path.
	_, err := LoadArchive("/no/such/path/trace.json")
	if err == nil {
		t.Error("LoadArchive should fail on non-existent file")
	}

	// Load from invalid JSON.
	badPath := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badPath, []byte("{invalid json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = LoadArchive(badPath)
	if err == nil {
		t.Error("LoadArchive should fail on invalid JSON")
	}
}

func TestDiff(t *testing.T) {
	now := time.Now()

	beforeEvents := []TraceEvent{
		{Timestamp: now, Component: ComponentMCP, Server: "locus", Tool: "scan", Latency: 10 * time.Millisecond},
		{Timestamp: now, Component: ComponentMCP, Server: "locus", Tool: "scan", Latency: 12 * time.Millisecond},
		{Timestamp: now, Component: ComponentMCP, Server: "locus", Tool: "scan", Latency: 11 * time.Millisecond},
		{Timestamp: now, Component: ComponentMCP, Server: "scribe", Tool: "list", Error: true},
	}

	afterEvents := []TraceEvent{
		{Timestamp: now, Component: ComponentMCP, Server: "locus", Tool: "scan", Latency: 50 * time.Millisecond},
		{Timestamp: now, Component: ComponentMCP, Server: "locus", Tool: "scan", Latency: 55 * time.Millisecond},
		{Timestamp: now, Component: ComponentMCP, Server: "locus", Tool: "scan", Latency: 52 * time.Millisecond},
		{Timestamp: now, Component: ComponentMCP, Server: "lex", Tool: "match", Error: true},
	}

	before := &Archive{Events: beforeEvents}
	after := &Archive{Events: afterEvents}

	result := Diff(before, after)

	if result.EventCountBefore != 4 {
		t.Errorf("event count before = %d, want 4", result.EventCountBefore)
	}
	if result.EventCountAfter != 4 {
		t.Errorf("event count after = %d, want 4", result.EventCountAfter)
	}

	// Error rate: before has 1/4 = 0.25, after has 1/4 = 0.25.
	if result.ErrorRateBefore != 0.25 {
		t.Errorf("error rate before = %f, want 0.25", result.ErrorRateBefore)
	}

	// Latency deltas should exist for locus|scan (common key).
	if len(result.LatencyDeltas) == 0 {
		t.Fatal("expected latency deltas")
	}
	found := false
	for _, ld := range result.LatencyDeltas {
		if ld.Server == "locus" && ld.Tool == "scan" {
			found = true
			if ld.Change <= 0 {
				t.Errorf("expected positive change (slower), got %f", ld.Change)
			}
		}
	}
	if !found {
		t.Error("expected latency delta for locus|scan")
	}

	// New errors: lex.match is in after but not before.
	if len(result.NewErrors) != 1 {
		t.Fatalf("new errors = %d, want 1", len(result.NewErrors))
	}
	if result.NewErrors[0].Server != "lex" {
		t.Errorf("new error server = %q, want lex", result.NewErrors[0].Server)
	}

	// Resolved errors: scribe.list is in before but not after.
	if len(result.ResolvedErrors) != 1 {
		t.Fatalf("resolved errors = %d, want 1", len(result.ResolvedErrors))
	}
	if result.ResolvedErrors[0].Server != "scribe" {
		t.Errorf("resolved error server = %q, want scribe", result.ResolvedErrors[0].Server)
	}
}

func TestDiffNoCommonKeys(t *testing.T) {
	before := &Archive{Events: []TraceEvent{
		{Server: "a", Tool: "x", Latency: 10 * time.Millisecond},
	}}
	after := &Archive{Events: []TraceEvent{
		{Server: "b", Tool: "y", Latency: 20 * time.Millisecond},
	}}

	result := Diff(before, after)
	if len(result.LatencyDeltas) != 0 {
		t.Errorf("expected no latency deltas for disjoint keys, got %d", len(result.LatencyDeltas))
	}
}

func TestExportEmpty(t *testing.T) {
	r := NewRing(10)
	archive := Export(r, "")
	if archive == nil {
		t.Fatal("export of empty ring should return non-nil archive")
	}
	if len(archive.Events) != 0 {
		t.Errorf("empty ring archive should have 0 events, got %d", len(archive.Events))
	}
	if archive.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set even for empty export")
	}
}

func TestDiffBothEmpty(t *testing.T) {
	before := &Archive{Events: nil}
	after := &Archive{Events: nil}

	result := Diff(before, after)
	if result.EventCountBefore != 0 || result.EventCountAfter != 0 {
		t.Error("diff of empty archives should show zero counts")
	}
	if result.ErrorRateBefore != 0 || result.ErrorRateAfter != 0 {
		t.Error("error rates should be zero for empty archives")
	}
}
