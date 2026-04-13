// tool_recorder.go — bridges Battery Recorder to Troupe EventLog.
//
// Every tool execution becomes an Event in the unified log.
// Set as middleware.SetDefaultRecorder at startup for catch-all coverage.
// Also add to specific Envelope builders for explicit recording.
//
// GOL-162, TSK-1097
package substrate

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dpopsuev/battery/middleware"
	"github.com/dpopsuev/troupe/signal"
)

var _ middleware.Recorder = (*ToolEventRecorder)(nil)

// ToolEventRecorder implements middleware.Recorder by emitting tool call
// events to a Troupe EventLog. TraceID is inherited from the traceFunc.
type ToolEventRecorder struct {
	eventLog  signal.EventLog
	traceFunc func() string // returns current trace ID
}

// NewToolEventRecorder creates a recorder that bridges to the EventLog.
func NewToolEventRecorder(log signal.EventLog, traceFunc func() string) *ToolEventRecorder {
	return &ToolEventRecorder{eventLog: log, traceFunc: traceFunc}
}

// Record emits a tool execution event to the EventLog.
func (r *ToolEventRecorder) Record(_ context.Context, tool string, input json.RawMessage, output string, err error, elapsed time.Duration) {
	var traceID string
	if r.traceFunc != nil {
		traceID = r.traceFunc()
	}

	isError := err != nil
	r.eventLog.Emit(signal.Event{
		TraceID: traceID,
		Source:  "tool",
		Kind:    "tool.executed",
		Data: toolExecution{
			Tool:    tool,
			Input:   truncate(string(input), 200),
			Output:  truncate(output, 200),
			IsError: isError,
			Elapsed: elapsed,
		},
	})
}

type toolExecution struct {
	Tool    string        `json:"tool"`
	Input   string        `json:"input"`
	Output  string        `json:"output"`
	IsError bool          `json:"is_error"`
	Elapsed time.Duration `json:"elapsed"`
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
