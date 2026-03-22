package repl

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/tools/builtin"
)

func testModel() Model {
	sess := session.New("test", "test-model", "/workspace")
	m := NewModel(Config{
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "agent",
	})
	m.ready = true
	m.width = 80
	m.height = 24
	return m
}

// asModel extracts Model from tea.Model which may be Model or *Model
// depending on whether handleKey (pointer receiver) or Update (value receiver) ran.
func asModel(t *testing.T, m tea.Model) Model {
	t.Helper()
	switch v := m.(type) {
	case Model:
		return v
	case *Model:
		return *v
	default:
		t.Fatalf("unexpected type %T", m)
		return Model{}
	}
}

func TestModel_WindowSizeMsg(t *testing.T) {
	m := testModel()
	m.ready = false
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := asModel(t, m2)
	if !model.ready {
		t.Fatal("should be ready after WindowSizeMsg")
	}
	if model.width != 120 || model.height != 40 {
		t.Fatalf("size = %dx%d, want 120x40", model.width, model.height)
	}
}

func TestModel_TextMsg_Streaming(t *testing.T) {
	m := testModel()
	m.state = stateStreaming
	m2, _ := m.Update(TextMsg("hello"))
	model := asModel(t, m2)
	if model.streamBuf.String() != "hello" {
		t.Fatalf("streamBuf = %q", model.streamBuf.String())
	}
}

func TestModel_TextMsg_Chunked(t *testing.T) {
	m := testModel()
	m.state = stateStreaming
	m.outputMode = outputChunked
	m2, _ := m.Update(TextMsg("hello"))
	model := asModel(t, m2)
	if model.chunkedBuf.String() != "hello" {
		t.Fatalf("chunkedBuf = %q", model.chunkedBuf.String())
	}
	if model.streamBuf.Len() != 0 {
		t.Fatal("streamBuf should be empty in chunked mode")
	}
}

func TestModel_ThinkingMsg(t *testing.T) {
	m := testModel()
	before := len(m.conversation)
	m2, _ := m.Update(ThinkingMsg("let me think"))
	model := asModel(t, m2)
	if len(model.conversation) != before+1 {
		t.Fatal("should append to conversation")
	}
}

func TestModel_ToolCallMsg_AgentMode(t *testing.T) {
	m := testModel()
	m.mode = agent.ModeAgent
	m.state = stateStreaming
	m2, _ := m.Update(ToolCallMsg{Call: driver.ToolCall{
		ID: "c1", Name: "Bash", Input: json.RawMessage(`{"command":"ls"}`),
	}})
	model := asModel(t, m2)
	if model.state != stateToolApproval {
		t.Fatalf("state = %d, want stateToolApproval", model.state)
	}
	if model.pendingTool == nil {
		t.Fatal("pendingTool should be set")
	}
}

func TestModel_ToolCallMsg_AutoMode(t *testing.T) {
	m := testModel()
	m.mode = agent.ModeAuto
	m.state = stateStreaming
	m2, _ := m.Update(ToolCallMsg{Call: driver.ToolCall{
		ID: "c1", Name: "Bash", Input: json.RawMessage(`{}`),
	}})
	model := asModel(t, m2)
	if model.state == stateToolApproval {
		t.Fatal("auto mode should not prompt for approval")
	}
}

func TestModel_ToolResultMsg_Success(t *testing.T) {
	m := testModel()
	before := len(m.conversation)
	m2, _ := m.Update(ToolResultMsg{Name: "Read", Output: "file contents", IsError: false})
	model := asModel(t, m2)
	if len(model.conversation) != before+1 {
		t.Fatal("should append to conversation")
	}
}

func TestModel_ToolResultMsg_Error(t *testing.T) {
	m := testModel()
	before := len(m.conversation)
	m2, _ := m.Update(ToolResultMsg{Name: "Read", Output: "not found", IsError: true})
	model := asModel(t, m2)
	if len(model.conversation) != before+1 {
		t.Fatal("should append to conversation")
	}
}

func TestModel_DoneMsg(t *testing.T) {
	m := testModel()
	m2, _ := m.Update(DoneMsg{Usage: &driver.Usage{InputTokens: 100, OutputTokens: 50}})
	model := asModel(t, m2)
	if model.totalIn != 100 || model.totalOut != 50 {
		t.Fatalf("tokens = %d/%d, want 100/50", model.totalIn, model.totalOut)
	}
}

func TestModel_AgentDoneMsg(t *testing.T) {
	m := testModel()
	m.state = stateStreaming
	m2, _ := m.Update(AgentDoneMsg{Result: "done"})
	model := asModel(t, m2)
	if model.state != stateInput {
		t.Fatalf("state = %d, want stateInput", model.state)
	}
}

func TestModel_AgentDoneMsg_WithError(t *testing.T) {
	m := testModel()
	m.state = stateStreaming
	m2, _ := m.Update(AgentDoneMsg{Err: errors.New("something failed")})
	model := asModel(t, m2)
	found := false
	for _, line := range model.conversation {
		if strings.Contains(line, "something failed") {
			found = true
		}
	}
	if !found {
		t.Fatal("error should appear in conversation")
	}
}

func TestModel_TickMsg_WhileStreaming(t *testing.T) {
	m := testModel()
	m.state = stateStreaming
	m.streamBuf.WriteString("buffered text")
	m.conversation = append(m.conversation, "assistant: ")

	m2, cmd := m.Update(TickMsg(time.Now()))
	model := asModel(t, m2)
	if cmd == nil {
		t.Fatal("should return tick cmd while streaming")
	}
	if model.streamBuf.Len() != 0 {
		t.Fatal("buffer should be flushed")
	}
}

func TestModel_TickMsg_WhileInput(t *testing.T) {
	m := testModel()
	m.state = stateInput
	_, cmd := m.Update(TickMsg(time.Now()))
	if cmd != nil {
		t.Fatal("no tick cmd when not streaming")
	}
}

func TestModel_HandleKey_CtrlC(t *testing.T) {
	m := testModel()
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model := asModel(t, m2)
	if !model.quitting {
		t.Fatal("should be quitting")
	}
	if cmd == nil {
		t.Fatal("should return quit cmd")
	}
}

func TestModel_HandleKey_HistoryUp(t *testing.T) {
	m := testModel()
	m.inputHistory = []string{"first", "second", "third"}
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	model := asModel(t, m2)
	if model.textInput.Value() != "third" {
		t.Fatalf("value = %q, want third", model.textInput.Value())
	}
}

func TestModel_HandleKey_HistoryDown(t *testing.T) {
	m := testModel()
	m.inputHistory = []string{"first", "second"}
	m.historyIdx = 0 // browsing at "first"
	m.textInput.SetValue("first")
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := asModel(t, m2)
	if model.textInput.Value() != "second" {
		t.Fatalf("value = %q, want second", model.textInput.Value())
	}
}

func TestModel_HandleApproval_Y(t *testing.T) {
	m := testModel()
	m.state = stateToolApproval
	m.pendingTool = &driver.ToolCall{Name: "Bash"}
	m2, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model := asModel(t, m2)
	if model.state != stateStreaming {
		t.Fatalf("state = %d, want stateStreaming", model.state)
	}
}

func TestModel_HandleApproval_N(t *testing.T) {
	m := testModel()
	m.state = stateToolApproval
	m.pendingTool = &driver.ToolCall{Name: "Bash"}
	m2, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model := asModel(t, m2)
	found := false
	for _, line := range model.conversation {
		if strings.Contains(line, "denied") {
			found = true
		}
	}
	if !found {
		t.Fatal("denied should appear in conversation")
	}
}

func TestModel_View_Welcome(t *testing.T) {
	m := testModel()
	view := m.View()
	// Logo contains "___" pattern from ASCII art
	if !strings.Contains(view, "___") {
		t.Fatalf("welcome should show logo, got: %q", view[:min(len(view), 200)])
	}
}


func TestModel_View_StatusBar(t *testing.T) {
	m := testModel()
	view := m.View()
	if !strings.Contains(view, "test-model") {
		t.Fatal("status bar should show model name")
	}
	if !strings.Contains(view, "agent") {
		t.Fatal("status bar should show mode")
	}
}

func TestModel_HandleSubmit_SlashCommand(t *testing.T) {
	m := testModel()
	m.textInput.SetValue("/help")
	m2, _ := m.handleSubmit()
	model := asModel(t, m2)
	found := false
	for _, line := range model.conversation {
		if strings.Contains(line, "commands:") {
			found = true
		}
	}
	if !found {
		t.Fatal("help output should appear in conversation")
	}
}

func TestModel_HandleSubmit_ModeSwitch(t *testing.T) {
	m := testModel()
	m.textInput.SetValue("/mode auto")
	m2, _ := m.handleSubmit()
	model := asModel(t, m2)
	if model.mode != agent.ModeAuto {
		t.Fatalf("mode = %s, want auto", model.mode)
	}
}

func TestModel_HandleSubmit_Empty(t *testing.T) {
	m := testModel()
	m.textInput.SetValue("")
	m2, cmd := m.handleSubmit()
	model := asModel(t, m2)
	if cmd != nil {
		t.Fatal("empty submit should return no cmd")
	}
	if model.state != stateInput {
		t.Fatal("state should remain input")
	}
}

func TestModel_HandleSubmit_Exit(t *testing.T) {
	m := testModel()
	m.textInput.SetValue("/exit")
	m2, _ := m.handleSubmit()
	model := asModel(t, m2)
	if !model.quitting {
		t.Fatal("should be quitting")
	}
}

func TestTruncate(t *testing.T) {
	if truncate("short", 10) != "short" {
		t.Fatal("short string should pass through")
	}
	long := strings.Repeat("x", 100)
	result := truncate(long, 10)
	if len(result) != 13 { // 10 + "..."
		t.Fatalf("truncated len = %d", len(result))
	}
}

func TestModel_FlushStreamBuffer(t *testing.T) {
	m := testModel()
	m.conversation = append(m.conversation, "assistant: ")
	m.streamBuf.WriteString("hello world")
	m.flushStreamBuffer()
	last := m.conversation[len(m.conversation)-1]
	if !strings.Contains(last, "hello world") {
		t.Fatalf("last line = %q", last)
	}
	if m.streamBuf.Len() != 0 {
		t.Fatal("buffer should be empty after flush")
	}
}

func TestModel_FlushStreamBuffer_Empty(t *testing.T) {
	m := testModel()
	before := len(m.conversation)
	m.flushStreamBuffer()
	if len(m.conversation) != before {
		t.Fatal("empty flush should not modify conversation")
	}
}

func TestModel_NewModel_DefaultMode(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	m := NewModel(Config{Tools: builtin.NewRegistry(), Session: sess})
	// Empty mode string defaults to agent
	if m.mode != agent.ModeAgent {
		t.Fatalf("default mode = %s, want agent", m.mode)
	}
}

func TestModel_NewModel_ParsesMode(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	m := NewModel(Config{Tools: builtin.NewRegistry(), Session: sess, Mode: "auto"})
	if m.mode != agent.ModeAuto {
		t.Fatalf("mode = %s, want auto", m.mode)
	}
}

// Ensure unused import suppression
var _ = fmt.Sprint
