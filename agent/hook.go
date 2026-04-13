// hook.go — HookRunner: shell command interception at tool boundaries.
//
// HookRunner is superseded by hook.EventDispatcher (GOL-161).
// Retained for backward compatibility until the REPL migrates.
// Convert via hook.ConvertLegacy().
//
// HookRunner implements BOTH ToolGate (pre) and Recorder (post):
//   - ToolGate.Check(): runs pre_tool_use hooks before execution
//   - Recorder.Record(): runs post_tool_use hooks after execution
//
// Hook matching: each hook has a Tools list; "*" matches all tools.
// Exit codes: 0 = allow, 2 = deny (stdout = reason), other = warn but continue.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os/exec"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
)

// HookConfig defines a single hook entry — a shell command and the tools it applies to.
type HookConfig struct {
	Command string   `yaml:"command"`
	Tools   []string `yaml:"tools"` // tool names or "*" for wildcard
}

// HookRunner intercepts tool calls via shell commands.
// Implements ToolGate (pre-tool) and Recorder (post-tool).
type HookRunner struct {
	preToolUse  []HookConfig
	postToolUse []HookConfig
	log         *slog.Logger
}

// NewHookRunner creates a HookRunner from pre and post hook configs.
func NewHookRunner(pre, post []HookConfig, log *slog.Logger) *HookRunner {
	if log == nil {
		log = telemetry.Nop()
	}
	return &HookRunner{
		preToolUse:  pre,
		postToolUse: post,
		log:         log,
	}
}

// hookDenyExitCode is the exit code that means "deny this tool call".
const hookDenyExitCode = 2

// preHookPayload is the JSON payload piped to pre_tool_use hooks.
type preHookPayload struct {
	HookEvent string          `json:"hook_event"`
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// postHookPayload is the JSON payload piped to post_tool_use hooks.
type postHookPayload struct {
	HookEvent  string          `json:"hook_event"`
	ToolName   string          `json:"tool_name"`
	ToolInput  json.RawMessage `json:"tool_input"`
	ToolOutput string          `json:"tool_output"`
	IsError    bool            `json:"is_error"`
}

// Check runs pre_tool_use hooks. First deny stops the chain.
func (h *HookRunner) Check(ctx context.Context, tool string, input json.RawMessage) (ToolGateResult, error) {
	for _, hook := range h.preToolUse {
		if !hookMatches(hook, tool) {
			continue
		}

		payload, err := json.Marshal(preHookPayload{
			HookEvent: "pre_tool_use",
			ToolName:  tool,
			ToolInput: input,
		})
		if err != nil {
			h.log.Warn("hook payload marshal failed",
				slog.String(telemetry.KeyComponent, "hook"),
				slog.String(telemetry.KeyTool, tool),
				slog.String(telemetry.KeyError, err.Error()),
			)
			continue
		}

		start := time.Now()
		stdout, exitCode, runErr := runHook(ctx, hook.Command, payload)
		elapsed := time.Since(start)

		if runErr != nil && exitCode < 0 {
			// Command failed to execute (not an exit code issue)
			h.log.Warn("hook execution error",
				slog.String(telemetry.KeyComponent, "hook"),
				slog.String(telemetry.KeyTool, tool),
				slog.String(telemetry.KeyAction, "pre_tool_use"),
				slog.String(telemetry.KeyError, runErr.Error()),
				slog.Duration(telemetry.KeyDuration, elapsed),
			)
			continue
		}

		switch exitCode {
		case 0:
			h.log.Debug("hook allowed",
				slog.String(telemetry.KeyComponent, "hook"),
				slog.String(telemetry.KeyTool, tool),
				slog.String(telemetry.KeyAction, "pre_tool_use"),
				slog.String(telemetry.KeyDecision, "allow"),
				slog.Duration(telemetry.KeyDuration, elapsed),
			)
		case hookDenyExitCode:
			reason := stdout
			if reason == "" {
				reason = "denied by pre_tool_use hook"
			}
			h.log.Warn("hook denied",
				slog.String(telemetry.KeyComponent, "hook"),
				slog.String(telemetry.KeyTool, tool),
				slog.String(telemetry.KeyAction, "pre_tool_use"),
				slog.String(telemetry.KeyDecision, "deny"),
				slog.Duration(telemetry.KeyDuration, elapsed),
			)
			return ToolGateResult{Allowed: false, Reason: reason}, nil
		default:
			h.log.Warn("hook unexpected exit code",
				slog.String(telemetry.KeyComponent, "hook"),
				slog.String(telemetry.KeyTool, tool),
				slog.String(telemetry.KeyAction, "pre_tool_use"),
				slog.String(telemetry.KeyDecision, "warn"),
				slog.Int(telemetry.KeyExitCode, exitCode),
				slog.Duration(telemetry.KeyDuration, elapsed),
			)
			// Other exit codes warn but continue
		}
	}

	return ToolGateResult{Allowed: true}, nil
}

// Record runs post_tool_use hooks. Exit code doesn't affect result.
func (h *HookRunner) Record(_ context.Context, tool string, input json.RawMessage, output string, err error, elapsed time.Duration) {
	for _, hook := range h.postToolUse {
		if !hookMatches(hook, tool) {
			continue
		}

		payload, marshalErr := json.Marshal(postHookPayload{
			HookEvent:  "post_tool_use",
			ToolName:   tool,
			ToolInput:  input,
			ToolOutput: output,
			IsError:    err != nil,
		})
		if marshalErr != nil {
			h.log.Warn("hook payload marshal failed",
				slog.String(telemetry.KeyComponent, "hook"),
				slog.String(telemetry.KeyTool, tool),
				slog.String(telemetry.KeyError, marshalErr.Error()),
			)
			continue
		}

		start := time.Now()
		_, _, runErr := runHook(context.Background(), hook.Command, payload)
		hookElapsed := time.Since(start)

		if runErr != nil {
			h.log.Warn("post hook execution error",
				slog.String(telemetry.KeyComponent, "hook"),
				slog.String(telemetry.KeyTool, tool),
				slog.String(telemetry.KeyAction, "post_tool_use"),
				slog.String(telemetry.KeyError, runErr.Error()),
				slog.Duration(telemetry.KeyDuration, hookElapsed),
			)
		} else {
			h.log.Debug("post hook completed",
				slog.String(telemetry.KeyComponent, "hook"),
				slog.String(telemetry.KeyTool, tool),
				slog.String(telemetry.KeyAction, "post_tool_use"),
				slog.Duration(telemetry.KeyDuration, hookElapsed),
			)
		}
	}
}

// hookMatches returns true if the hook applies to the given tool name.
func hookMatches(hook HookConfig, tool string) bool {
	for _, t := range hook.Tools {
		if t == "*" || t == tool {
			return true
		}
	}
	return false
}

// runHook executes a shell command via sh -c, piping payload to stdin.
// Returns stdout, exit code, and error. Exit code is -1 if the command
// failed to start.
func runHook(ctx context.Context, command string, payload []byte) (string, int, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Stdin = bytes.NewReader(payload)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		// Try to extract exit code from ExitError.
		if exitError, ok := err.(*exec.ExitError); ok { //nolint:errorlint // need concrete type for ExitCode()
			return stdout.String(), exitError.ExitCode(), err
		}
		return stdout.String(), -1, err
	}

	return stdout.String(), 0, nil
}

// Ensure interface compliance.
var (
	_ ToolGate = (*HookRunner)(nil)
	_ Recorder = (*HookRunner)(nil)
)
