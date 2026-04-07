// Package builtin implements the agent's built-in tools: file operations,
// shell execution, and code search. These are the same tools Claude Code
// has natively, reimplemented in Go so any model driver can use them.
package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	batterytool "github.com/dpopsuev/battery/tool"
)

// Sentinel errors — re-exported from Battery.
var (
	ErrToolNotFound = batterytool.ErrNotFound
	ErrEmptyInput   = batterytool.ErrEmptyInput
)

// Tool is the interface every built-in tool implements.
// Type alias to battery/tool.Tool — single source of truth.
type Tool = batterytool.Tool

// ToolExecutor is the interface for tool dispatch.
// Type alias to battery/tool.Executor — single source of truth.
type ToolExecutor = batterytool.Executor

// Registry holds registered tools and dispatches calls.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a registry with all built-in tools pre-registered.
func NewRegistry() *Registry {
	r := &Registry{tools: make(map[string]Tool)}
	r.Register(&ReadTool{})
	r.Register(&WriteTool{})
	r.Register(&EditTool{})
	r.Register(&BashTool{})
	r.Register(&GlobTool{})
	r.Register(&GrepTool{})
	return r
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}
	return t, nil
}

// Execute dispatches a tool call by name.
func (r *Registry) Execute(ctx context.Context, name string, input json.RawMessage) (string, error) {
	t, err := r.Get(name)
	if err != nil {
		return "", err
	}
	return t.Execute(ctx, input)
}

// All returns all registered tools.
func (r *Registry) All() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// Names returns all registered tool names sorted.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.tools))
	for name := range r.tools {
		out = append(out, name)
	}
	return out
}
