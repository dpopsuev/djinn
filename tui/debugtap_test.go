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
