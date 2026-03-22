package claude

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/mcp"
)

func TestCodeDriver_InterfaceSatisfaction(t *testing.T) {
	var _ driver.Driver = (*CodeDriver)(nil)
}

func TestCodeDriver_Lifecycle(t *testing.T) {
	exec := func(ctx context.Context, name string, cmd []string, timeout int64) (ExecResult, error) {
		resp, _ := json.Marshal(claudeJSONResponse{Result: "done"})
		return ExecResult{ExitCode: 0, Stdout: string(resp)}, nil
	}

	d := NewCodeDriver(driver.DriverConfig{Model: "claude-sonnet-4-6"}, exec)
	ctx := context.Background()

	if err := d.Start(ctx, "sandbox-1"); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Double start
	if err := d.Start(ctx, "sandbox-2"); !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("expected ErrAlreadyStarted, got %v", err)
	}

	if err := d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "fix the bug"}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	msg := <-d.Recv(ctx)
	if msg.Role != driver.RoleAssistant {
		t.Fatalf("Role = %q, want %q", msg.Role, driver.RoleAssistant)
	}
	if msg.Content != "done" {
		t.Fatalf("Content = %q, want %q", msg.Content, "done")
	}

	if err := d.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestCodeDriver_BuildCommand(t *testing.T) {
	d := NewCodeDriver(
		driver.DriverConfig{Model: "claude-opus-4-6"},
		nil,
		WithSystemPrompt("You are a Go expert"),
	)

	cmd := d.buildCommand("fix the auth bug")

	hasP := false
	hasModel := false
	hasSystem := false
	for i, arg := range cmd {
		if arg == flagPrint && i+1 < len(cmd) && cmd[i+1] == "fix the auth bug" {
			hasP = true
		}
		if arg == flagModel && i+1 < len(cmd) && cmd[i+1] == "claude-opus-4-6" {
			hasModel = true
		}
		if arg == flagAppendSystem && i+1 < len(cmd) && cmd[i+1] == "You are a Go expert" {
			hasSystem = true
		}
	}
	if !hasP {
		t.Fatal("missing -p flag with prompt")
	}
	if !hasModel {
		t.Fatal("missing --model flag")
	}
	if !hasSystem {
		t.Fatal("missing --append-system-prompt flag")
	}
}

func TestCodeDriver_BuildCommand_WithMCP(t *testing.T) {
	servers := []mcp.Server{
		{Name: "scribe", Type: mcp.TypeHTTP, URL: "http://localhost:8080/"},
	}

	d := NewCodeDriver(
		driver.DriverConfig{},
		nil,
		WithMCPServers(servers),
	)

	cmd := d.buildCommand("do something")

	hasMCPConfig := false
	for i, arg := range cmd {
		if arg == flagMCPConfig && i+1 < len(cmd) {
			hasMCPConfig = true
			// Verify the file exists
			path := cmd[i+1]
			if path == "" {
				t.Fatal("MCP config path is empty")
			}
		}
	}
	if !hasMCPConfig {
		t.Fatal("missing --mcp-config flag when servers configured")
	}

	// Clean up
	d.Stop(context.Background())
}

func TestCodeDriver_ParseResponse_JSON(t *testing.T) {
	d := NewCodeDriver(driver.DriverConfig{}, nil)

	resp, _ := json.Marshal(claudeJSONResponse{
		Result:    "fixed the bug",
		SessionID: "sess-123",
	})

	content := d.parseResponse(ExecResult{ExitCode: 0, Stdout: string(resp)})
	if content != "fixed the bug" {
		t.Fatalf("parsed content = %q, want %q", content, "fixed the bug")
	}
}

func TestCodeDriver_ParseResponse_NonZeroExit(t *testing.T) {
	d := NewCodeDriver(driver.DriverConfig{}, nil)

	content := d.parseResponse(ExecResult{ExitCode: 1, Stderr: "API key invalid"})
	if content != "claude exit code 1: API key invalid" {
		t.Fatalf("error content = %q", content)
	}
}

func TestCodeDriver_ParseResponse_RawText(t *testing.T) {
	d := NewCodeDriver(driver.DriverConfig{}, nil)

	content := d.parseResponse(ExecResult{ExitCode: 0, Stdout: "plain text response"})
	if content != "plain text response" {
		t.Fatalf("raw content = %q, want %q", content, "plain text response")
	}
}

func TestCodeDriver_ExecError(t *testing.T) {
	exec := func(ctx context.Context, name string, cmd []string, timeout int64) (ExecResult, error) {
		return ExecResult{}, errors.New("container not running")
	}

	d := NewCodeDriver(driver.DriverConfig{}, exec)
	ctx := context.Background()
	d.Start(ctx, "sandbox-1")

	d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "test"})

	msg := <-d.Recv(ctx)
	if msg.Content != "exec error: container not running" {
		t.Fatalf("error message = %q", msg.Content)
	}
}

func TestCodeDriver_SendAfterStop(t *testing.T) {
	d := NewCodeDriver(driver.DriverConfig{}, nil)
	ctx := context.Background()
	d.Start(ctx, "sb")
	d.Stop(ctx)

	if err := d.Send(ctx, driver.Message{}); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("expected ErrNotRunning, got %v", err)
	}
}
