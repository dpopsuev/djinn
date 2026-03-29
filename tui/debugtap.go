// debugtap.go — TUIRecorder captures rendered TUI frames for replay and live inspection.
package tui

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

// RecordingFrame is a single captured TUI render with component breakdown.
type RecordingFrame struct {
	Timestamp  time.Time        `json:"timestamp"`
	Frame      string           `json:"frame"`
	Width      int              `json:"width"`
	Height     int              `json:"height"`
	State      string           `json:"state"`
	Role       string           `json:"role"`
	Components *RecordingComponents `json:"components,omitempty"`
}

// RecordingComponents captures individual panel state for component-level inspection.
type RecordingComponents struct {
	Transcript []string `json:"transcript,omitempty"` // output panel lines
	Overlay    string   `json:"overlay,omitempty"`    // ephemeral overlay text
	InputValue string   `json:"input_value,omitempty"`
	InputFocus bool     `json:"input_focus"`
	FocusedIdx int      `json:"focused_idx"`
	Dashboard  string   `json:"dashboard,omitempty"` // rendered dashboard line
}

// TUIRecorder captures TUI frames to a ring buffer and optionally a JSONL file.
type TUIRecorder struct {
	mu     sync.RWMutex
	frames []RecordingFrame
	max    int // ring buffer capacity
	file   *os.File
	enc    *json.Encoder

	// Live state for the debug server
	state  string
	role   string
	width  int
	height int
}

// NewTUIRecorder creates a debug tap with the given ring buffer capacity.
// If path is non-empty, frames are also written to a JSONL file.
func NewTUIRecorder(capacity int, path string) (*TUIRecorder, error) {
	dt := &TUIRecorder{
		frames: make([]RecordingFrame, 0, capacity),
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

// Capture records a rendered frame with optional component breakdown.
func (dt *TUIRecorder) Capture(frame, state, role string, width, height int, components ...*RecordingComponents) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	df := RecordingFrame{
		Timestamp: time.Now(),
		Frame:     frame,
		Width:     width,
		Height:    height,
		State:     state,
		Role:      role,
	}
	if len(components) > 0 {
		df.Components = components[0]
	}

	// Ring buffer
	if len(dt.frames) >= dt.max {
		dt.frames = dt.frames[1:]
	}
	dt.frames = append(dt.frames, df)

	// File output
	if dt.enc != nil {
		dt.enc.Encode(df) //nolint:errcheck // error intentionally ignored
	}

	dt.state = state
	dt.role = role
	dt.width = width
	dt.height = height
}

// Last returns the most recent frame.
func (dt *TUIRecorder) Last() (RecordingFrame, bool) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	if len(dt.frames) == 0 {
		return RecordingFrame{}, false
	}
	return dt.frames[len(dt.frames)-1], true
}

// LastN returns the most recent N frames.
func (dt *TUIRecorder) LastN(n int) []RecordingFrame {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	if n > len(dt.frames) {
		n = len(dt.frames)
	}
	result := make([]RecordingFrame, n)
	copy(result, dt.frames[len(dt.frames)-n:])
	return result
}

// SearchFrames returns frames whose content matches the pattern.
func (dt *TUIRecorder) SearchFrames(pattern string) []RecordingFrame {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	var results []RecordingFrame
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
func (dt *TUIRecorder) DetectTransitions() []Transition {
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
func (dt *TUIRecorder) Close() error {
	if dt.file != nil {
		return dt.file.Close()
	}
	return nil
}

// ServeHTTP starts a debug HTTP server on a random port.
// Returns the listener (caller can get the port from ln.Addr()).
// ServeHTTP starts a debug HTTP server. Pass "" for random port.
func (dt *TUIRecorder) ServeHTTP(addr string) (net.Listener, error) {
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	mux := http.NewServeMux()

	// GET /debug/view — current frame. ?plain strips ANSI codes.
	mux.HandleFunc("GET /debug/view", func(w http.ResponseWriter, r *http.Request) {
		frame, ok := dt.Last()
		if !ok {
			http.Error(w, "no frames captured", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		content := frame.Frame
		if r.URL.Query().Has("plain") {
			content = stripANSI(content)
		}
		fmt.Fprint(w, content)
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
		json.NewEncoder(w).Encode(state) //nolint:errcheck // error intentionally ignored
	})

	// GET /debug/frames?n=5 — last N frames as JSON array
	mux.HandleFunc("GET /debug/frames", func(w http.ResponseWriter, r *http.Request) {
		n := 5
		if v := r.URL.Query().Get("n"); v != "" {
			fmt.Sscanf(v, "%d", &n) //nolint:errcheck // error intentionally ignored
		}
		frames := dt.LastN(n)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(frames) //nolint:errcheck // error intentionally ignored
	})

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second} //nolint:mnd // debug server
	go srv.Serve(ln)                                                      //nolint:errcheck // fire-and-forget debug server goroutine
	return ln, nil
}
