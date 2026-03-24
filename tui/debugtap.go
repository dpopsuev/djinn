// debugtap.go — captures rendered TUI frames for replay and live inspection.
package tui

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// DebugFrame is a single captured TUI render.
type DebugFrame struct {
	Timestamp time.Time `json:"timestamp"`
	Frame     string    `json:"frame"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	State     string    `json:"state"`
	Role      string    `json:"role"`
}

// DebugTap captures TUI frames to a ring buffer and optionally a JSONL file.
type DebugTap struct {
	mu     sync.RWMutex
	frames []DebugFrame
	max    int // ring buffer capacity
	file   *os.File
	enc    *json.Encoder

	// Live state for the debug server
	state string
	role  string
	width int
	height int
}

// NewDebugTap creates a debug tap with the given ring buffer capacity.
// If path is non-empty, frames are also written to a JSONL file.
func NewDebugTap(capacity int, path string) (*DebugTap, error) {
	dt := &DebugTap{
		frames: make([]DebugFrame, 0, capacity),
		max:    capacity,
	}
	if path != "" {
		f, err := os.Create(path)
		if err != nil {
			return nil, fmt.Errorf("debug tap file: %w", err)
		}
		dt.file = f
		dt.enc = json.NewEncoder(f)
	}
	return dt, nil
}

// Capture records a rendered frame.
func (dt *DebugTap) Capture(frame, state, role string, width, height int) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	df := DebugFrame{
		Timestamp: time.Now(),
		Frame:     frame,
		Width:     width,
		Height:    height,
		State:     state,
		Role:      role,
	}

	// Ring buffer
	if len(dt.frames) >= dt.max {
		dt.frames = dt.frames[1:]
	}
	dt.frames = append(dt.frames, df)

	// File output
	if dt.enc != nil {
		dt.enc.Encode(df) //nolint:errcheck
	}

	dt.state = state
	dt.role = role
	dt.width = width
	dt.height = height
}

// Last returns the most recent frame.
func (dt *DebugTap) Last() (DebugFrame, bool) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	if len(dt.frames) == 0 {
		return DebugFrame{}, false
	}
	return dt.frames[len(dt.frames)-1], true
}

// LastN returns the most recent N frames.
func (dt *DebugTap) LastN(n int) []DebugFrame {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	if n > len(dt.frames) {
		n = len(dt.frames)
	}
	result := make([]DebugFrame, n)
	copy(result, dt.frames[len(dt.frames)-n:])
	return result
}

// SearchFrames returns frames whose content matches the pattern.
func (dt *DebugTap) SearchFrames(pattern string) []DebugFrame {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	var results []DebugFrame
	for _, f := range dt.frames {
		if strings.Contains(f.Frame, pattern) {
			results = append(results, f)
		}
	}
	return results
}

// Transition records a state or role change between frames.
type Transition struct {
	FromIndex int
	ToIndex   int
	FromState string
	ToState   string
	FromRole  string
	ToRole    string
	Timestamp time.Time
}

// DetectTransitions returns state/role changes between consecutive frames.
func (dt *DebugTap) DetectTransitions() []Transition {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	var transitions []Transition
	for i := 1; i < len(dt.frames); i++ {
		prev := dt.frames[i-1]
		curr := dt.frames[i]
		if prev.State != curr.State || prev.Role != curr.Role {
			transitions = append(transitions, Transition{
				FromIndex: i - 1,
				ToIndex:   i,
				FromState: prev.State,
				ToState:   curr.State,
				FromRole:  prev.Role,
				ToRole:    curr.Role,
				Timestamp: curr.Timestamp,
			})
		}
	}
	return transitions
}

// Close closes the JSONL file if open.
func (dt *DebugTap) Close() error {
	if dt.file != nil {
		return dt.file.Close()
	}
	return nil
}

// ServeHTTP starts a debug HTTP server on a random port.
// Returns the listener (caller can get the port from ln.Addr()).
// ServeHTTP starts a debug HTTP server. Pass "" for random port.
func (dt *DebugTap) ServeHTTP(addr string) (net.Listener, error) {
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	mux := http.NewServeMux()

	// GET /debug/view — current frame (raw text)
	mux.HandleFunc("GET /debug/view", func(w http.ResponseWriter, r *http.Request) {
		frame, ok := dt.Last()
		if !ok {
			http.Error(w, "no frames captured", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, frame.Frame)
	})

	// GET /debug/state — current state as JSON
	mux.HandleFunc("GET /debug/state", func(w http.ResponseWriter, r *http.Request) {
		dt.mu.RLock()
		state := map[string]any{
			"state":       dt.state,
			"role":        dt.role,
			"width":       dt.width,
			"height":      dt.height,
			"frame_count": len(dt.frames),
		}
		dt.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(state) //nolint:errcheck
	})

	// GET /debug/frames?n=5 — last N frames as JSON array
	mux.HandleFunc("GET /debug/frames", func(w http.ResponseWriter, r *http.Request) {
		n := 5
		if v := r.URL.Query().Get("n"); v != "" {
			fmt.Sscanf(v, "%d", &n) //nolint:errcheck
		}
		frames := dt.LastN(n)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(frames) //nolint:errcheck
	})

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	go http.Serve(ln, mux) //nolint:errcheck
	return ln, nil
}
