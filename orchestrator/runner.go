// Package orchestrator owns agent lifecycle — starting agent runs,
// approval bridging, and shell execution routing.
//
// Extracted from cli/repl/model.go (SRP: separate orchestration from TUI).
// No tui imports — the caller wraps results into TUI messages.
package orchestrator

import (
	"context"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/contextmgr"
	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/uniform"
)

// AgentRunner encapsulates agent lifecycle operations.
// Constructed once per REPL session, shared across agent invocations.
type AgentRunner struct {
	Driver       driver.ChatDriver
	Tools        builtin.ToolExecutor
	Envelope     *agent.ToolEnvelope
	Session      *contextmgr.Session
	SystemPrompt string
	MaxTurns     int
	Enforcer     policy.ToolPolicyEnforcer
	Token        policy.CapabilityToken
	Router       *uniform.ToolClearance
	Log          *slog.Logger

	// Sandbox
	SandboxHandle string
	SandboxExec   func(ctx context.Context, cmd []string) (string, string, error)
}

// RunAgent starts an agent loop and blocks until completion.
// Returns the agent's output and any error. The caller wraps
// the result into a TUI message.
func (r *AgentRunner) RunAgent(ctx context.Context, prompt string, mode agent.Mode, approvalCh chan bool, handler agent.EventHandler, currentRole string) (string, error) {
	agentLog := djinnlog.For(r.Log, "agent")

	var tools builtin.ToolExecutor = r.Tools
	if r.Router != nil {
		tools = r.Router
	}

	cfg := agent.Config{
		Driver:       r.Driver,
		Tools:        tools,
		Envelope:     r.Envelope,
		Session:      r.Session,
		SystemPrompt: r.SystemPrompt,
		MaxTurns:     r.MaxTurns,
		ToolsEnabled: mode.ToolsEnabled(),
		Mode:         mode,
		Approve:      ApprovalForMode(mode, approvalCh),
		Enforcer:     r.Enforcer,
		Token:        r.Token,
		Handler:      handler,
		Log:          agentLog,
	}

	// Sandbox: everyone sandboxed except GenSec.
	if r.SandboxHandle != "" && currentRole != "gensec" {
		cfg.SandboxHandle = r.SandboxHandle
		cfg.SandboxExec = r.SandboxExec
		cfg.SandboxWorkDir = r.Session.WorkDir
		cfg.SandboxMount = "/workspace"
	}

	return agent.Run(ctx, cfg, prompt)
}

// RunShell executes a shell command, routing through sandbox if available.
// Returns combined output and any error.
func (r *AgentRunner) RunShell(ctx context.Context, cmd, workDir, currentRole string) (string, error) {
	if r.SandboxExec != nil && currentRole != "gensec" {
		stdout, stderr, err := r.SandboxExec(ctx, strings.Fields(cmd))
		output := stdout
		if stderr != "" {
			output += "\n" + stderr
		}
		return output, err
	}

	execCmd := exec.CommandContext(ctx, "bash", "-c", cmd)
	execCmd.Dir = workDir
	out, err := execCmd.CombinedOutput()
	return string(out), err
}

// ApprovalForMode returns the approval function for the given mode.
// Auto mode auto-approves, agent mode blocks on channel, others deny.
func ApprovalForMode(mode agent.Mode, ch chan bool) agent.ApprovalFunc {
	switch mode {
	case agent.ModeAuto:
		return agent.AutoApprove
	case agent.ModeAgent:
		return func(_ driver.ToolCall) bool {
			return <-ch
		}
	default:
		return agent.DenyAll
	}
}
