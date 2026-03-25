package repl

import (
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/djinnlog"
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
	if sess.Mode != "plan" {
		t.Fatalf("session mode = %q, want plan", sess.Mode)
	}
	if result.ModeChange != "plan" {
		t.Fatalf("ModeChange = %q, want plan", result.ModeChange)
	}
}

func TestExecuteCommand_Mode_InvalidMode(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/mode", Args: []string{"yolo"}}, sess)
	if !strings.Contains(result.Output, "unknown mode") {
		t.Fatalf("output = %q", result.Output)
	}
	if result.ModeChange != "" {
		t.Fatal("invalid mode should not set ModeChange")
	}
}

func TestExecuteCommand_Mode_AllModes(t *testing.T) {
	for _, mode := range []string{"ask", "plan", "agent", "auto"} {
		sess := session.New("test", "model", "/workspace")
		result := ExecuteCommand(Command{Name: "/mode", Args: []string{mode}}, sess)
		if sess.Mode != mode {
			t.Fatalf("mode %q: session.Mode = %q", mode, sess.Mode)
		}
		if result.ModeChange != mode {
			t.Fatalf("mode %q: ModeChange = %q", mode, result.ModeChange)
		}
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

func TestCommandNames_Sorted(t *testing.T) {
	names := CommandNames()
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Fatalf("not sorted: %q before %q", names[i-1], names[i])
		}
	}
}

func TestCommandNames_Count(t *testing.T) {
	names := CommandNames()
	if len(names) < 20 {
		t.Fatalf("expected 20+ commands, got %d", len(names))
	}
	// Spot-check key commands are present.
	has := make(map[string]bool)
	for _, n := range names {
		has[n] = true
	}
	for _, want := range []string{"/help", "/exit", "/model", "/workspace", "/diff", "/config"} {
		if !has[want] {
			t.Errorf("missing command: %s", want)
		}
	}
}

// --- ParseCommand edge cases ---

func TestParseCommand_MultipleArgs(t *testing.T) {
	cmd, ok := ParseCommand("/role create mybot agent")
	if !ok {
		t.Fatal("expected command")
	}
	if cmd.Name != "/role" {
		t.Fatalf("Name = %q", cmd.Name)
	}
	if len(cmd.Args) != 3 {
		t.Fatalf("Args = %v, want 3 elements", cmd.Args)
	}
	if cmd.Args[0] != "create" || cmd.Args[1] != "mybot" || cmd.Args[2] != "agent" {
		t.Fatalf("Args = %v", cmd.Args)
	}
}

func TestParseCommand_SlashOnly(t *testing.T) {
	cmd, ok := ParseCommand("/")
	if !ok {
		t.Fatal("/ should parse as command")
	}
	if cmd.Name != "/" {
		t.Fatalf("Name = %q", cmd.Name)
	}
}

// --- ExecuteCommand: cost ---

func TestExecuteCommand_Cost(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/cost"}, sess)
	if !strings.Contains(result.Output, "tokens") {
		t.Fatalf("output = %q", result.Output)
	}
}

// --- ExecuteCommand: help ---

func TestExecuteCommand_Help(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/help"}, sess)
	if result.Output == "" {
		t.Fatal("help should produce output")
	}
	if !strings.Contains(result.Output, "/model") {
		t.Fatal("help should list /model")
	}
	if !strings.Contains(result.Output, "/exit") {
		t.Fatal("help should list /exit")
	}
}

// --- ExecuteCommand: copy with empty history ---

func TestExecuteCommand_Copy_Empty(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/copy"}, sess)
	if !strings.Contains(result.Output, "nothing to copy") {
		t.Fatalf("output = %q", result.Output)
	}
}

// --- ExecuteCommand: copy with only user messages ---

func TestExecuteCommand_Copy_NoAssistant(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.Append(session.Entry{Role: "user", Content: "question"})
	result := ExecuteCommand(Command{Name: "/copy"}, sess)
	if !strings.Contains(result.Output, "no assistant response") {
		t.Fatalf("output = %q", result.Output)
	}
}

// --- ExecuteCommand: compact with short history ---

func TestExecuteCommand_Compact_Short(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.Append(session.Entry{Content: "one message"})
	result := ExecuteCommand(Command{Name: "/compact"}, sess)
	if !strings.Contains(result.Output, "nothing to compact") {
		t.Fatalf("output = %q, want 'nothing to compact'", result.Output)
	}
}

// --- ExecuteCommand: permissions with different modes ---

func TestExecuteCommand_Permissions_AutoMode(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.Mode = "auto"
	result := ExecuteCommand(Command{Name: "/permissions"}, sess)
	if !strings.Contains(result.Output, "auto-approve") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_Permissions_AskMode(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.Mode = "ask"
	result := ExecuteCommand(Command{Name: "/permissions"}, sess)
	if !strings.Contains(result.Output, "none") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_Permissions_PlanMode(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.Mode = "plan"
	result := ExecuteCommand(Command{Name: "/permissions"}, sess)
	if !strings.Contains(result.Output, "none") {
		t.Fatalf("output = %q", result.Output)
	}
}

// --- ExecuteCommand: workspace ---

func TestExecuteCommand_Workspace_Show(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.Workspace = "aeon"
	sess.WorkDirs = []string{"/repo1", "/repo2"}
	result := ExecuteCommand(Command{Name: "/workspace"}, sess)
	if !strings.Contains(result.Output, "aeon") {
		t.Fatalf("output = %q", result.Output)
	}
	if !strings.Contains(result.Output, "/repo1") {
		t.Fatalf("output should list repos: %q", result.Output)
	}
}

func TestExecuteCommand_Workspace_Show_Ephemeral(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.Workspace = ""
	result := ExecuteCommand(Command{Name: "/workspace"}, sess)
	if !strings.Contains(result.Output, "ephemeral") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_Workspace_Repos(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.WorkDirs = []string{"/repo1", "/repo2"}
	result := ExecuteCommand(Command{Name: "/workspace", Args: []string{"repos"}}, sess)
	if !strings.Contains(result.Output, "/repo1") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_Workspace_Add(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/workspace", Args: []string{"add", "/new/repo"}}, sess)
	if !strings.Contains(result.Output, "added") {
		t.Fatalf("output = %q", result.Output)
	}
	found := false
	for _, d := range sess.WorkDirs {
		if d == "/new/repo" {
			found = true
		}
	}
	if !found {
		t.Fatal("repo not added to WorkDirs")
	}
}

func TestExecuteCommand_Workspace_Save(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.Workspace = "myws"
	result := ExecuteCommand(Command{Name: "/workspace", Args: []string{"save"}}, sess)
	if !strings.Contains(result.Output, "saved") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_Workspace_InvalidSubcommand(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/workspace", Args: []string{"bogus"}}, sess)
	if !strings.Contains(result.Output, "usage") {
		t.Fatalf("output = %q", result.Output)
	}
}

// --- ExecuteCommand: workspace-repos ---

func TestExecuteCommand_WorkspaceRepos_Empty(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.WorkDirs = nil
	result := ExecuteCommand(Command{Name: "/workspace-repos"}, sess)
	if !strings.Contains(result.Output, "no repos") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_WorkspaceRepos_WithRepos(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.WorkDirs = []string{"/a", "/b"}
	result := ExecuteCommand(Command{Name: "/workspace-repos"}, sess)
	if !strings.Contains(result.Output, "/a") || !strings.Contains(result.Output, "/b") {
		t.Fatalf("output = %q", result.Output)
	}
}

// --- ExecuteCommand: workspace-save ---

func TestExecuteCommand_WorkspaceSave_NoName(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.Workspace = ""
	result := ExecuteCommand(Command{Name: "/workspace-save"}, sess)
	if !strings.Contains(result.Output, "name") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_WorkspaceSave_WithName(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.Workspace = "myws"
	result := ExecuteCommand(Command{Name: "/workspace-save"}, sess)
	if !strings.Contains(result.Output, "saved") {
		t.Fatalf("output = %q", result.Output)
	}
}

// --- ExecuteCommand: workspace-add ---

func TestExecuteCommand_WorkspaceAdd_NoArgs(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/workspace-add"}, sess)
	if !strings.Contains(result.Output, "usage") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_WorkspaceAdd_WithPath(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/workspace-add", Args: []string{"/new/path"}}, sess)
	if !strings.Contains(result.Output, "added") {
		t.Fatalf("output = %q", result.Output)
	}
}

// --- ExecuteCommand: workspace-switch ---

func TestExecuteCommand_WorkspaceSwitch_NoArgs(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/workspace-switch"}, sess)
	if !strings.Contains(result.Output, "usage") {
		t.Fatalf("output = %q", result.Output)
	}
}

// --- ExecuteCommand: config ---

func TestExecuteCommand_Config_Show(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.Driver = "claude"
	result := ExecuteCommand(Command{Name: "/config"}, sess)
	if !strings.Contains(result.Output, "claude") {
		t.Fatalf("output = %q", result.Output)
	}
	if !strings.Contains(result.Output, "model") {
		t.Fatalf("output should show model: %q", result.Output)
	}
}

func TestExecuteCommand_Config_EmptyMode(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	sess.Mode = ""
	result := ExecuteCommand(Command{Name: "/config"}, sess)
	if !strings.Contains(result.Output, "agent") {
		t.Fatalf("empty mode should default to agent: %q", result.Output)
	}
}

func TestExecuteCommand_Config_Save_Subcommand(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	tmpFile := t.TempDir() + "/djinn-test.yaml"
	result := ExecuteCommand(Command{Name: "/config", Args: []string{"save", tmpFile}}, sess)
	if !strings.Contains(result.Output, "saved") {
		t.Fatalf("output = %q", result.Output)
	}
}

// --- ExecuteCommand: config-save ---

func TestExecuteCommand_ConfigSave_Default(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	// Use temp dir to avoid polluting cwd
	tmpFile := t.TempDir() + "/djinn.yaml"
	result := ExecuteCommand(Command{Name: "/config-save", Args: []string{tmpFile}}, sess)
	if !strings.Contains(result.Output, "saved") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_ConfigSave_BadPath(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/config-save", Args: []string{"/nonexistent/dir/cfg.yaml"}}, sess)
	if !strings.Contains(result.Output, "error") {
		t.Fatalf("output = %q, want error", result.Output)
	}
}

// --- ExecuteCommand: log (with globalRing) ---

func TestExecuteCommand_Log_NoRing(t *testing.T) {
	// globalRing should be nil by default in tests
	oldRing := globalRing
	globalRing = nil
	defer func() { globalRing = oldRing }()

	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/log"}, sess)
	if !strings.Contains(result.Output, "not initialized") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_Log_WithRing_Empty(t *testing.T) {
	oldRing := globalRing
	globalRing = djinnlog.NewRingHandler(100)
	defer func() { globalRing = oldRing }()

	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/log"}, sess)
	if !strings.Contains(result.Output, "no log entries") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_Log_WithRing_Entries(t *testing.T) {
	oldRing := globalRing
	ring := djinnlog.NewRingHandler(100)
	logger := slog.New(ring)
	logger.Info("test message")
	globalRing = ring
	defer func() { globalRing = oldRing }()

	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/log"}, sess)
	if !strings.Contains(result.Output, "test message") {
		t.Fatalf("output = %q", result.Output)
	}
}

func TestExecuteCommand_Log_CountArg(t *testing.T) {
	oldRing := globalRing
	ring := djinnlog.NewRingHandler(100)
	logger := slog.New(ring)
	for i := 0; i < 30; i++ {
		logger.Info(fmt.Sprintf("msg-%d", i))
	}
	globalRing = ring
	defer func() { globalRing = oldRing }()

	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/log", Args: []string{"5"}}, sess)
	// Should show only last 5 entries
	lines := strings.Split(strings.TrimSpace(result.Output), "\n")
	if len(lines) > 5 {
		t.Fatalf("expected at most 5 lines, got %d", len(lines))
	}
}

func TestExecuteCommand_Log_LevelFilter(t *testing.T) {
	oldRing := globalRing
	ring := djinnlog.NewRingHandler(100)
	logger := slog.New(ring)
	logger.Info("info msg")
	logger.Error("error msg")
	globalRing = ring
	defer func() { globalRing = oldRing }()

	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/log", Args: []string{"error"}}, sess)
	if !strings.Contains(result.Output, "error msg") {
		t.Fatalf("output = %q", result.Output)
	}
	if strings.Contains(result.Output, "info msg") {
		t.Fatal("error filter should exclude info messages")
	}
}

// --- ExecuteCommand: review ---

func TestExecuteCommand_Review(t *testing.T) {
	sess := session.New("test", "model", "/workspace")
	result := ExecuteCommand(Command{Name: "/review"}, sess)
	if result.Output == "" {
		t.Fatal("review should produce output")
	}
}

// --- ExecuteCommand: sessions ---

func TestExecuteCommand_Sessions(t *testing.T) {
	sess := session.New("test", "model", t.TempDir())
	result := ExecuteCommand(Command{Name: "/sessions"}, sess)
	// Either lists sessions or says none found — should not crash
	if result.Output == "" {
		t.Fatal("sessions should produce output")
	}
}

// --- All output modes for /output ---

func TestExecuteCommand_Output_AllModes(t *testing.T) {
	for _, mode := range []string{"streaming", "chunked", "follow", "static"} {
		sess := session.New("test", "model", "/workspace")
		result := ExecuteCommand(Command{Name: "/output", Args: []string{mode}}, sess)
		if !strings.Contains(result.Output, mode) {
			t.Fatalf("mode %s: output = %q", mode, result.Output)
		}
	}
}
