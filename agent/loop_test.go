package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/dpopsuev/djinn/driver"
	claudedriver "github.com/dpopsuev/djinn/driver/claude"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/tools/builtin"
)

func sseTextResponse(text string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, `event: message_start
data: {"type":"message_start","message":{"id":"msg-1","role":"assistant"}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"%s"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":10}}

event: message_stop
data: {"type":"message_stop"}

`, text)
	}
}

func TestRun_SimpleText(t *testing.T) {
	srv := httptest.NewServer(sseTextResponse("Hello from Claude"))
	defer srv.Close()

	// Test collectResponse directly with event channels
	events := make(chan driver.StreamEvent, 10)
	events <- driver.StreamEvent{Type: driver.EventText, Text: "Hello "}
	events <- driver.StreamEvent{Type: driver.EventText, Text: "world"}
	events <- driver.StreamEvent{Type: driver.EventDone, Usage: &driver.Usage{OutputTokens: 5}}
	close(events)

	var handler testHandler
	resp, err := collectResponse(events, &handler)
	if err != nil {
		t.Fatalf("collectResponse: %v", err)
	}

	if resp.text != "Hello world" {
		t.Fatalf("text = %q, want %q", resp.text, "Hello world")
	}
	if len(resp.toolCalls) != 0 {
		t.Fatalf("toolCalls = %d, want 0", len(resp.toolCalls))
	}
	if handler.textReceived != "Hello world" {
		t.Fatalf("handler text = %q", handler.textReceived)
	}
	if !handler.doneReceived {
		t.Fatal("handler should have received done")
	}
}

func TestRun_WithToolCalls(t *testing.T) {
	events := make(chan driver.StreamEvent, 10)
	events <- driver.StreamEvent{Type: driver.EventText, Text: "Let me read the file."}
	events <- driver.StreamEvent{
		Type:     driver.EventToolUse,
		ToolCall: &driver.ToolCall{ID: "call-1", Name: "Read", Input: json.RawMessage(`{"path": "test.go"}`)},
	}
	events <- driver.StreamEvent{Type: driver.EventDone}
	close(events)

	var handler testHandler
	resp, err := collectResponse(events, &handler)
	if err != nil {
		t.Fatalf("collectResponse: %v", err)
	}

	if len(resp.toolCalls) != 1 {
		t.Fatalf("toolCalls = %d, want 1", len(resp.toolCalls))
	}
	if resp.toolCalls[0].Name != "Read" {
		t.Fatalf("tool name = %q", resp.toolCalls[0].Name)
	}
	if len(handler.toolCalls) != 1 {
		t.Fatalf("handler toolCalls = %d", len(handler.toolCalls))
	}
}

func TestExecuteTools_Success(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	os.WriteFile(testFile, []byte("hello world\n"), 0o644)

	input, _ := json.Marshal(map[string]string{"path": testFile})
	calls := []driver.ToolCall{
		{ID: "call-1", Name: "Read", Input: input},
	}

	var handler testHandler
	cfg := Config{
		Tools:   builtin.NewRegistry(),
		Approve: AutoApprove,
		Handler: &handler,
	}

	blocks, err := executeTools(context.Background(), cfg, calls)
	if err != nil {
		t.Fatalf("executeTools: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(blocks))
	}
	if blocks[0].ToolResult == nil {
		t.Fatal("expected tool result block")
	}
	if blocks[0].ToolResult.IsError {
		t.Fatalf("tool result is error: %s", blocks[0].ToolResult.Output)
	}
	if len(handler.toolResults) != 1 {
		t.Fatalf("handler results = %d", len(handler.toolResults))
	}
}

func TestExecuteTools_Denied(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"command": "rm -rf /"})
	calls := []driver.ToolCall{
		{ID: "call-1", Name: "Bash", Input: input},
	}

	cfg := Config{
		Tools:   builtin.NewRegistry(),
		Approve: DenyAll,
		Handler: NilHandler{},
	}

	blocks, err := executeTools(context.Background(), cfg, calls)
	if err != nil {
		t.Fatalf("executeTools: %v", err)
	}

	if !blocks[0].ToolResult.IsError {
		t.Fatal("denied tool should return error result")
	}
	if blocks[0].ToolResult.Output != "tool call denied by operator" {
		t.Fatalf("denied output = %q", blocks[0].ToolResult.Output)
	}
}

func TestExecuteTools_ToolNotFound(t *testing.T) {
	calls := []driver.ToolCall{
		{ID: "call-1", Name: "NonExistent", Input: json.RawMessage(`{}`)},
	}

	cfg := Config{
		Tools:   builtin.NewRegistry(),
		Approve: AutoApprove,
		Handler: NilHandler{},
	}

	blocks, err := executeTools(context.Background(), cfg, calls)
	if err != nil {
		t.Fatalf("executeTools: %v", err)
	}

	if !blocks[0].ToolResult.IsError {
		t.Fatal("unknown tool should return error result")
	}
}

func TestApproveByName(t *testing.T) {
	approve := ApproveByName("Read", "Grep")

	if !approve(driver.ToolCall{Name: "Read"}) {
		t.Fatal("Read should be approved")
	}
	if !approve(driver.ToolCall{Name: "Grep"}) {
		t.Fatal("Grep should be approved")
	}
	if approve(driver.ToolCall{Name: "Bash"}) {
		t.Fatal("Bash should be denied")
	}
}

func TestCollectResponse_Thinking(t *testing.T) {
	events := make(chan driver.StreamEvent, 10)
	events <- driver.StreamEvent{Type: driver.EventThinking, Thinking: "Let me think..."}
	events <- driver.StreamEvent{Type: driver.EventText, Text: "Answer"}
	events <- driver.StreamEvent{Type: driver.EventDone}
	close(events)

	resp, _ := collectResponse(events, NilHandler{})
	if resp.text != "Answer" {
		t.Fatalf("text = %q", resp.text)
	}

	// Should have thinking block + text block
	hasThinking := false
	for _, b := range resp.blocks {
		if b.Type == driver.BlockThinking {
			hasThinking = true
		}
	}
	if !hasThinking {
		t.Fatal("expected thinking block in response")
	}
}

func TestSessionIntegration(t *testing.T) {
	sess := session.New("test-sess", "test-model", "/workspace")

	sess.Append(session.Entry{Role: driver.RoleUser, Content: "hello"})
	sess.Append(session.Entry{Role: driver.RoleAssistant, Content: "hi"})

	if sess.History.Len() != 2 {
		t.Fatalf("history = %d, want 2", sess.History.Len())
	}
}

// testHandler records events for assertions.
type testHandler struct {
	textReceived string
	toolCalls    []driver.ToolCall
	toolResults  []string
	doneReceived bool
}

func (h *testHandler) OnText(text string)              { h.textReceived += text }
func (h *testHandler) OnThinking(string)               {}
func (h *testHandler) OnToolCall(call driver.ToolCall) { h.toolCalls = append(h.toolCalls, call) }
func (h *testHandler) OnToolResult(id, name, output string, _ bool) {
	h.toolResults = append(h.toolResults, name+": "+output)
}
func (h *testHandler) OnDone(*driver.Usage) { h.doneReceived = true }
func (h *testHandler) OnError(error)        {}

// --- Full Run() cycle tests ---

func newTestAPIDriver(t *testing.T, handler http.HandlerFunc) *claudedriver.APIDriver {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Cleanup(func() { os.Unsetenv("ANTHROPIC_API_KEY") })

	d, err := claudedriver.NewAPIDriver(
		driver.DriverConfig{Model: "test-model", MaxTokens: 1024},
		claudedriver.WithTools(builtin.NewRegistry()),
		claudedriver.WithAPIURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewAPIDriver: %v", err)
	}
	if err := d.Start(context.Background(), ""); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { d.Stop(context.Background()) })
	return d
}

func TestRun_FullCycle_TextOnly(t *testing.T) {
	d := newTestAPIDriver(t, sseTextResponse("Hello from the agent"))

	sess := session.New("test-run", "test-model", t.TempDir())
	var handler testHandler

	result, err := Run(context.Background(), Config{
		Driver:   d,
		Tools:    builtin.NewRegistry(),
		Session:  sess,
		MaxTurns: 5,
		Approve:  AutoApprove,
		Handler:  &handler,
	}, "say hello")

	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != "Hello from the agent" {
		t.Fatalf("result = %q, want %q", result, "Hello from the agent")
	}
	if !handler.doneReceived {
		t.Fatal("handler should have received done")
	}
	if handler.textReceived != "Hello from the agent" {
		t.Fatalf("handler text = %q", handler.textReceived)
	}
	// Session should have 2 entries (user + assistant)
	if sess.History.Len() != 2 {
		t.Fatalf("session history = %d, want 2", sess.History.Len())
	}
}

func TestRun_ToolApprovalDenied(t *testing.T) {
	// Mock that returns a tool call
	toolSSE := `event: message_start
data: {"type":"message_start","message":{"id":"msg-1","role":"assistant"}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"call-1","name":"Bash","input":{}}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":5}}

event: message_stop
data: {"type":"message_stop"}

`

	callCount := 0
	handler := func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/event-stream")
		if callCount == 1 {
			// First call: return tool use
			fmt.Fprint(w, toolSSE)
		} else {
			// Second call (after tool result): return text
			fmt.Fprint(w, `event: message_start
data: {"type":"message_start","message":{"id":"msg-2","role":"assistant"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"OK, tool was denied"}}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}

event: message_stop
data: {"type":"message_stop"}

`)
		}
	}

	d := newTestAPIDriver(t, handler)
	sess := session.New("test-denied", "test-model", t.TempDir())

	result, err := Run(context.Background(), Config{
		Driver:   d,
		Tools:    builtin.NewRegistry(),
		Session:  sess,
		MaxTurns: 5,
		Approve:  DenyAll, // deny everything
		Handler:  NilHandler{},
	}, "run a dangerous command")

	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	_ = result
	// Should have completed without crashing despite denied tool
}
