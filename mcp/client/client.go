package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dpopsuev/djinn/djinnlog"
)

// Sentinel errors.
var (
	ErrServerNotFound = errors.New("MCP server not found")
	ErrToolNotFound   = errors.New("MCP tool not found")
	ErrServerStopped  = errors.New("MCP server stopped")
)

// Client manages connections to multiple MCP servers.
type Client struct {
	mu      sync.RWMutex
	servers map[string]*ServerConn
	log     *slog.Logger
}

// New creates an MCP client.
func New(log *slog.Logger) *Client {
	if log == nil {
		log = djinnlog.Nop()
	}
	return &Client{
		servers: make(map[string]*ServerConn),
		log:     log,
	}
}

// ServerConn represents a connection to a single MCP server.
type ServerConn struct {
	name      string
	transport transport
	reqID     atomic.Int64
	tools     []ToolDef
	mu        sync.Mutex
}

// transport abstracts stdio vs HTTP communication.
type transport interface {
	Send(req jsonRPCRequest) (jsonRPCResponse, error)
	Close() error
}

// ConnectStdio connects to an MCP server via stdio subprocess.
func (c *Client) ConnectStdio(ctx context.Context, name, command string, args []string, env map[string]string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", command, err)
	}

	t := &stdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}

	conn := &ServerConn{name: name, transport: t}
	if err := c.initializeServer(conn); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("initialize %s: %w", name, err)
	}

	c.mu.Lock()
	c.servers[name] = conn
	c.mu.Unlock()

	c.log.Info("MCP server connected", "server", name, "transport", "stdio", "tools", len(conn.tools))
	return nil
}

// ConnectHTTP connects to an MCP server via HTTP.
func (c *Client) ConnectHTTP(ctx context.Context, name, url string) error {
	t := &httpTransport{url: strings.TrimRight(url, "/"), client: http.DefaultClient}

	conn := &ServerConn{name: name, transport: t}
	if err := c.initializeServer(conn); err != nil {
		return fmt.Errorf("initialize %s: %w", name, err)
	}

	c.mu.Lock()
	c.servers[name] = conn
	c.mu.Unlock()

	c.log.Info("MCP server connected", "server", name, "transport", "http", "tools", len(conn.tools))
	return nil
}

func (c *Client) initializeServer(conn *ServerConn) error {
	// Send initialize
	resp, err := conn.send("initialize", initializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo:      clientInfo{Name: "djinn", Version: "0.1.0"},
	})
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	if resp.Error != nil {
		return resp.Error
	}

	// Send initialized notification — must complete before tools/list
	if err := conn.sendNotification("notifications/initialized", nil); err != nil {
		// Some servers don't respond to notifications — continue anyway
	}

	// List tools
	toolsResp, err := conn.send("tools/list", nil)
	if err != nil {
		return fmt.Errorf("tools/list: %w", err)
	}
	if toolsResp.Error != nil {
		return toolsResp.Error
	}

	var result toolsListResult
	if err := json.Unmarshal(toolsResp.Result, &result); err != nil {
		return fmt.Errorf("parse tools: %w", err)
	}
	conn.tools = result.Tools
	return nil
}

func (conn *ServerConn) send(method string, params any) (jsonRPCResponse, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	id := conn.reqID.Add(1)
	req := jsonRPCRequest{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Method:  method,
		Params:  params,
	}
	return conn.transport.Send(req)
}

func (conn *ServerConn) sendNotification(method string, params any) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	req := jsonRPCRequest{
		JSONRPC: jsonRPCVersion,
		Method:  method,
		Params:  params,
	}
	resp, err := conn.transport.Send(req)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

// Tools returns all tool definitions from all connected servers.
func (c *Client) Tools() []ToolDef {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var all []ToolDef
	for _, conn := range c.servers {
		all = append(all, conn.tools...)
	}
	return all
}

// ServerTools returns tool definitions for a specific server.
func (c *Client) ServerTools(name string) ([]ToolDef, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	conn, ok := c.servers[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrServerNotFound, name)
	}
	return conn.tools, nil
}

// DefaultCallTimeout is the maximum time for a single MCP tool call.
const DefaultCallTimeout = 30 * time.Second

// Call executes a tool on the specified server with a timeout.
// If the call fails with a session/connection error, auto-reinitializes and retries once.
func (c *Client) Call(ctx context.Context, serverName, toolName string, input json.RawMessage) (string, error) {
	result, err := c.callOnce(ctx, serverName, toolName, input)
	if err != nil && isSessionError(err) {
		// MCP invisible reconnect — auto-reinitialize and retry
		c.mu.RLock()
		conn, ok := c.servers[serverName]
		c.mu.RUnlock()
		if ok {
			c.log.Info("MCP reconnecting", "server", serverName)
			if reinitErr := c.initializeServer(conn); reinitErr == nil {
				c.log.Info("MCP reconnected", "server", serverName)
				return c.callOnce(ctx, serverName, toolName, input)
			}
		}
	}
	return result, err
}

func (c *Client) callOnce(ctx context.Context, serverName, toolName string, input json.RawMessage) (string, error) {
	c.mu.RLock()
	conn, ok := c.servers[serverName]
	c.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("%w: %s", ErrServerNotFound, serverName)
	}

	callCtx, cancel := context.WithTimeout(ctx, DefaultCallTimeout)
	defer cancel()

	type callResultType struct {
		resp jsonRPCResponse
		err  error
	}
	ch := make(chan callResultType, 1)
	go func() {
		resp, err := conn.send("tools/call", toolCallParams{
			Name:      toolName,
			Arguments: input,
		})
		ch <- callResultType{resp, err}
	}()

	select {
	case <-callCtx.Done():
		return "", fmt.Errorf("tools/call %s.%s: %w", serverName, toolName, callCtx.Err())
	case cr := <-ch:
		if cr.err != nil {
			return "", fmt.Errorf("tools/call %s.%s: %w", serverName, toolName, cr.err)
		}
		if cr.resp.Error != nil {
			return "", cr.resp.Error
		}

		var result toolCallResult
		if err := json.Unmarshal(cr.resp.Result, &result); err != nil {
			return "", fmt.Errorf("parse result: %w", err)
		}

		var sb strings.Builder
		for _, block := range result.Content {
			if block.Type == "text" {
				sb.WriteString(block.Text)
			}
		}

		if result.IsError {
			return sb.String(), fmt.Errorf("tool error: %s", sb.String())
		}
		return sb.String(), nil
	}
}

// isSessionError returns true if the error suggests a session/connection failure
// that can be recovered by re-initializing the MCP connection.
func isSessionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "session") ||
		strings.Contains(msg, "connection") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "refused") ||
		strings.Contains(msg, "broken pipe")
}

// Healthy checks if a server is responsive by sending tools/list.
func (c *Client) Healthy(name string) bool {
	c.mu.RLock()
	conn, ok := c.servers[name]
	c.mu.RUnlock()
	if !ok {
		return false
	}
	resp, err := conn.send("tools/list", nil)
	return err == nil && resp.Error == nil
}

// ServerNames returns names of all connected servers.
func (c *Client) ServerNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.servers))
	for name := range c.servers {
		names = append(names, name)
	}
	return names
}

// Close disconnects all servers.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error
	for name, conn := range c.servers {
		if err := conn.transport.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close %s: %w", name, err))
		}
	}
	c.servers = make(map[string]*ServerConn)
	return errors.Join(errs...)
}

// --- stdio transport ---

type stdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
}

func (t *stdioTransport) Send(req jsonRPCRequest) (jsonRPCResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return jsonRPCResponse{}, err
	}
	data = append(data, '\n')

	if _, err := t.stdin.Write(data); err != nil {
		return jsonRPCResponse{}, fmt.Errorf("write: %w", err)
	}

	line, err := t.stdout.ReadBytes('\n')
	if err != nil {
		return jsonRPCResponse{}, fmt.Errorf("read: %w", err)
	}

	var resp jsonRPCResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return jsonRPCResponse{}, fmt.Errorf("parse response: %w", err)
	}
	return resp, nil
}

func (t *stdioTransport) Close() error {
	t.stdin.Close()
	return t.cmd.Process.Kill()
}

// --- HTTP transport (SSE-aware) ---

type httpTransport struct {
	url    string
	client *http.Client
}

func (t *httpTransport) Send(req jsonRPCRequest) (jsonRPCResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return jsonRPCResponse{}, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, t.url, bytes.NewReader(data))
	if err != nil {
		return jsonRPCResponse{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")

	httpResp, err := t.client.Do(httpReq)
	if err != nil {
		return jsonRPCResponse{}, fmt.Errorf("http post: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return jsonRPCResponse{}, fmt.Errorf("read body: %w", err)
	}

	// Try direct JSON first (Streamable HTTP)
	var resp jsonRPCResponse
	if err := json.Unmarshal(body, &resp); err == nil {
		return resp, nil
	}

	// Fall back to SSE parsing: extract JSON from "data: {json}" lines
	jsonData := extractSSEData(string(body))
	if jsonData == "" {
		return jsonRPCResponse{}, fmt.Errorf("parse response: no JSON found in SSE body")
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		return jsonRPCResponse{}, fmt.Errorf("parse SSE data: %w", err)
	}
	return resp, nil
}

// extractSSEData finds the first "data: {json}" line in an SSE response.
func extractSSEData(body string) string {
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if after, ok := strings.CutPrefix(line, "data: "); ok {
			// Verify it looks like JSON
			trimmed := strings.TrimSpace(after)
			if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
				return trimmed
			}
		}
	}
	return ""
}

func (t *httpTransport) Close() error {
	return nil // HTTP connections are stateless
}
