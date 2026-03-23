package acceptance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/cli/repl"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/tui"
)

func testTUIModel(t *testing.T, mode string) *repl.Model {
	t.Helper()
	sess := session.New("tui-test", "test-model", "/workspace")
	m := repl.NewModel(repl.Config{
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    mode,
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return toModelPtr(m2)
}

func toModelPtr(m tea.Model) *repl.Model {
	switch v := m.(type) {
	case repl.Model:
		return &v
	case *repl.Model:
		return v
	default:
		panic("unexpected type")
	}
}

func TestTUI_SlashCommandNoStreaming(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := repl.ExecuteCommand(repl.Command{Name: "/help"}, sess)
	if result.Output == "" {
		t.Fatal("/help should produce output")
	}
}

func TestTUI_TextBufferedAndFlushed(t *testing.T) {
	m := testTUIModel(t, "agent")
	m.SetState(repl.StateStreaming)
	m.AppendConversation("assistant: ")

	m2, _ := m.Update(tui.TextMsg("hello world"))
	model := toModelPtr(m2)

	if model.StreamBufString() != "hello world" {
		t.Fatalf("streamBuf = %q", model.StreamBufString())
	}

	m3, _ := model.Update(tui.TickMsg(time.Now()))
	model2 := toModelPtr(m3)
	if model2.StreamBufString() != "" {
		t.Fatal("buffer should be empty after tick flush")
	}
}

func TestTUI_ToolApprovalInAgentMode(t *testing.T) {
	m := testTUIModel(t, "agent")
	m.SetState(repl.StateStreaming)
	m2, _ := m.Update(tui.ToolCallMsg{Call: driver.ToolCall{
		ID: "c1", Name: "Bash", Input: json.RawMessage(`{}`),
	}})
	model := toModelPtr(m2)
	if model.CurrentState() != repl.StateToolApproval {
		t.Fatalf("state = %d, want StateToolApproval", model.CurrentState())
	}
}

func TestTUI_AutoModeSkipsApproval(t *testing.T) {
	m := testTUIModel(t, "auto")
	m.SetState(repl.StateStreaming)
	m2, _ := m.Update(tui.ToolCallMsg{Call: driver.ToolCall{
		ID: "c1", Name: "Bash", Input: json.RawMessage(`{}`),
	}})
	model := toModelPtr(m2)
	if model.CurrentState() == repl.StateToolApproval {
		t.Fatal("auto mode should not prompt for approval")
	}
}

func TestTUI_InputHistoryUpDown(t *testing.T) {
	m := testTUIModel(t, "agent")
	m.AddInputHistory("first prompt")
	m.AddInputHistory("second prompt")

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	model := toModelPtr(m2)
	if model.TextInputValue() != "second prompt" {
		t.Fatalf("up = %q, want second prompt", model.TextInputValue())
	}
}

func TestTUI_StatusBarShowsMode(t *testing.T) {
	m := testTUIModel(t, "plan")
	view := m.View()
	if !strings.Contains(view, "plan") {
		t.Fatal("status bar should show mode")
	}
}

func TestTUI_AllModesValid(t *testing.T) {
	for _, name := range []string{"ask", "plan", "agent", "auto"} {
		m, err := agent.ParseMode(name)
		if err != nil {
			t.Fatalf("ParseMode(%q): %v", name, err)
		}
		if m.String() != name {
			t.Fatalf("roundtrip: %q != %q", m.String(), name)
		}
	}
}

// --- Event Loop Tests (DJN-BUG-8) ---
// These tests push messages through multiple Update() copies,
// simulating the exact value-copy path that Bubbletea does.

// multiUpdate pushes N messages through the Model, copying on each step.
// This catches strings.Builder and any other non-copyable fields.
func multiUpdate(t *testing.T, m tea.Model, msgs ...tea.Msg) tea.Model {
	t.Helper()
	for i, msg := range msgs {
		defer func(step int) {
			if r := recover(); r != nil {
				t.Fatalf("PANIC at Update step %d (msg=%T): %v", step, msgs[step], r)
			}
		}(i)
		m, _ = m.Update(msg)
	}
	return m
}

func TestTUI_StreamingCycle_NoPanic(t *testing.T) {
	// The most basic flow: init → type text → stream response → done.
	// This catches value-copy panics on strings.Builder fields (DJN-BUG-7).
	m := testTUIModel(t, "agent")

	// Simulate: user types, agent streams tokens, agent finishes.
	m.SetState(repl.StateStreaming)
	m.AppendConversation(tui.AssistStyle.Render(tui.LabelAssist) + ": ")

	result := multiUpdate(t, *m,
		tui.TextMsg("Hello "),        // token 1
		tui.TickMsg(time.Now()),       // flush 1
		tui.TextMsg("world, "),        // token 2
		tui.TickMsg(time.Now()),       // flush 2
		tui.TextMsg("how are you?"),   // token 3
		tui.TickMsg(time.Now()),       // flush 3
		tui.AgentDoneMsg{},            // completion
	)

	model := toModelPtr(result)
	if model.CurrentState() != repl.StateInput {
		t.Fatalf("state after done = %d, want StateInput", model.CurrentState())
	}
}

func TestTUI_MultipleConversationTurns_NoPanic(t *testing.T) {
	// Two full conversation turns: prompt → stream → done → prompt → stream → done.
	m := testTUIModel(t, "agent")

	// Turn 1
	m.SetState(repl.StateStreaming)
	m.AppendConversation(tui.AssistStyle.Render(tui.LabelAssist) + ": ")

	result := multiUpdate(t, *m,
		tui.TextMsg("response one"),
		tui.TickMsg(time.Now()),
		tui.AgentDoneMsg{},
	)

	// Turn 2: re-enter streaming
	model := toModelPtr(result)
	model.SetState(repl.StateStreaming)
	model.AppendConversation(tui.AssistStyle.Render(tui.LabelAssist) + ": ")

	result2 := multiUpdate(t, *model,
		tui.TextMsg("response two"),
		tui.TickMsg(time.Now()),
		tui.AgentDoneMsg{},
	)

	model2 := toModelPtr(result2)
	if model2.CurrentState() != repl.StateInput {
		t.Fatalf("state after turn 2 = %d", model2.CurrentState())
	}
	if model2.ConversationLen() < 4 {
		t.Fatalf("conversation should have entries from both turns, got %d", model2.ConversationLen())
	}
}

func TestTUI_ToolApprovalCycle_NoPanic(t *testing.T) {
	// Stream → tool call → approval → stream → done.
	m := testTUIModel(t, "agent")
	m.SetState(repl.StateStreaming)
	m.AppendConversation(tui.AssistStyle.Render(tui.LabelAssist) + ": ")

	result := multiUpdate(t, *m,
		tui.TextMsg("Let me read that file."),
		tui.TickMsg(time.Now()),
		tui.ToolCallMsg{Call: driver.ToolCall{ID: "c1", Name: "Read", Input: json.RawMessage(`{"path":"main.go"}`)}},
	)

	model := toModelPtr(result)
	if model.CurrentState() != repl.StateToolApproval {
		t.Fatalf("state = %d, want tool approval", model.CurrentState())
	}

	// Approve via key press
	result2 := multiUpdate(t, *model,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}},
	)

	model2 := toModelPtr(result2)
	if model2.CurrentState() == repl.StateToolApproval {
		t.Fatal("should leave approval state after pressing y")
	}
}

func TestTUI_SlashCommand_NoPanic(t *testing.T) {
	// Type /help through the event loop — verify no panic on copy.
	m := testTUIModel(t, "agent")

	// Simulate typing /help and pressing Enter
	m.SetTextInput("/help")
	result := multiUpdate(t, *m,
		tea.KeyMsg{Type: tea.KeyEnter},
	)

	model := toModelPtr(result)
	if model.ConversationLen() == 0 {
		t.Fatal("help should add output to conversation")
	}
}

func TestTUI_RoleSwitch_NoPanic(t *testing.T) {
	// Switch roles via /role command through the event loop.
	// Needs a driver because switchRole calls SetSystemPrompt.
	sess := session.New("tui-test", "test-model", "/workspace")
	m := repl.NewModel(repl.Config{
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "agent",
		Driver:  &stubs.StubChatDriver{},
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	mp := toModelPtr(m2)

	mp.SetTextInput("/role executor")
	result := multiUpdate(t, *mp,
		tea.KeyMsg{Type: tea.KeyEnter},
	)

	model := toModelPtr(result)
	view := model.View()
	if !strings.Contains(strings.ToUpper(view), "EXECUTOR") {
		t.Fatal("dashboard should show EXECUTOR after role switch")
	}
}

func TestTUI_ViewRender_NoPanic(t *testing.T) {
	// Render View() after streaming — catches any render-path panics.
	m := testTUIModel(t, "agent")
	m.SetState(repl.StateStreaming)
	m.AppendConversation(tui.AssistStyle.Render(tui.LabelAssist) + ": ")

	result := multiUpdate(t, *m,
		tui.TextMsg("hello"),
		tui.TickMsg(time.Now()),
	)

	model := toModelPtr(result)
	view := model.View()
	if view == "" {
		t.Fatal("view should not be empty during streaming")
	}
	if strings.Contains(view, "[0m[38") {
		t.Fatal("view contains raw ANSI escape literals (DJN-BUG-5)")
	}
}

// --- DebugTap Acceptance Tests ---

func TestTUI_DebugTap_CapturesFrames(t *testing.T) {
	dt, err := tui.NewDebugTap(10, "")
	if err != nil {
		t.Fatal(err)
	}

	sess := session.New("debug-test", "test-model", "/workspace")
	m := repl.NewModel(repl.Config{
		Tools:    builtin.NewRegistry(),
		Session:  sess,
		Mode:     "agent",
		DebugTap: dt,
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := toModelPtr(m2)

	// View() should capture a frame
	_ = model.View()

	last, ok := dt.Last()
	if !ok {
		t.Fatal("DebugTap should have captured a frame from View()")
	}
	if last.Width != 80 {
		t.Fatalf("width = %d", last.Width)
	}
	if last.Role != "gensec" {
		t.Fatalf("role = %q, want gensec", last.Role)
	}
}

func TestTUI_DebugTap_Nil_NoPanic(t *testing.T) {
	// With nil DebugTap, View() should not panic
	m := testTUIModel(t, "agent")
	view := m.View()
	if view == "" {
		t.Fatal("view should not be empty")
	}
}

func TestTUI_DebugTap_NotStartedWithoutFlag(t *testing.T) {
	// Creating a DebugTap does NOT start the HTTP server.
	// ServeHTTP must be called explicitly (--live-debug).
	dt, err := tui.NewDebugTap(10, "")
	if err != nil {
		t.Fatal(err)
	}
	defer dt.Close() //nolint:errcheck

	// Just capturing — no server running. This is --debug-tap without --live-debug.
	dt.Capture("frame", "input", "gensec", 80, 24)
	last, ok := dt.Last()
	if !ok || last.Frame != "frame" {
		t.Fatal("capture should work without server")
	}
}
