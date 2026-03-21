package claude

import (
	"context"
	"testing"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/tier"
)

func TestClaudeDriver_InterfaceSatisfaction(t *testing.T) {
	var _ driver.Driver = (*ClaudeDriver)(nil)
}

func TestClaudeDriver_Lifecycle(t *testing.T) {
	d := New(driver.DriverConfig{Model: "claude-opus-4-6"})
	ctx := context.Background()

	if err := d.Start(ctx, "sandbox-1"); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Double start should fail
	if err := d.Start(ctx, "sandbox-2"); err == nil {
		t.Fatal("second Start should fail")
	}

	if err := d.Send(ctx, driver.Message{Role: "user", Content: "implement auth"}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	// Read response
	ch := d.Recv(ctx)
	msg := <-ch
	if msg.Role != "assistant" {
		t.Fatalf("response Role = %q, want %q", msg.Role, "assistant")
	}
	if msg.Content != "completed: implement auth" {
		t.Fatalf("response Content = %q, want %q", msg.Content, "completed: implement auth")
	}

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Fatal("channel should be closed after completion")
	}

	if err := d.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// Send after stop should fail
	d2 := New(driver.DriverConfig{})
	d2.Start(ctx, "sb")
	d2.Stop(ctx)
	if err := d2.Send(ctx, driver.Message{}); err == nil {
		t.Fatal("Send after Stop should fail")
	}
}

func TestClaudeDriver_ContainerEnv_ModTier(t *testing.T) {
	d := New(
		driver.DriverConfig{Model: "claude-opus-4-6", MaxTokens: 4096},
		WithScope(tier.Scope{Level: tier.Mod, Name: "auth"}),
		WithClaudeMD("# Auth Module\nFix the auth bug."),
	)

	ctx := context.Background()
	d.Start(ctx, "sandbox-42")
	d.Send(ctx, driver.Message{Role: "user", Content: "fix bug"})

	env := d.ContainerEnv()

	if env.Model != "claude-opus-4-6" {
		t.Fatalf("Model = %q, want %q", env.Model, "claude-opus-4-6")
	}
	if env.Prompt != "fix bug" {
		t.Fatalf("Prompt = %q, want %q", env.Prompt, "fix bug")
	}
	if env.ClaudeMD != "# Auth Module\nFix the auth bug." {
		t.Fatalf("ClaudeMD = %q, want injected content", env.ClaudeMD)
	}
	if env.Scope.Level != tier.Mod {
		t.Fatalf("Scope.Level = %v, want Mod", env.Scope.Level)
	}
	if env.Scope.Name != "auth" {
		t.Fatalf("Scope.Name = %q, want %q", env.Scope.Name, "auth")
	}
	if env.EnvVars["DJINN_TIER"] != "mod" {
		t.Fatalf("DJINN_TIER = %q, want %q", env.EnvVars["DJINN_TIER"], "mod")
	}
	if env.EnvVars["DJINN_SCOPE"] != "auth" {
		t.Fatalf("DJINN_SCOPE = %q, want %q", env.EnvVars["DJINN_SCOPE"], "auth")
	}
	if env.EnvVars["DJINN_SANDBOX"] != "sandbox-42" {
		t.Fatalf("DJINN_SANDBOX = %q, want %q", env.EnvVars["DJINN_SANDBOX"], "sandbox-42")
	}
	if env.EnvVars["ANTHROPIC_MODEL"] != "claude-opus-4-6" {
		t.Fatalf("ANTHROPIC_MODEL = %q, want %q", env.EnvVars["ANTHROPIC_MODEL"], "claude-opus-4-6")
	}
	if env.EnvVars["CLAUDE_MAX_TOKENS"] != "4096" {
		t.Fatalf("CLAUDE_MAX_TOKENS = %q, want %q", env.EnvVars["CLAUDE_MAX_TOKENS"], "4096")
	}
}

func TestClaudeDriver_ContainerEnv_EcoTier(t *testing.T) {
	d := New(
		driver.DriverConfig{Model: "claude-sonnet-4-6"},
		WithScope(tier.Scope{Level: tier.Eco, Name: "workspace"}),
	)

	ctx := context.Background()
	d.Start(ctx, "sandbox-eco")

	env := d.ContainerEnv()
	if env.EnvVars["DJINN_TIER"] != "eco" {
		t.Fatalf("DJINN_TIER = %q, want %q", env.EnvVars["DJINN_TIER"], "eco")
	}
	if env.ClaudeMD != "" {
		t.Fatalf("ClaudeMD = %q, want empty (no injection)", env.ClaudeMD)
	}
}

func TestClaudeDriver_ContainerEnv_NoModel(t *testing.T) {
	d := New(driver.DriverConfig{})
	ctx := context.Background()
	d.Start(ctx, "sb")

	env := d.ContainerEnv()
	if _, ok := env.EnvVars["ANTHROPIC_MODEL"]; ok {
		t.Fatal("ANTHROPIC_MODEL should not be set for empty model")
	}
	if _, ok := env.EnvVars["CLAUDE_MAX_TOKENS"]; ok {
		t.Fatal("CLAUDE_MAX_TOKENS should not be set for zero value")
	}
}
