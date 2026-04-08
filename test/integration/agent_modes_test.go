//go:build integration

package integration

import (
	"testing"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/repl"
	"github.com/dpopsuev/djinn/contextmgr"
	"github.com/dpopsuev/djinn/driver"
)

func TestMode_AskDisablesTools(t *testing.T) {
	if agent.ModeAsk.ToolsEnabled() {
		t.Fatal("ask mode should disable tools")
	}
}

func TestMode_PlanDisablesTools(t *testing.T) {
	if agent.ModePlan.ToolsEnabled() {
		t.Fatal("plan mode should disable tools")
	}
}

func TestMode_AgentRequiresApproval(t *testing.T) {
	if !agent.ModeAgent.ToolsEnabled() {
		t.Fatal("agent mode should enable tools")
	}
	fn := agent.ModeAgent.DefaultApprove()
	if fn(driver.ToolCall{Name: "Read"}) {
		t.Fatal("agent mode default should deny (interactive required)")
	}
}

func TestMode_AutoApprovesAll(t *testing.T) {
	if !agent.ModeAuto.ToolsEnabled() {
		t.Fatal("auto mode should enable tools")
	}
	fn := agent.ModeAuto.DefaultApprove()
	if !fn(driver.ToolCall{Name: "Bash"}) {
		t.Fatal("auto mode should approve all")
	}
}

func TestMode_SwitchViaCommand(t *testing.T) {
	sess := contextmgr.New("test", "model", "/workspace")
	result := repl.ExecuteCommand(repl.Command{Name: "/mode", Args: []string{"auto"}}, sess)
	if sess.Mode != "auto" {
		t.Fatalf("contextmgr.Mode = %q, want auto", sess.Mode)
	}
	if result.ModeChange != "auto" {
		t.Fatalf("ModeChange = %q", result.ModeChange)
	}
}

func TestMode_PersistedOnSession(t *testing.T) {
	dir := t.TempDir()
	store, _ := contextmgr.NewStore(dir)

	sess := contextmgr.New("test", "model", "/workspace")
	sess.Mode = "plan"
	store.Save(sess)

	loaded, _ := store.Load("test")
	if loaded.Mode != "plan" {
		t.Fatalf("loaded mode = %q, want plan", loaded.Mode)
	}
}

func TestMode_CycleNext(t *testing.T) {
	tests := []struct{ from, to agent.Mode }{
		{agent.ModeAsk, agent.ModePlan},
		{agent.ModePlan, agent.ModeAgent},
		{agent.ModeAgent, agent.ModeAuto},
		{agent.ModeAuto, agent.ModeAsk},
	}
	for _, tt := range tests {
		if tt.from.Next() != tt.to {
			t.Fatalf("%s.Next() = %s, want %s", tt.from, tt.from.Next(), tt.to)
		}
	}
}

func TestMode_ParseInvalid(t *testing.T) {
	_, err := agent.ParseMode("yolo")
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}
