// waste.go — Agent waste detection following Lean manufacturing's 7 wastes.
//
// The WasteClassifier is middleware that observes tool calls and classifies
// wasteful patterns: duplicate file reads (transportation), errors (defect),
// idle time (waiting), and unrelated file switching (motion).
//
// Overproduction and inventory are deferred classifiers — they require
// multi-turn context that isn't available at tool-call time.
package agent

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/djinnlog"
)

// WasteKind categorizes agent waste using Lean manufacturing's 7 wastes
// adapted for LLM agent workflows.
type WasteKind string

const (
	// WasteOverproduction: tool output not used by agent.
	WasteOverproduction WasteKind = "overproduction"

	// WasteWaiting: idle time between tool calls.
	WasteWaiting WasteKind = "waiting"

	// WasteTransportation: duplicate read of same file/path.
	WasteTransportation WasteKind = "transportation"

	// WasteOverProcessing: solution exceeds requirements.
	WasteOverProcessing WasteKind = "over-processing"

	// WasteInventory: stale context accumulated without proportional output.
	WasteInventory WasteKind = "inventory"

	// WasteMotion: switching between unrelated files/packages.
	WasteMotion WasteKind = "motion"

	// WasteDefect: compile fail, test fail, revert.
	WasteDefect WasteKind = "defect"
)

// AllWasteKinds returns every defined WasteKind constant.
func AllWasteKinds() []WasteKind {
	return []WasteKind{
		WasteOverproduction,
		WasteWaiting,
		WasteTransportation,
		WasteOverProcessing,
		WasteInventory,
		WasteMotion,
		WasteDefect,
	}
}

// WasteRecord captures a single waste classification event.
type WasteRecord struct {
	Tool      string
	Kind      WasteKind
	Reason    string
	Timestamp time.Time
}

// WasteMetrics summarizes waste across a session.
type WasteMetrics struct {
	Total     int
	ByKind    map[WasteKind]int
	WasteRate float64 // waste / total calls
}

// waitingThreshold is the idle time between calls that triggers WasteWaiting.
const waitingThreshold = 5 * time.Second

// fileTools are tool names that operate on file paths.
var fileTools = map[string]bool{
	"Read": true,
	"Glob": true,
	"Grep": true,
	"Edit": true,
}

// WasteClassifier observes tool executions and classifies wasteful patterns.
type WasteClassifier struct {
	records    []WasteRecord
	readCache  map[string]bool // file path → already read
	lastPkg    string          // last package directory accessed (for motion detection)
	lastCall   time.Time       // for waiting detection
	totalCalls int             // total tool calls observed
	mu         sync.Mutex
	log        *slog.Logger
}

// NewWasteClassifier creates a classifier that tracks waste across a session.
func NewWasteClassifier(log *slog.Logger) *WasteClassifier {
	if log == nil {
		log = djinnlog.Nop()
	}
	return &WasteClassifier{
		readCache: make(map[string]bool),
		log:       log,
	}
}

// ClassifyCall analyzes a completed tool call and returns a WasteRecord if
// waste is detected, or nil if the call is clean.
func (wc *WasteClassifier) ClassifyCall(toolName, input, output string, isError bool, elapsed time.Duration) *WasteRecord {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	wc.totalCalls++
	now := time.Now()

	// Check each waste type in priority order.
	var record *WasteRecord

	// 1. Defect: tool returned an error.
	if record == nil && isError {
		record = &WasteRecord{
			Tool:      toolName,
			Kind:      WasteDefect,
			Reason:    "tool returned error",
			Timestamp: now,
		}
	}

	// 2. Transportation: duplicate file read.
	if record == nil {
		record = wc.classifyTransportation(toolName, input, now)
	}

	// 3. Waiting: idle time between calls.
	if record == nil && !wc.lastCall.IsZero() {
		idleTime := now.Add(-elapsed).Sub(wc.lastCall)
		if idleTime > waitingThreshold {
			record = &WasteRecord{
				Tool:      toolName,
				Kind:      WasteWaiting,
				Reason:    "idle " + idleTime.Truncate(time.Millisecond).String() + " between calls",
				Timestamp: now,
			}
		}
	}

	// 4. Motion: switching between unrelated packages.
	if record == nil {
		record = wc.classifyMotion(toolName, input, now)
	}

	// Update tracking state.
	wc.lastCall = now
	wc.updatePkgTracking(toolName, input)

	if record != nil {
		wc.records = append(wc.records, *record)
		wc.log.InfoContext(context.Background(), "waste detected",
			slog.String(djinnlog.KeyTool, toolName),
			slog.String(djinnlog.KeyWasteKind, string(record.Kind)),
			slog.String(djinnlog.KeyReason, record.Reason),
		)
	} else {
		wc.log.DebugContext(context.Background(), "tool call classified",
			slog.String(djinnlog.KeyTool, toolName),
			slog.String(djinnlog.KeyWasteKind, "none"),
		)
	}

	return record
}

// ClassifyOverproduction checks if the previous tool's output was used by the agent.
// This is a deferred classifier — it must be called after the agent responds.
// TODO: implement output reference tracking.
func (wc *WasteClassifier) ClassifyOverproduction(_, _ string) *WasteRecord {
	return nil
}

// ClassifyInventory checks if context is growing without proportional output.
// TODO: implement context-to-output ratio tracking.
func (wc *WasteClassifier) ClassifyInventory(_, _ int) *WasteRecord {
	return nil
}

// Metrics returns a snapshot of waste metrics across the session.
func (wc *WasteClassifier) Metrics() WasteMetrics {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	byKind := make(map[WasteKind]int)
	for _, r := range wc.records {
		byKind[r.Kind]++
	}

	var rate float64
	if wc.totalCalls > 0 {
		rate = float64(len(wc.records)) / float64(wc.totalCalls)
	}

	return WasteMetrics{
		Total:     len(wc.records),
		ByKind:    byKind,
		WasteRate: rate,
	}
}

// Records returns a copy of all waste records.
func (wc *WasteClassifier) Records() []WasteRecord {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	out := make([]WasteRecord, len(wc.records))
	copy(out, wc.records)
	return out
}

// classifyTransportation checks if a file tool is re-reading a path already seen.
func (wc *WasteClassifier) classifyTransportation(toolName, input string, now time.Time) *WasteRecord {
	if !fileTools[toolName] {
		return nil
	}

	path := extractPath(input)
	if path == "" {
		return nil
	}

	if wc.readCache[path] {
		return &WasteRecord{
			Tool:      toolName,
			Kind:      WasteTransportation,
			Reason:    "duplicate access: " + path,
			Timestamp: now,
		}
	}

	wc.readCache[path] = true
	return nil
}

// classifyMotion checks if consecutive file tool calls are in different packages.
func (wc *WasteClassifier) classifyMotion(toolName, input string, now time.Time) *WasteRecord {
	if !fileTools[toolName] {
		return nil
	}

	path := extractPath(input)
	if path == "" {
		return nil
	}

	pkg := filepath.Dir(path)
	if wc.lastPkg != "" && pkg != wc.lastPkg {
		return &WasteRecord{
			Tool:      toolName,
			Kind:      WasteMotion,
			Reason:    "package switch: " + wc.lastPkg + " -> " + pkg,
			Timestamp: now,
		}
	}

	return nil
}

// updatePkgTracking updates the last-package state for motion detection.
func (wc *WasteClassifier) updatePkgTracking(toolName, input string) {
	if !fileTools[toolName] {
		return
	}
	path := extractPath(input)
	if path == "" {
		return
	}
	wc.lastPkg = filepath.Dir(path)
}

// extractPath pulls a file_path or path from JSON tool input.
// Handles common tool input shapes: {"file_path": "..."} and {"path": "..."}.
func extractPath(input string) string {
	// Fast path: look for file_path first, then path.
	for _, key := range []string{"file_path", "path"} {
		idx := strings.Index(input, `"`+key+`"`)
		if idx < 0 {
			continue
		}
		// Find the value after the key.
		rest := input[idx+len(key)+2:]
		// Skip `: "`
		colonIdx := strings.Index(rest, `"`)
		if colonIdx < 0 {
			continue
		}
		rest = rest[colonIdx+1:]
		endIdx := strings.Index(rest, `"`)
		if endIdx < 0 {
			continue
		}
		return rest[:endIdx]
	}
	return ""
}
