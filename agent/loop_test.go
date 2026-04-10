package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/dpopsuev/djinn/cortex"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/tools/builtin"
)

func TestRun_SimpleText(t *testing.T) {
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
	sess := cortex.New("test-sess", "test-model", "/workspace")

	sess.Append(cortex.Entry{Role: driver.RoleUser, Content: "hello"})
	sess.Append(cortex.Entry{Role: driver.RoleAssistant, Content: "hi"})

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

// --- Full Run() cycle tests using scriptedTestDriver ---

// scriptedTestDriver is a minimal ChatDriver for testing agent.Run().
// Defined locally to avoid import cycle (agent → testkit/stubs → broker → agent).
type scriptedTestDriver struct {
	mu      sync.Mutex
	turns   [][]driver.StreamEvent
	current int
}

func newScriptedTestDriver(turns ...[]driver.StreamEvent) *scriptedTestDriver {
	return &scriptedTestDriver{turns: turns}
}

func (d *scriptedTestDriver) Start(context.Context, driver.SandboxHandle) error { return nil }
func (d *scriptedTestDriver) Stop(context.Context) error                        { return nil }
func (d *scriptedTestDriver) Send(context.Context, driver.Message) error        { return nil }
func (d *scriptedTestDriver) SendRich(context.Context, driver.RichMessage) error {
	return nil
}
func (d *scriptedTestDriver) AppendAssistant(driver.RichMessage) {}
func (d *scriptedTestDriver) SetSystemPrompt(string)             {}
func (d *scriptedTestDriver) ContextWindow() int                 { return 200_000 }

func (d *scriptedTestDriver) Chat(context.Context) (<-chan driver.StreamEvent, error) {
	d.mu.Lock()
	turn := d.current
	d.current++
	d.mu.Unlock()

	ch := make(chan driver.StreamEvent, 20)
	go func() {
		defer close(ch)
		if turn >= len(d.turns) {
			ch <- driver.StreamEvent{Type: driver.EventText, Text: "(no more turns)"}
			ch <- driver.StreamEvent{Type: driver.EventDone, Usage: &driver.Usage{}}
			return
		}
		for _, e := range d.turns[turn] {
			ch <- e
		}
	}()
	return ch, nil
}

func TestRun_FullCycle_TextOnly(t *testing.T) {
	d := newScriptedTestDriver([]driver.StreamEvent{
		{Type: driver.EventText, Text: "Hello from the agent"},
		{Type: driver.EventDone, Usage: &driver.Usage{OutputTokens: 10}},
	})

	sess := cortex.New("test-run", "test-model", t.TempDir())
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
	d := newScriptedTestDriver(
		// Turn 1: tool call
		[]driver.StreamEvent{
			{Type: driver.EventToolUse, ToolCall: &driver.ToolCall{
				ID: "call-1", Name: "Bash", Input: json.RawMessage(`{}`),
			}},
			{Type: driver.EventDone, Usage: &driver.Usage{OutputTokens: 5}},
		},
		// Turn 2: text response after denied tool result
		[]driver.StreamEvent{
			{Type: driver.EventText, Text: "OK, tool was denied"},
			{Type: driver.EventDone, Usage: &driver.Usage{OutputTokens: 5}},
		},
	)

	sess := cortex.New("test-denied", "test-model", t.TempDir())

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

// --- scriptedExecutor: local ToolExecutor for Run() tests ---
// Avoids import cycle (agent → testkit/stubs → broker → agent).
// Named to avoid collision with stubExecutor in envelope_test.go.

type scriptedExecutor struct {
	mu       sync.Mutex
	handlers map[string]func(json.RawMessage) (string, error)
	calls    []toolCallRecord
}

type toolCallRecord struct {
	Name  string
	Input json.RawMessage
}

func newScriptedExecutor() *scriptedExecutor {
	return &scriptedExecutor{handlers: make(map[string]func(json.RawMessage) (string, error))}
}

func (e *scriptedExecutor) Register(name string, fn func(json.RawMessage) (string, error)) {
	e.handlers[name] = fn
}

func (e *scriptedExecutor) Execute(_ context.Context, name string, input json.RawMessage) (string, error) {
	e.mu.Lock()
	e.calls = append(e.calls, toolCallRecord{Name: name, Input: input})
	e.mu.Unlock()

	fn, ok := e.handlers[name]
	if !ok {
		return "", errors.New("tool not found: " + name)
	}
	return fn(input)
}

func (e *scriptedExecutor) All() []builtin.Tool { return nil }
func (e *scriptedExecutor) Names() []string     { return nil }

func (e *scriptedExecutor) CallCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.calls)
}

func (e *scriptedExecutor) Calls() []toolCallRecord {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]toolCallRecord, len(e.calls))
	copy(out, e.calls)
	return out
}

// --- Layer 2: agent.Run() tool call execution tests ---

func TestRun_ToolCallExecution(t *testing.T) {
	// Turn 1: LLM requests Write tool
	// Turn 2: LLM sees tool result, responds with "done"
	d := newScriptedTestDriver(
		[]driver.StreamEvent{
			{Type: driver.EventText, Text: "I'll write the file"},
			{Type: driver.EventToolUse, ToolCall: &driver.ToolCall{
				ID:    "call-1",
				Name:  "Write",
				Input: json.RawMessage(`{"path":"test.txt","content":"hello"}`),
			}},
			{Type: driver.EventDone, Usage: &driver.Usage{OutputTokens: 15}},
		},
		[]driver.StreamEvent{
			{Type: driver.EventText, Text: "done"},
			{Type: driver.EventDone, Usage: &driver.Usage{OutputTokens: 5}},
		},
	)

	executor := newScriptedExecutor()
	executor.Register("Write", func(input json.RawMessage) (string, error) {
		var args map[string]string
		json.Unmarshal(input, &args) //nolint:errcheck // test
		return "wrote " + args["path"], nil
	})

	sess := cortex.New("test-tool", "test-model", t.TempDir())
	var handler testHandler

	result, err := Run(context.Background(), Config{
		Driver:       d,
		Tools:        executor,
		Session:      sess,
		MaxTurns:     5,
		ToolsEnabled: true,
		Approve:      AutoApprove,
		Handler:      &handler,
	}, "write test.txt")

	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != "done" {
		t.Fatalf("result = %q, want %q", result, "done")
	}

	// Verify tool was called
	if executor.CallCount() != 1 {
		t.Fatalf("tool calls = %d, want 1", executor.CallCount())
	}
	call := executor.Calls()[0]
	if call.Name != "Write" {
		t.Fatalf("tool name = %q, want Write", call.Name)
	}

	// Verify handler received events
	if len(handler.toolCalls) != 1 {
		t.Fatalf("handler toolCalls = %d, want 1", len(handler.toolCalls))
	}
	if len(handler.toolResults) != 1 {
		t.Fatalf("handler toolResults = %d, want 1", len(handler.toolResults))
	}

	// Verify session has full history: user + assistant(tool_call) + user(tool_result) + assistant(done)
	if sess.History.Len() != 4 {
		t.Fatalf("session history = %d, want 4", sess.History.Len())
	}
}

func TestRun_ToolError(t *testing.T) {
	// Turn 1: LLM requests a tool that fails
	// Turn 2: LLM sees error result, responds gracefully
	d := newScriptedTestDriver(
		[]driver.StreamEvent{
			{Type: driver.EventToolUse, ToolCall: &driver.ToolCall{
				ID:    "call-1",
				Name:  "Read",
				Input: json.RawMessage(`{"path":"/nonexistent"}`),
			}},
			{Type: driver.EventDone, Usage: &driver.Usage{OutputTokens: 10}},
		},
		[]driver.StreamEvent{
			{Type: driver.EventText, Text: "file not found"},
			{Type: driver.EventDone, Usage: &driver.Usage{OutputTokens: 5}},
		},
	)

	executor := newScriptedExecutor()
	executor.Register("Read", func(_ json.RawMessage) (string, error) {
		return "", errors.New("open /nonexistent: no such file")
	})

	sess := cortex.New("test-err", "test-model", t.TempDir())

	result, err := Run(context.Background(), Config{
		Driver:       d,
		Tools:        executor,
		Session:      sess,
		MaxTurns:     5,
		ToolsEnabled: true,
		Approve:      AutoApprove,
		Handler:      NilHandler{},
	}, "read the file")

	if err != nil {
		t.Fatalf("Run should not error on tool failure: %v", err)
	}
	if result != "file not found" {
		t.Fatalf("result = %q, want %q", result, "file not found")
	}

	// Tool was called, error result was sent back, agent continued
	if executor.CallCount() != 1 {
		t.Fatalf("tool calls = %d, want 1", executor.CallCount())
	}
}

func TestRun_MaxTurnsExceeded(t *testing.T) {
	// Every turn returns a tool call — should stop at max turns
	infiniteToolTurn := []driver.StreamEvent{
		{Type: driver.EventToolUse, ToolCall: &driver.ToolCall{
			ID:    "call-inf",
			Name:  "Noop",
			Input: json.RawMessage(`{}`),
		}},
		{Type: driver.EventDone, Usage: &driver.Usage{OutputTokens: 5}},
	}

	d := newScriptedTestDriver(
		infiniteToolTurn,
		infiniteToolTurn,
		infiniteToolTurn,
		infiniteToolTurn, // 4th turn — won't be reached with MaxTurns=3
	)

	executor := newScriptedExecutor()
	executor.Register("Noop", func(_ json.RawMessage) (string, error) {
		return "ok", nil
	})

	sess := cortex.New("test-max", "test-model", t.TempDir())

	_, err := Run(context.Background(), Config{
		Driver:       d,
		Tools:        executor,
		Session:      sess,
		MaxTurns:     3,
		ToolsEnabled: true,
		Approve:      AutoApprove,
		Handler:      NilHandler{},
	}, "loop forever")

	// agent.Run() should complete without error — it just stops after max turns
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Exactly 3 tool calls (one per turn, 3 turns max)
	if executor.CallCount() != 3 {
		t.Fatalf("tool calls = %d, want 3", executor.CallCount())
	}
}
