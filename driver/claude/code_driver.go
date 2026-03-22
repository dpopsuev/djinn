package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/mcp"
)

// Claude Code CLI flags and defaults.
const (
	claudeBinary         = "claude"
	flagPrint            = "-p"
	flagOutputFormat     = "--output-format"
	flagModel            = "--model"
	flagMaxTurns         = "--max-turns"
	flagAllowedTools     = "--allowedTools"
	flagPermissionMode   = "--permission-mode"
	flagAppendSystem     = "--append-system-prompt"
	outputFormatJSON     = "json"
	permissionAcceptEdit = "acceptEdits"
	defaultMaxTurns      = "20"
	defaultAllowedTools  = "Read,Edit,Write,Bash,Glob,Grep"
	defaultExecTimeout   = int64(300) // 5 minutes
)

// ExecFunc runs a command inside a container and returns the result.
// Injected by the orchestrator — breaks the dependency on sandbox package.
type ExecFunc func(ctx context.Context, name string, cmd []string, timeout int64) (ExecResult, error)

// ExecResult mirrors sandbox/misbah.ExecResult to avoid import cycle.
type ExecResult struct {
	ExitCode int32
	Stdout   string
	Stderr   string
}

// CodeDriver implements driver.Driver by running Claude Code CLI
// inside a Misbah container via an exec function.
type CodeDriver struct {
	config       driver.DriverConfig
	execFn       ExecFunc
	systemPrompt string
	mcpServers   []mcp.Server
	mcpConfigDir string // temp dir for MCP config file, cleaned up on Stop

	mu      sync.Mutex
	sandbox driver.SandboxHandle
	recvCh  chan driver.Message
	started bool
	stopped bool
}

// CodeDriverOption configures a CodeDriver.
type CodeDriverOption func(*CodeDriver)

// WithMCPServers registers MCP servers to expose to Claude Code.
func WithMCPServers(servers []mcp.Server) CodeDriverOption {
	return func(d *CodeDriver) { d.mcpServers = servers }
}

// WithSystemPrompt sets the system prompt appended to Claude Code.
func WithSystemPrompt(prompt string) CodeDriverOption {
	return func(d *CodeDriver) { d.systemPrompt = prompt }
}

// NewCodeDriver creates a driver that runs Claude Code CLI via exec.
// execFn is called to execute commands inside the container.
func NewCodeDriver(config driver.DriverConfig, execFn ExecFunc, opts ...CodeDriverOption) *CodeDriver {
	d := &CodeDriver{
		config: config,
		execFn: execFn,
		recvCh: make(chan driver.Message, 1),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *CodeDriver) Start(ctx context.Context, sandbox driver.SandboxHandle) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.started {
		return ErrAlreadyStarted
	}
	d.sandbox = sandbox
	d.started = true
	return nil
}

func (d *CodeDriver) Send(ctx context.Context, msg driver.Message) error {
	d.mu.Lock()
	if !d.started || d.stopped {
		d.mu.Unlock()
		return ErrNotRunning
	}
	sandbox := d.sandbox
	d.mu.Unlock()

	cmd := d.buildCommand(msg.Content)

	result, err := d.execFn(ctx, sandbox, cmd, defaultExecTimeout)
	if err != nil {
		d.recvCh <- driver.Message{
			Role:    driver.RoleAssistant,
			Content: fmt.Sprintf("exec error: %v", err),
		}
		close(d.recvCh)
		return nil
	}

	response := d.parseResponse(result)
	d.recvCh <- driver.Message{
		Role:    driver.RoleAssistant,
		Content: response,
	}
	close(d.recvCh)
	return nil
}

func (d *CodeDriver) Recv(ctx context.Context) <-chan driver.Message {
	return d.recvCh
}

func (d *CodeDriver) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopped = true
	if d.mcpConfigDir != "" {
		os.RemoveAll(d.mcpConfigDir)
		d.mcpConfigDir = ""
	}
	return nil
}

const flagMCPConfig = "--mcp-config"

func (d *CodeDriver) buildCommand(prompt string) []string {
	cmd := []string{
		claudeBinary, flagPrint, prompt,
		flagOutputFormat, outputFormatJSON,
		flagPermissionMode, permissionAcceptEdit,
		flagMaxTurns, defaultMaxTurns,
		flagAllowedTools, defaultAllowedTools,
	}

	if d.config.Model != "" {
		cmd = append(cmd, flagModel, d.config.Model)
	}

	if d.systemPrompt != "" {
		cmd = append(cmd, flagAppendSystem, d.systemPrompt)
	}

	// Write MCP config file if servers are configured
	if len(d.mcpServers) > 0 {
		dir, err := os.MkdirTemp("", "djinn-mcp-*")
		if err == nil {
			path, err := mcp.WriteConfigFile(dir, d.mcpServers)
			if err == nil {
				d.mcpConfigDir = dir
				cmd = append(cmd, flagMCPConfig, path)
			}
		}
	}

	return cmd
}

// claudeJSONResponse is the structure Claude Code returns in JSON mode.
type claudeJSONResponse struct {
	Result    string `json:"result"`
	SessionID string `json:"session_id"`
	Usage     struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (d *CodeDriver) parseResponse(result ExecResult) string {
	if result.ExitCode != 0 {
		return fmt.Sprintf("claude exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// Try to parse JSON response
	stdout := strings.TrimSpace(result.Stdout)
	var resp claudeJSONResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		// Fallback: return raw stdout if not valid JSON
		return stdout
	}

	return resp.Result
}
