package mcp

import (
	"encoding/json"
	"fmt"
	"sync"
)

// ToolDef matches mcp/client.ToolDef for tool definitions.
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolCall records a tool invocation.
type ToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// MockMCPServer responds to JSON-RPC 2.0 MCP protocol requests in-process.
type MockMCPServer struct {
	mu      sync.Mutex
	tools   map[string]ToolDef
	results map[string]string // tool name → canned result text
	calls   []ToolCall
	errors  map[string]string // tool name → error message (if set, return error)
}

// NewMockMCPServer creates a new MockMCPServer ready for tool registration.
func NewMockMCPServer() *MockMCPServer {
	return &MockMCPServer{
		tools:   make(map[string]ToolDef),
		results: make(map[string]string),
		errors:  make(map[string]string),
	}
}

// RegisterTool adds a tool with a canned result.
func (m *MockMCPServer) RegisterTool(name, description, result string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[name] = ToolDef{
		Name:        name,
		Description: description,
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}
	m.results[name] = result
}

// RegisterError makes a tool return an error.
func (m *MockMCPServer) RegisterError(name, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[name] = errMsg
}

// Calls returns all recorded tool calls.
func (m *MockMCPServer) Calls() []ToolCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ToolCall{}, m.calls...)
}

// Handle processes a JSON-RPC request and returns a response.
func (m *MockMCPServer) Handle(method string, id int64, params json.RawMessage) (json.RawMessage, error) {
	switch method {
	case "initialize":
		result, _ := json.Marshal(map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo":      map[string]string{"name": "mock", "version": "0.1.0"},
		})
		return result, nil

	case "notifications/initialized":
		return nil, nil // no response for notifications

	case "tools/list":
		m.mu.Lock()
		tools := make([]ToolDef, 0, len(m.tools))
		for _, t := range m.tools {
			tools = append(tools, t)
		}
		m.mu.Unlock()
		result, _ := json.Marshal(map[string]any{"tools": tools})
		return result, nil

	case "tools/call":
		var callParams struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(params, &callParams); err != nil {
			return nil, fmt.Errorf("invalid tools/call params: %w", err)
		}

		m.mu.Lock()
		m.calls = append(m.calls, ToolCall{Name: callParams.Name, Arguments: callParams.Arguments})

		if errMsg, ok := m.errors[callParams.Name]; ok {
			m.mu.Unlock()
			result, _ := json.Marshal(map[string]any{
				"content": []map[string]string{{"type": "text", "text": errMsg}},
				"isError": true,
			})
			return result, nil
		}

		text, ok := m.results[callParams.Name]
		m.mu.Unlock()
		if !ok {
			return nil, fmt.Errorf("unknown tool: %s", callParams.Name)
		}
		result, _ := json.Marshal(map[string]any{
			"content": []map[string]string{{"type": "text", "text": text}},
		})
		return result, nil

	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}
