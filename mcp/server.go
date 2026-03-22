// Package mcp provides MCP server configuration for wiring Aeon ecosystem
// tools into agent containers. It generates the MCP config file that Claude
// Code (or any MCP-capable agent CLI) reads to discover available tools.
package mcp

// Transport types for MCP servers.
const (
	TypeHTTP  = "http"
	TypeStdio = "stdio"
)

// Server describes an MCP server that can be registered with an agent CLI.
type Server struct {
	Name    string            // unique identifier (e.g. "scribe", "locus")
	Type    string            // TypeHTTP or TypeStdio
	URL     string            // for HTTP servers (e.g. "http://localhost:8080/")
	Command string            // for stdio servers (e.g. "lex")
	Args    []string          // arguments for stdio command
	Env     map[string]string // environment variables
	CWD     string            // working directory for stdio command
}
