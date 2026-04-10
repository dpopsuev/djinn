package builtin

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"strings"

	mcpclient "github.com/dpopsuev/djinn/mcp/client"
)

// CompositeExecutor routes tool calls to either MCP servers or built-in tools.
// When an MCP tool name matches a built-in tool name (after stripping the
// mcp__<server>__ prefix), the MCP tool takes precedence. This implements
// the tool upgrade path: MCP servers can shadow built-in implementations.
type CompositeExecutor struct {
	builtin   *Registry
	mcp       *mcpclient.Client
	overrides map[string]string // builtin tool name → MCP server name
	log       *slog.Logger
}

// NewCompositeExecutor creates a CompositeExecutor that merges built-in and
// MCP tools. MCP tools whose raw name matches a built-in tool name are
// registered as overrides — calls to that name route through MCP.
func NewCompositeExecutor(builtin *Registry, mcp *mcpclient.Client, log *slog.Logger) *CompositeExecutor {
	ce := &CompositeExecutor{
		builtin:   builtin,
		mcp:       mcp,
		overrides: make(map[string]string),
		log:       log,
	}
	ce.detectOverrides()
	return ce
}

// mcpRawNames maps raw MCP tool name → server name for raw-name dispatch.
// This allows Execute("artifact") to route to the MCP server even though
// the tool is registered as "mcp__scribe__artifact" in the registry.
func (ce *CompositeExecutor) buildMCPIndex() map[string]string {
	if ce.mcp == nil {
		return nil
	}
	idx := make(map[string]string)
	for _, t := range ce.mcp.MCPTools() {
		idx[t.RawName()] = t.ServerName()
	}
	return idx
}

// detectOverrides scans MCP tools for names that match built-in tools.
// When found, the MCP version takes precedence.
func (ce *CompositeExecutor) detectOverrides() {
	if ce.mcp == nil {
		return
	}
	builtinNames := make(map[string]bool, len(ce.builtin.tools))
	for name := range ce.builtin.tools {
		builtinNames[name] = true
	}
	for _, mcpTool := range ce.mcp.MCPTools() {
		// MCPTool.Name() returns "mcp__<server>__<tool>". The raw tool name
		// from the server is the def.Name accessible via the tool list.
		rawName := mcpTool.RawName()
		if builtinNames[rawName] {
			// Find which server this tool belongs to.
			serverName := mcpTool.ServerName()
			ce.overrides[rawName] = serverName
			ce.log.Info("tool upgraded to MCP", "tool", rawName, "server", serverName)
		}
	}
}

// Execute dispatches a tool call:
//  1. If the tool name is in the overrides map (builtin shadowed by MCP), route to MCP.
//  2. If the builtin registry has it, use builtin.
//  3. If the raw tool name matches an MCP tool, route to MCP.
//     This supports ToolClearance calling Execute("artifact") when "artifact"
//     is an MCP tool from Scribe (registered as "mcp__scribe__artifact").
//  4. Otherwise return tool-not-found.
func (ce *CompositeExecutor) Execute(ctx context.Context, name string, input json.RawMessage) (string, error) {
	// 1. Overrides: builtin name shadowed by MCP.
	if serverName, ok := ce.overrides[name]; ok {
		return ce.mcp.Call(ctx, serverName, name, input)
	}
	// 2. Builtin registry.
	if _, err := ce.builtin.Get(name); err == nil {
		return ce.builtin.Execute(ctx, name, input)
	}
	// 3. Full MCP name dispatch (e.g. "mcp__scribe__artifact" → server=scribe, tool=artifact).
	if ce.mcp != nil && strings.HasPrefix(name, "mcp__") {
		parts := strings.SplitN(name, "__", 3)
		if len(parts) == 3 {
			return ce.mcp.Call(ctx, parts[1], parts[2], input)
		}
	}
	// 4. Raw MCP tool name dispatch (e.g. "artifact" → scribe server).
	if ce.mcp != nil {
		idx := ce.buildMCPIndex()
		if serverName, ok := idx[name]; ok {
			return ce.mcp.Call(ctx, serverName, name, input)
		}
	}
	// 4. Not found.
	return ce.builtin.Execute(ctx, name, input)
}

// All returns all available tools — built-in tools plus MCP tools.
// Overridden built-in tools are excluded (replaced by MCP versions).
func (ce *CompositeExecutor) All() []Tool {
	var all []Tool

	// Add non-overridden builtins.
	for _, t := range ce.builtin.All() {
		if _, overridden := ce.overrides[t.Name()]; !overridden {
			all = append(all, t)
		}
	}

	// Add MCP tools.
	if ce.mcp != nil {
		for _, t := range ce.mcp.MCPTools() {
			all = append(all, t)
		}
	}

	return all
}

// Names returns all available tool names sorted.
func (ce *CompositeExecutor) Names() []string {
	seen := make(map[string]bool)
	var names []string

	// Add non-overridden builtins.
	for _, n := range ce.builtin.Names() {
		if _, overridden := ce.overrides[n]; !overridden {
			if !seen[n] {
				names = append(names, n)
				seen[n] = true
			}
		}
	}

	// Add MCP tools.
	if ce.mcp != nil {
		for _, t := range ce.mcp.MCPTools() {
			n := t.Name()
			if !seen[n] {
				names = append(names, n)
				seen[n] = true
			}
		}
	}

	sort.Strings(names)
	return names
}

// Overrides returns the current override map (builtin name -> MCP server).
// Primarily for logging and diagnostics.
func (ce *CompositeExecutor) Overrides() map[string]string {
	out := make(map[string]string, len(ce.overrides))
	for k, v := range ce.overrides {
		out[k] = v
	}
	return out
}
