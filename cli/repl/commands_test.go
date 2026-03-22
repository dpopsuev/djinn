package repl

import (
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/session"
)

func TestParseCommand_Valid(t *testing.T) {
	cmd, ok := ParseCommand("/model claude-opus-4-6")
	if !ok {
		t.Fatal("expected command")
	}
	if cmd.Name != "/model" {
		t.Fatalf("Name = %q", cmd.Name)
	}
	if len(cmd.Args) != 1 || cmd.Args[0] != "claude-opus-4-6" {
		t.Fatalf("Args = %v", cmd.Args)
	}
}

func TestParseCommand_NoArgs(t *testing.T) {
	cmd, ok := ParseCommand("/help")
	if !ok {
		t.Fatal("expected command")
	}
	if cmd.Name != "/help" {
		t.Fatalf("Name = %q", cmd.Name)
	}
	if len(cmd.Args) != 0 {
		t.Fatalf("Args = %v, want empty", cmd.Args)
	}
}

func TestParseCommand_NotCommand(t *testing.T) {
	_, ok := ParseCommand("hello world")
	if ok {
		t.Fatal("non-slash input should not parse as command")
	}
}

func TestExecuteCommand_Exit(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/exit"}, sess)
	if !result.Exit {
		t.Fatal("expected Exit = true")
	}
}

func TestExecuteCommand_Quit(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/quit"}, sess)
	if !result.Exit {
		t.Fatal("expected Exit = true")
	}
}

func TestExecuteCommand_Clear(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.Append(session.Entry{Content: "old message"})

	result := ExecuteCommand(Command{Name: "/clear"}, sess)
	if !result.Cleared {
		t.Fatal("expected Cleared = true")
	}
	if sess.History.Len() != 0 {
		t.Fatalf("history should be empty, got %d", sess.History.Len())
	}
}

func TestExecuteCommand_ModelShow(t *testing.T) {
	sess := session.New("test", "claude-sonnet-4-6", "/workspace")
	result := ExecuteCommand(Command{Name: "/model"}, sess)
	if result.Output != "current model: claude-sonnet-4-6" {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_ModelSwitch(t *testing.T) {
	sess := session.New("test", "claude-sonnet-4-6", "/workspace")
	result := ExecuteCommand(Command{Name: "/model", Args: []string{"claude-opus-4-6"}}, sess)
	if sess.Model != "claude-opus-4-6" {
		t.Fatalf("model = %q, want claude-opus-4-6", sess.Model)
	}
	if result.Output != "model set to claude-opus-4-6" {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_Status(t *testing.T) {
	sess := session.New("sess-1", "claude", "/workspace")
	result := ExecuteCommand(Command{Name: "/status"}, sess)
	if result.Output == "" {
		t.Fatal("status output should not be empty")
	}
}

func TestExecuteCommand_Unknown(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/bogus"}, sess)
	if result.Output == "" {
		t.Fatal("unknown command should produce output")
	}
}

func TestExecuteCommand_Compact(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	for range 10 {
		sess.Append(session.Entry{Content: "padding message"})
	}
	result := ExecuteCommand(Command{Name: "/compact"}, sess)
	if result.Output == "" {
		t.Fatal("compact should produce output")
	}
	// History should be shorter after compaction
	if sess.History.Len() >= 10 {
		t.Fatalf("history = %d, should be compacted", sess.History.Len())
	}
}

func TestExecuteCommand_Diff(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/diff"}, sess)
	// Should not crash, may produce empty diff or git output
	_ = result
}

func TestExecuteCommand_Mode_Show(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/mode"}, sess)
	if !strings.Contains(result.Output, "agent") {
		t.Fatalf("mode output should show current mode: %q", result.Output)
	}
}

func TestExecuteCommand_Mode_Switch(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/mode", Args: []string{"plan"}}, sess)
	if result.Output == "" {
		t.Fatal("mode switch should produce output")
	}
}

func TestExecuteCommand_Permissions(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/permissions"}, sess)
	if result.Output == "" {
		t.Fatal("permissions should produce output")
	}
}

func TestExecuteCommand_Memory(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/memory"}, sess)
	if result.Output == "" {
		t.Fatal("memory should produce output")
	}
}

func TestExecuteCommand_Mcp(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/mcp"}, sess)
	if result.Output == "" {
		t.Fatal("mcp should produce output")
	}
}

func TestExecuteCommand_Resume(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/resume"}, sess)
	if result.Output == "" {
		t.Fatal("resume should produce output")
	}
}

func TestExecuteCommand_Output_Show(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/output"}, sess)
	if !strings.Contains(result.Output, "streaming") {
		t.Fatalf("output should list modes: %q", result.Output)
	}
}

func TestExecuteCommand_Output_Set(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/output", Args: []string{"chunked"}}, sess)
	if !strings.Contains(result.Output, "chunked") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_Output_Invalid(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/output", Args: []string{"invalid"}}, sess)
	if !strings.Contains(result.Output, "unknown") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_Copy(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.Append(session.Entry{Role: "assistant", Content: "last response"})
	result := ExecuteCommand(Command{Name: "/copy"}, sess)
	if result.Output == "" {
		t.Fatal("copy should produce output")
	}
}
