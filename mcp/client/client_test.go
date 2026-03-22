package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/dpopsuev/djinn/djinnlog"
)

// mockMCPHandler handles JSON-RPC requests for testing.
func mockMCPHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Logf("bad request: %v", err)
			return
		}

		var result any
		switch req.Method {
		case "initialize":
			result = initializeResult{ProtocolVersion: "2024-11-05"}
		case "notifications/initialized":
			result = struct{}{}
		case "tools/list":
			result = toolsListResult{
				Tools: []ToolDef{
					{
						Name:        "artifact",
						Description: "Manage work artifacts",
						InputSchema: json.RawMessage(`{"type":"object","properties":{"action":{"type":"string"}}}`),
					},
					{
						Name:        "graph",
						Description: "Navigate relationships",
						InputSchema: json.RawMessage(`{"type":"object","properties":{"action":{"type":"string"}}}`),
					},
				},
			}
		case "tools/call":
			var params toolCallParams
			raw, _ := json.Marshal(req.Params)
			json.Unmarshal(raw, &params)
			result = toolCallResult{
				Content: []contentBlock{
					{Type: "text", Text: fmt.Sprintf("called %s", params.Name)},
				},
			}
		default:
			result = struct{}{}
		}

		resultJSON, _ := json.Marshal(result)
		resp := jsonRPCResponse{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Result:  resultJSON,
		}
		json.NewEncoder(w).Encode(resp)
	}
}

func TestConnectHTTP(t *testing.T) {
	srv := httptest.NewServer(mockMCPHandler(t))
	defer srv.Close()

	c := New(djinnlog.Nop())
	defer c.Close()

	if err := c.ConnectHTTP(context.Background(), "scribe", srv.URL); err != nil {
		t.Fatalf("ConnectHTTP: %v", err)
	}

	names := c.ServerNames()
	if len(names) != 1 || names[0] != "scribe" {
		t.Fatalf("names = %v", names)
	}
}

func TestTools_ListFromServer(t *testing.T) {
	srv := httptest.NewServer(mockMCPHandler(t))
	defer srv.Close()

	c := New(djinnlog.Nop())
	defer c.Close()
	c.ConnectHTTP(context.Background(), "scribe", srv.URL)

	tools := c.Tools()
	if len(tools) != 2 {
		t.Fatalf("tools = %d, want 2", len(tools))
	}
	if tools[0].Name != "artifact" {
		t.Fatalf("tool[0] = %q", tools[0].Name)
	}
}

func TestTools_MultipleServers(t *testing.T) {
	srv1 := httptest.NewServer(mockMCPHandler(t))
	defer srv1.Close()
	srv2 := httptest.NewServer(mockMCPHandler(t))
	defer srv2.Close()

	c := New(djinnlog.Nop())
	defer c.Close()
	c.ConnectHTTP(context.Background(), "scribe", srv1.URL)
	c.ConnectHTTP(context.Background(), "limes", srv2.URL)

	tools := c.Tools()
	if len(tools) != 4 { // 2 from each
		t.Fatalf("tools = %d, want 4", len(tools))
	}
}

func TestCall_Success(t *testing.T) {
	srv := httptest.NewServer(mockMCPHandler(t))
	defer srv.Close()

	c := New(djinnlog.Nop())
	defer c.Close()
	c.ConnectHTTP(context.Background(), "scribe", srv.URL)

	result, err := c.Call(context.Background(), "scribe", "artifact", json.RawMessage(`{"action":"list"}`))
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result != "called artifact" {
		t.Fatalf("result = %q", result)
	}
}

func TestCall_ServerNotFound(t *testing.T) {
	c := New(djinnlog.Nop())
	_, err := c.Call(context.Background(), "nonexistent", "tool", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMCPTools_Adapter(t *testing.T) {
	srv := httptest.NewServer(mockMCPHandler(t))
	defer srv.Close()

	c := New(djinnlog.Nop())
	defer c.Close()
	c.ConnectHTTP(context.Background(), "scribe", srv.URL)

	tools := c.MCPTools()
	if len(tools) != 2 {
		t.Fatalf("MCPTools = %d", len(tools))
	}

	tool := tools[0]
	if tool.Name() != "mcp__scribe__artifact" {
		t.Fatalf("Name() = %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Fatal("Description should not be empty")
	}
	if tool.InputSchema() == nil {
		t.Fatal("InputSchema should not be nil")
	}

	// Execute
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"list"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "called artifact" {
		t.Fatalf("result = %q", result)
	}
}

func TestClose(t *testing.T) {
	srv := httptest.NewServer(mockMCPHandler(t))
	defer srv.Close()

	c := New(djinnlog.Nop())
	c.ConnectHTTP(context.Background(), "scribe", srv.URL)

	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if len(c.ServerNames()) != 0 {
		t.Fatal("servers should be empty after Close")
	}
}

// --- Config tests ---

func TestLoadMCPConfig_DjinnYAML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "djinn.yaml"), []byte(`
mcp:
  scribe:
    url: "http://localhost:8080/"
  lex:
    command: misbah-lex
    args: ["serve"]
`), 0644)

	configs := LoadMCPConfig(dir, t.TempDir())
	if len(configs) < 2 {
		t.Fatalf("configs = %d, want >= 2", len(configs))
	}
	if !configs["scribe"].IsHTTP() {
		t.Fatal("scribe should be HTTP")
	}
	if configs["lex"].IsHTTP() {
		t.Fatal("lex should be stdio")
	}
}

func TestLoadMCPConfig_CursorMCP(t *testing.T) {
	home := t.TempDir()
	cursorDir := filepath.Join(home, ".cursor")
	os.MkdirAll(cursorDir, 0755)
	os.WriteFile(filepath.Join(cursorDir, "mcp.json"), []byte(`{
		"mcpServers": {
			"scribe": {"url": "http://localhost:8080/"},
			"limes": {"url": "http://localhost:8083/"}
		}
	}`), 0644)

	t.Setenv("HOME", home)
	configs := LoadMCPConfig(t.TempDir(), t.TempDir())
	if _, ok := configs["scribe"]; !ok {
		t.Fatal("scribe not found from cursor config")
	}
}

func TestLoadMCPConfig_DjinnOverridesCursor(t *testing.T) {
	home := t.TempDir()
	cursorDir := filepath.Join(home, ".cursor")
	os.MkdirAll(cursorDir, 0755)
	os.WriteFile(filepath.Join(cursorDir, "mcp.json"), []byte(`{
		"mcpServers": {"scribe": {"url": "http://cursor:8080/"}}
	}`), 0644)

	workDir := t.TempDir()
	os.WriteFile(filepath.Join(workDir, "djinn.yaml"), []byte(`
mcp:
  scribe:
    url: "http://djinn:9090/"
`), 0644)

	t.Setenv("HOME", home)
	configs := LoadMCPConfig(workDir, t.TempDir())
	if configs["scribe"].URL != "http://djinn:9090/" {
		t.Fatalf("scribe URL = %q, want djinn override", configs["scribe"].URL)
	}
}

func TestLoadMCPConfig_NoFiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // prevent reading real ~/.cursor/mcp.json
	configs := LoadMCPConfig(t.TempDir(), t.TempDir())
	if len(configs) != 0 {
		t.Fatalf("should be empty with no config files, got %d", len(configs))
	}
}
