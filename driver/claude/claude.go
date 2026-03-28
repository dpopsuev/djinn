package claude

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/tier"
)

// Sentinel errors for ClaudeDriver.
var (
	ErrAlreadyStarted = errors.New("driver already started")
	ErrNotRunning     = errors.New("driver not running")
)

// Environment variable keys set by ContainerEnv.
const (
	EnvDjinnTier       = "DJINN_TIER"
	EnvDjinnScope      = "DJINN_SCOPE"
	EnvDjinnSandbox    = "DJINN_SANDBOX"
	EnvAnthropicModel  = "ANTHROPIC_MODEL"
	EnvClaudeMaxTokens = "CLAUDE_MAX_TOKENS"
)

// ContainerEnv describes the environment configuration for running Claude Code
// inside a Misbah container. This is the spec Misbah needs to create the jail.
type ContainerEnv struct {
	Model      string
	Prompt     string
	ClaudeMD   string            // injected CLAUDE.md content
	MCPServers map[string]string // server name -> socket path
	EnvVars    map[string]string
	Scope      tier.Scope
}

// ClaudeDriver implements driver.Driver for Claude Code agents.
// In the MVP, it generates container environment specs and records messages.
// Real LLM communication is deferred to Misbah container integration.
type ClaudeDriver struct {
	config   driver.DriverConfig
	scope    tier.Scope
	claudeMD string

	mu      sync.Mutex
	sandbox driver.SandboxHandle
	prompt  string
	recvCh  chan driver.Message
	started bool
	stopped bool
}

// Option configures a ClaudeDriver.
type Option func(*ClaudeDriver)

// WithScope sets the tier scope for container environment generation.
func WithScope(scope tier.Scope) Option {
	return func(d *ClaudeDriver) { d.scope = scope }
}

// WithClaudeMD sets the CLAUDE.md content to inject into the container.
func WithClaudeMD(content string) Option {
	return func(d *ClaudeDriver) { d.claudeMD = content }
}

// New creates a new ClaudeDriver with the given config and options.
func New(config driver.DriverConfig, opts ...Option) *ClaudeDriver {
	d := &ClaudeDriver{
		config: config,
		recvCh: make(chan driver.Message, 1),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *ClaudeDriver) Start(ctx context.Context, sandbox driver.SandboxHandle) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.started {
		return ErrAlreadyStarted
	}
	d.sandbox = sandbox
	d.started = true
	return nil
}

func (d *ClaudeDriver) Send(ctx context.Context, msg driver.Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.started || d.stopped {
		return ErrNotRunning
	}
	d.prompt = msg.Content

	// MVP: emit a canned completion and close the channel.
	// Real implementation would stream from Claude Code process.
	d.recvCh <- driver.Message{
		Role:    driver.RoleAssistant,
		Content: fmt.Sprintf("completed: %s", msg.Content),
	}
	close(d.recvCh)
	return nil
}

func (d *ClaudeDriver) Recv(ctx context.Context) <-chan driver.Message {
	return d.recvCh
}

func (d *ClaudeDriver) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopped = true
	return nil
}

// ContainerEnv returns the environment spec for a Misbah container.
func (d *ClaudeDriver) ContainerEnv() ContainerEnv {
	d.mu.Lock()
	defer d.mu.Unlock()

	env := ContainerEnv{
		Model:      d.config.Model,
		Prompt:     d.prompt,
		ClaudeMD:   d.claudeMD,
		MCPServers: make(map[string]string),
		EnvVars:    make(map[string]string),
		Scope:      d.scope,
	}

	env.EnvVars[EnvDjinnTier] = d.scope.Level.String()
	env.EnvVars[EnvDjinnScope] = d.scope.Name
	env.EnvVars[EnvDjinnSandbox] = d.sandbox

	if d.config.Model != "" {
		env.EnvVars[EnvAnthropicModel] = d.config.Model
	}
	if d.config.MaxTokens > 0 {
		env.EnvVars[EnvClaudeMaxTokens] = fmt.Sprintf("%d", d.config.MaxTokens)
	}

	return env
}
