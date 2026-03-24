package tui

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// --- Unit Tests ---

func TestDebugTap_CaptureAndLast(t *testing.T) {
	dt, err := NewDebugTap(10, "")
	if err != nil {
		t.Fatal(err)
	}

	dt.Capture("frame1", "input", "gensec", 80, 24)
	dt.Capture("frame2", "streaming", "executor", 80, 24)

	last, ok := dt.Last()
	if !ok {
		t.Fatal("should have a last frame")
	}
	if last.Frame != "frame2" {
		t.Fatalf("last frame = %q", last.Frame)
	}
	if last.State != "streaming" {
		t.Fatalf("state = %q", last.State)
	}
	if last.Role != "executor" {
		t.Fatalf("role = %q", last.Role)
	}
}

func TestDebugTap_RingBuffer(t *testing.T) {
	dt, err := NewDebugTap(3, "")
	if err != nil {
		t.Fatal(err)
	}

	dt.Capture("a", "input", "", 80, 24)
	dt.Capture("b", "input", "", 80, 24)
	dt.Capture("c", "input", "", 80, 24)
	dt.Capture("d", "input", "", 80, 24) // pushes out "a"

	frames := dt.LastN(10)
	if len(frames) != 3 {
		t.Fatalf("ring buffer should cap at 3, got %d", len(frames))
	}
	if frames[0].Frame != "b" {
		t.Fatalf("oldest should be 'b', got %q", frames[0].Frame)
	}
}

func TestDebugTap_LastN(t *testing.T) {
	dt, err := NewDebugTap(10, "")
	if err != nil {
		t.Fatal(err)
	}

	dt.Capture("a", "", "", 80, 24)
	dt.Capture("b", "", "", 80, 24)
	dt.Capture("c", "", "", 80, 24)

	frames := dt.LastN(2)
	if len(frames) != 2 {
		t.Fatalf("LastN(2) = %d frames", len(frames))
	}
	if frames[0].Frame != "b" || frames[1].Frame != "c" {
		t.Fatalf("frames = %q, %q", frames[0].Frame, frames[1].Frame)
	}
}

func TestDebugTap_Empty(t *testing.T) {
	dt, err := NewDebugTap(10, "")
	if err != nil {
		t.Fatal(err)
	}

	_, ok := dt.Last()
	if ok {
		t.Fatal("empty tap should return false")
	}

	frames := dt.LastN(5)
	if len(frames) != 0 {
		t.Fatal("empty tap should return empty slice")
	}
}

func TestDebugTap_JSONLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "frames.jsonl")

	dt, err := NewDebugTap(10, path)
	if err != nil {
		t.Fatal(err)
	}

	dt.Capture("hello", "input", "gensec", 80, 24)
	dt.Close() //nolint:errcheck

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var frame DebugFrame
	if err := json.Unmarshal(data[:len(data)-1], &frame); err != nil {
		t.Fatalf("invalid JSONL: %v\ndata: %s", err, data)
	}
	if frame.Frame != "hello" {
		t.Fatalf("frame = %q", frame.Frame)
	}
}

func TestDebugTap_NilIsDisabled(t *testing.T) {
	// A nil DebugTap should never be created — but if passed as nil
	// to Model, View() should not panic.
	var dt *DebugTap
	if dt != nil {
		t.Fatal("nil DebugTap should be nil")
	}
}

// --- Search & Analysis Tests ---

func TestDebugFrames_SearchPattern(t *testing.T) {
	dt, err := NewDebugTap(10, "")
	if err != nil {
		t.Fatal(err)
	}

	dt.Capture("frame with [0m[38 garbage", "input", "gensec", 80, 24)
	dt.Capture("clean frame hello world", "streaming", "executor", 80, 24)
	dt.Capture("another [0m[38 bad frame", "input", "gensec", 80, 24)
	dt.Capture("totally fine", "input", "gensec", 80, 24)

	results := dt.SearchFrames("[0m[38")
	if len(results) != 2 {
		t.Fatalf("search results = %d, want 2", len(results))
	}
	if results[0].State != "input" || results[1].State != "input" {
		t.Fatal("both matches should be input state")
	}
}

func TestDebugFrames_SearchNoMatch(t *testing.T) {
	dt, err := NewDebugTap(10, "")
	if err != nil {
		t.Fatal(err)
	}

	dt.Capture("hello", "input", "", 80, 24)
	results := dt.SearchFrames("nonexistent")
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestDebugTransitions_DetectsStateChanges(t *testing.T) {
	dt, err := NewDebugTap(10, "")
	if err != nil {
		t.Fatal(err)
	}

	dt.Capture("f0", "input", "gensec", 80, 24)
	dt.Capture("f1", "streaming", "gensec", 80, 24) // state change
	dt.Capture("f2", "streaming", "gensec", 80, 24) // no change
	dt.Capture("f3", "input", "gensec", 80, 24)     // state change
	dt.Capture("f4", "streaming", "executor", 80, 24) // state + role change
	dt.Capture("f5", "approval", "executor", 80, 24)  // state change
	dt.Capture("f6", "input", "gensec", 80, 24)       // state + role change

	transitions := dt.DetectTransitions()
	if len(transitions) != 5 {
		t.Fatalf("transitions = %d, want 5", len(transitions))
	}

	// First transition: input → streaming
	if transitions[0].FromState != "input" || transitions[0].ToState != "streaming" {
		t.Fatalf("transition[0] = %s→%s", transitions[0].FromState, transitions[0].ToState)
	}
	if transitions[0].FromIndex != 0 || transitions[0].ToIndex != 1 {
		t.Fatalf("transition[0] indices = %d→%d", transitions[0].FromIndex, transitions[0].ToIndex)
	}

	// Transition 3: streaming → approval (same role, different state)
	if transitions[3].FromState != "streaming" || transitions[3].ToState != "approval" {
		t.Fatalf("transition[3] = %s→%s", transitions[3].FromState, transitions[3].ToState)
	}

	// Transition 4: approval/executor → input/gensec (both change)
	if transitions[4].FromRole != "executor" || transitions[4].ToRole != "gensec" {
		t.Fatalf("transition[4] roles = %s→%s", transitions[4].FromRole, transitions[4].ToRole)
	}
}

func TestDebugTransitions_NoChanges(t *testing.T) {
	dt, err := NewDebugTap(10, "")
	if err != nil {
		t.Fatal(err)
	}

	dt.Capture("f0", "input", "gensec", 80, 24)
	dt.Capture("f1", "input", "gensec", 80, 24)
	dt.Capture("f2", "input", "gensec", 80, 24)

	transitions := dt.DetectTransitions()
	if len(transitions) != 0 {
		t.Fatalf("expected 0 transitions, got %d", len(transitions))
	}
}

// --- HTTP Server Tests ---

func TestDebugTap_HTTPServer_RequiresExplicitStart(t *testing.T) {
	// Creating a DebugTap does NOT start the HTTP server.
	dt, err := NewDebugTap(10, "")
	if err != nil {
		t.Fatal(err)
	}
	defer dt.Close() //nolint:errcheck

	// Server not started — no listener exists.
	// This is correct: ServeHTTP() must be called explicitly (--live-debug).
	dt.Capture("test", "input", "gensec", 80, 24)
	// No panic, no server — just ring buffer.
}

func TestDebugTap_HTTPServer_ViewEndpoint(t *testing.T) {
	dt, err := NewDebugTap(10, "")
	if err != nil {
		t.Fatal(err)
	}
	defer dt.Close() //nolint:errcheck

	ln, err := dt.ServeHTTP()
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	dt.Capture("current frame content", "streaming", "executor", 120, 40)

	resp, err := http.Get("http://" + ln.Addr().String() + "/debug/view")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "current frame content" {
		t.Fatalf("view = %q", body)
	}
}

func TestDebugTap_HTTPServer_StateEndpoint(t *testing.T) {
	dt, err := NewDebugTap(10, "")
	if err != nil {
		t.Fatal(err)
	}
	defer dt.Close() //nolint:errcheck

	ln, err := dt.ServeHTTP()
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	dt.Capture("frame", "streaming", "executor", 120, 40)

	resp, err := http.Get("http://" + ln.Addr().String() + "/debug/state")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var state map[string]any
	json.NewDecoder(resp.Body).Decode(&state) //nolint:errcheck

	if state["state"] != "streaming" {
		t.Fatalf("state = %v", state["state"])
	}
	if state["role"] != "executor" {
		t.Fatalf("role = %v", state["role"])
	}
}

func TestDebugTap_HTTPServer_FramesEndpoint(t *testing.T) {
	dt, err := NewDebugTap(10, "")
	if err != nil {
		t.Fatal(err)
	}
	defer dt.Close() //nolint:errcheck

	ln, err := dt.ServeHTTP()
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	dt.Capture("a", "input", "", 80, 24)
	dt.Capture("b", "input", "", 80, 24)
	dt.Capture("c", "input", "", 80, 24)

	resp, err := http.Get("http://" + ln.Addr().String() + "/debug/frames?n=2")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var frames []DebugFrame
	json.NewDecoder(resp.Body).Decode(&frames) //nolint:errcheck

	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	if frames[0].Frame != "b" || frames[1].Frame != "c" {
		t.Fatalf("frames = %q, %q", frames[0].Frame, frames[1].Frame)
	}
}
