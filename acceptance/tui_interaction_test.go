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
	"github.com/dpopsuev/djinn/tools/builtin"
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

	m2, _ := m.Update(repl.TextMsg("hello world"))
	model := toModelPtr(m2)

	if model.StreamBufString() != "hello world" {
		t.Fatalf("streamBuf = %q", model.StreamBufString())
	}

	m3, _ := model.Update(repl.TickMsg(time.Now()))
	model2 := toModelPtr(m3)
	if model2.StreamBufString() != "" {
		t.Fatal("buffer should be empty after tick flush")
	}
}

func TestTUI_ToolApprovalInAgentMode(t *testing.T) {
	m := testTUIModel(t, "agent")
	m.SetState(repl.StateStreaming)
	m2, _ := m.Update(repl.ToolCallMsg{Call: driver.ToolCall{
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
	m2, _ := m.Update(repl.ToolCallMsg{Call: driver.ToolCall{
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
