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

// --- SSE transport tests ---

func TestConnectHTTP_SSE(t *testing.T) {
	// Mock server that responds with SSE format (like real MCP HTTP servers)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		var result any
		switch req.Method {
		case "initialize":
			result = initializeResult{ProtocolVersion: "2024-11-05"}
		case "notifications/initialized":
			result = struct{}{}
		case "tools/list":
			result = toolsListResult{
				Tools: []ToolDef{
					{Name: "artifact", Description: "Manage artifacts", InputSchema: json.RawMessage(`{"type":"object"}`)},
				},
			}
		case "tools/call":
			result = toolCallResult{
				Content: []contentBlock{{Type: "text", Text: "sse result"}},
			}
		default:
			result = struct{}{}
		}

		resultJSON, _ := json.Marshal(result)
		resp := jsonRPCResponse{JSONRPC: jsonRPCVersion, ID: req.ID, Result: resultJSON}
		respJSON, _ := json.Marshal(resp)

		// Respond in SSE format
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", respJSON)
	}))
	defer srv.Close()

	c := New(djinnlog.Nop())
	defer c.Close()

	if err := c.ConnectHTTP(context.Background(), "scribe-sse", srv.URL); err != nil {
		t.Fatalf("ConnectHTTP SSE: %v", err)
	}

	tools := c.Tools()
	if len(tools) != 1 {
		t.Fatalf("tools = %d, want 1", len(tools))
	}
	if tools[0].Name != "artifact" {
		t.Fatalf("tool = %q", tools[0].Name)
	}

	// Test tool call via SSE
	result, err := c.Call(context.Background(), "scribe-sse", "artifact", json.RawMessage(`{"action":"list"}`))
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result != "sse result" {
		t.Fatalf("result = %q", result)
	}
}

// TestConnectHTTP_SSE_Chunked reproduces the real Scribe behavior:
// Transfer-Encoding: chunked + Content-Type: text/event-stream
// This is the exact format that caused "no JSON found in SSE body" in production.
func TestConnectHTTP_SSE_Chunked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		var result any
		switch req.Method {
		case "initialize":
			result = initializeResult{ProtocolVersion: "2024-11-05"}
		case "notifications/initialized":
			result = struct{}{}
		case "tools/list":
			result = toolsListResult{
				Tools: []ToolDef{
					{Name: "artifact", Description: "Manage artifacts", InputSchema: json.RawMessage(`{"type":"object"}`)},
				},
			}
		case "tools/call":
			result = toolCallResult{
				Content: []contentBlock{{Type: "text", Text: "chunked sse result"}},
			}
		default:
			result = struct{}{}
		}

		resultJSON, _ := json.Marshal(result)
		resp := jsonRPCResponse{JSONRPC: jsonRPCVersion, ID: req.ID, Result: resultJSON}
		respJSON, _ := json.Marshal(resp)

		// Respond EXACTLY like real Scribe: chunked SSE
		w.Header().Set("Content-Type", "text/event-stream")
		// Transfer-Encoding: chunked is automatic when using Flush()
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("server doesn't support flushing")
		}
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", respJSON)
		flusher.Flush()
	}))
	defer srv.Close()

	c := New(djinnlog.Nop())
	defer c.Close()

	err := c.ConnectHTTP(context.Background(), "scribe-chunked", srv.URL)
	if err != nil {
		t.Fatalf("ConnectHTTP chunked SSE: %v", err)
	}

	tools := c.Tools()
	if len(tools) != 1 {
		t.Fatalf("tools = %d, want 1", len(tools))
	}

	// Test tool call through chunked SSE
	result, err := c.Call(context.Background(), "scribe-chunked", "artifact", json.RawMessage(`{"action":"list"}`))
	if err != nil {
		t.Fatalf("Call through chunked SSE: %v", err)
	}
	if result != "chunked sse result" {
		t.Fatalf("result = %q", result)
	}
}

// TestConnectHTTP_SSE_RequiresAcceptHeader reproduces the real Scribe bug:
// Scribe returns 400 "Accept must contain both 'application/json' and 'text/event-stream'"
// when the Accept header is missing.
func TestConnectHTTP_SSE_RequiresAcceptHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mimic real Scribe: reject if Accept header doesn't contain both types
		accept := r.Header.Get("Accept")
		if accept == "" || !(contains(accept, "application/json") && contains(accept, "text/event-stream")) {
			http.Error(w, "Accept must contain both 'application/json' and 'text/event-stream'", http.StatusBadRequest)
			return
		}

		var req jsonRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		var result any
		switch req.Method {
		case "initialize":
			result = initializeResult{ProtocolVersion: "2024-11-05"}
		case "notifications/initialized":
			result = struct{}{}
		case "tools/list":
			result = toolsListResult{
				Tools: []ToolDef{
					{Name: "artifact", Description: "Manage artifacts", InputSchema: json.RawMessage(`{"type":"object"}`)},
				},
			}
		default:
			result = struct{}{}
		}

		resultJSON, _ := json.Marshal(result)
		resp := jsonRPCResponse{JSONRPC: jsonRPCVersion, ID: req.ID, Result: resultJSON}
		respJSON, _ := json.Marshal(resp)

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", respJSON)
	}))
	defer srv.Close()

	c := New(djinnlog.Nop())
	defer c.Close()

	err := c.ConnectHTTP(context.Background(), "strict-scribe", srv.URL)
	if err != nil {
		t.Fatalf("ConnectHTTP should work with proper Accept header: %v", err)
	}

	if len(c.Tools()) != 1 {
		t.Fatalf("tools = %d, want 1", len(c.Tools()))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && stringContains(s, substr))
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestExtractSSEData(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"event: message\ndata: {\"id\":1}\n\n", `{"id":1}`},
		{"{\"id\":1}", ""},  // no SSE prefix — extractSSEData returns empty, direct JSON handled elsewhere
		{"data: {\"result\":\"ok\"}\n", `{"result":"ok"}`},
		{"event: error\ndata: not json\n", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := extractSSEData(tt.input)
		if got != tt.want {
			t.Errorf("extractSSEData(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLoadMCPConfig_NoFiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // prevent reading real ~/.cursor/mcp.json
	configs := LoadMCPConfig(t.TempDir(), t.TempDir())
	if len(configs) != 0 {
		t.Fatalf("should be empty with no config files, got %d", len(configs))
	}
}
