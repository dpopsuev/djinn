package builtin

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/dpopsuev/djinn/djinnlog"
	mcpclient "github.com/dpopsuev/djinn/mcp/client"
	mcptest "github.com/dpopsuev/djinn/testkit/mcp"
)

// setupMockMCPClient creates an MCP client connected to a mock server
// with the given tools registered.
func setupMockMCPClient(t *testing.T, tools map[string]string) (*mcpclient.Client, *mcptest.MockMCPServer) {
	t.Helper()
	mock := mcptest.NewMockMCPServer()
	for name, result := range tools {
		mock.RegisterTool(name, "Mock "+name, result)
	}
	srv := httptest.NewServer(mock.HTTPHandler())
	t.Cleanup(srv.Close)

	client := mcpclient.New(djinnlog.Nop())
	t.Cleanup(func() { client.Close() })
	if err := client.ConnectHTTP(context.Background(), "testserver", srv.URL); err != nil {
		t.Fatalf("ConnectHTTP: %v", err)
	}
	return client, mock
}

func TestCompositeExecutor_BuiltinFallback(t *testing.T) {
	registry := NewRegistry()
	// No MCP client — nil means all calls fall through to builtin.
	composite := NewCompositeExecutor(registry, nil, djinnlog.Nop())

	// All builtin tool names should be present.
	names := composite.Names()
	found := make(map[string]bool)
	for _, n := range names {
		found[n] = true
	}
	if !found["Read"] {
		t.Fatal("expected Read in composite names")
	}
	if !found["Bash"] {
		t.Fatal("expected Bash in composite names")
	}
	if len(composite.Overrides()) != 0 {
		t.Fatalf("overrides should be empty with nil MCP, got %v", composite.Overrides())
	}
}

func TestCompositeExecutor_MCPOverride(t *testing.T) {
	registry := NewRegistry()
	// Register an MCP tool with the same name as a builtin: "Read".
	mcpClient, _ := setupMockMCPClient(t, map[string]string{
		"Read": "mcp-read-result",
	})

	composite := NewCompositeExecutor(registry, mcpClient, djinnlog.Nop())

	// "Read" should be overridden.
	overrides := composite.Overrides()
	if overrides["Read"] != "testserver" {
		t.Fatalf("Read override = %q, want testserver", overrides["Read"])
	}

	// Execute "Read" should route to MCP.
	result, err := composite.Execute(context.Background(), "Read", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute Read: %v", err)
	}
	if result != "mcp-read-result" {
		t.Fatalf("result = %q, want mcp-read-result", result)
	}
}

func TestCompositeExecutor_Available_MergesTools(t *testing.T) {
	registry := NewRegistry()
	mcpClient, _ := setupMockMCPClient(t, map[string]string{
		"custom_tool": "custom-result",
	})

	composite := NewCompositeExecutor(registry, mcpClient, djinnlog.Nop())

	names := composite.Names()
	sort.Strings(names)

	// Should contain builtin tools.
	found := make(map[string]bool)
	for _, n := range names {
		found[n] = true
	}
	if !found["Bash"] {
		t.Fatal("missing builtin Bash")
	}
	if !found["Edit"] {
		t.Fatal("missing builtin Edit")
	}
	// Should contain MCP tool with prefix.
	if !found["mcp__testserver__custom_tool"] {
		t.Fatalf("missing MCP tool mcp__testserver__custom_tool in %v", names)
	}

	// All() should include both.
	allTools := composite.All()
	allNames := make(map[string]bool)
	for _, tool := range allTools {
		allNames[tool.Name()] = true
	}
	if !allNames["Bash"] {
		t.Fatal("All() missing Bash")
	}
	if !allNames["mcp__testserver__custom_tool"] {
		t.Fatal("All() missing mcp__testserver__custom_tool")
	}
}

func TestCompositeExecutor_OverriddenBuiltinExcludedFromAll(t *testing.T) {
	registry := NewRegistry()
	// Override "Read" via MCP.
	mcpClient, _ := setupMockMCPClient(t, map[string]string{
		"Read": "mcp-read",
	})

	composite := NewCompositeExecutor(registry, mcpClient, djinnlog.Nop())

	// All() should NOT contain the builtin Read (it's overridden).
	allTools := composite.All()
	for _, tool := range allTools {
		// The builtin Read has Name() == "Read". The MCP Read has
		// Name() == "mcp__testserver__Read".
		if tool.Name() == "Read" {
			t.Fatal("overridden builtin Read should be excluded from All()")
		}
	}

	// But the MCP version should be present.
	foundMCP := false
	for _, tool := range allTools {
		if tool.Name() == "mcp__testserver__Read" {
			foundMCP = true
		}
	}
	if !foundMCP {
		t.Fatal("MCP Read should be in All()")
	}
}

func TestCompositeExecutor_NilMCPClient(t *testing.T) {
	registry := NewRegistry()
	composite := NewCompositeExecutor(registry, nil, djinnlog.Nop())

	// Should work fine with nil MCP client.
	names := composite.Names()
	if len(names) == 0 {
		t.Fatal("should have builtin tools")
	}

	all := composite.All()
	if len(all) == 0 {
		t.Fatal("All() should return builtin tools")
	}
}

func TestCompositeExecutor_RawMCPNameDispatch(t *testing.T) {
	// "artifact" is NOT a builtin tool, but it IS an MCP tool.
	// Execute("artifact") should route to MCP via raw name dispatch.
	registry := NewRegistry()
	mcpClient, _ := setupMockMCPClient(t, map[string]string{
		"artifact": "artifact list result",
	})

	composite := NewCompositeExecutor(registry, mcpClient, djinnlog.Nop())

	// No overrides — "artifact" isn't a builtin.
	if len(composite.Overrides()) != 0 {
		t.Fatalf("should have no overrides, got %v", composite.Overrides())
	}

	// But Execute("artifact") should still work via raw name dispatch.
	result, err := composite.Execute(context.Background(), "artifact", json.RawMessage(`{"action":"list"}`))
	if err != nil {
		t.Fatalf("Execute artifact: %v", err)
	}
	if result != "artifact list result" {
		t.Fatalf("result = %q", result)
	}
}

func TestAutoConnect_WithMockServer(t *testing.T) {
	// Simulate the full auto-connect flow: MockMCPServer → HTTP → Client → CompositeExecutor.
	mock := mcptest.NewMockMCPServer()
	mock.RegisterTool("artifact", "Manage artifacts", "artifact list result")
	mock.RegisterTool("graph", "Navigate graph", "graph result")

	srv := httptest.NewServer(mock.HTTPHandler())
	defer srv.Close()

	client := mcpclient.New(djinnlog.Nop())
	defer client.Close()

	if err := client.ConnectHTTP(context.Background(), "scribe", srv.URL); err != nil {
		t.Fatalf("ConnectHTTP: %v", err)
	}

	// Verify tools are listed.
	serverTools, err := client.ServerTools("scribe")
	if err != nil {
		t.Fatalf("ServerTools: %v", err)
	}
	if len(serverTools) != 2 {
		t.Fatalf("server tools = %d, want 2", len(serverTools))
	}

	// Build composite.
	registry := NewRegistry()
	composite := NewCompositeExecutor(registry, client, djinnlog.Nop())

	// Merged names should include MCP tools.
	names := composite.Names()
	found := make(map[string]bool)
	for _, n := range names {
		found[n] = true
	}
	if !found["mcp__scribe__artifact"] {
		t.Fatalf("missing mcp__scribe__artifact in %v", names)
	}
	if !found["mcp__scribe__graph"] {
		t.Fatalf("missing mcp__scribe__graph in %v", names)
	}

	// Call an MCP tool.
	result, err := client.Call(context.Background(), "scribe", "artifact", json.RawMessage(`{"action":"list"}`))
	if err != nil {
		t.Fatalf("Call artifact: %v", err)
	}
	if result != "artifact list result" {
		t.Fatalf("result = %q", result)
	}

	// Verify calls were recorded.
	calls := mock.Calls()
	if len(calls) != 1 || calls[0].Name != "artifact" {
		t.Fatalf("calls = %v", calls)
	}
}
