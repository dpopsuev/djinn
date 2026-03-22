package repl

import (
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
