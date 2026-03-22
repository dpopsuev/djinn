// Package agent implements the agentic ReAct loop: send messages to
// an LLM driver, receive responses (which may contain tool calls),
// execute tools, feed results back, and repeat until the model is done.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// Defaults.
const (
	DefaultMaxTurns = 20
)

// Sentinel errors.
var (
	ErrMaxTurnsExceeded = errors.New("max turns exceeded")
)

// ApprovalFunc is called before executing a tool. Returns true to approve.
// The REPL provides an interactive version; headless auto-approves.
type ApprovalFunc func(call driver.ToolCall) bool

// EventHandler receives real-time events from the agent loop.
type EventHandler interface {
	OnText(text string)
	OnThinking(text string)
	OnToolCall(call driver.ToolCall)
	OnToolResult(callID, name, output string, isError bool)
	OnDone(usage *driver.Usage)
	OnError(err error)
}

// Config configures an agent loop execution.
type Config struct {
	Driver       driver.ChatDriver
	Tools        *builtin.Registry
	Session      *session.Session
	SystemPrompt string
	MaxTurns     int
	ToolsEnabled bool // false = ask/plan mode (no tool execution)
	Mode         Mode // agent mode for auto-weave
	Approve      ApprovalFunc
	Handler      EventHandler
	Log          *slog.Logger
}

// Run executes the agentic ReAct loop: send → receive → tool calls → repeat.
// Returns the final text response.
func Run(ctx context.Context, cfg Config, userPrompt string) (string, error) {
	if cfg.MaxTurns == 0 {
		cfg.MaxTurns = DefaultMaxTurns
	}
	if cfg.Log == nil {
		cfg.Log = djinnlog.Nop()
	}

	// Plan mode auto-weave: enrich prompt with Scribe + Lex context
	if cfg.Mode == ModePlan {
		userPrompt = AutoWeaveContext(ctx, cfg.Tools, userPrompt)
		cfg.Log.Debug("plan mode auto-weave applied")
	}

	// Append user message to session
	cfg.Session.Append(session.Entry{
		Role:    driver.RoleUser,
		Content: userPrompt,
	})

	// Send user message to driver
	if err := cfg.Driver.Send(ctx, driver.Message{
		Role:    driver.RoleUser,
		Content: userPrompt,
	}); err != nil {
		return "", fmt.Errorf("send: %w", err)
	}

	var finalText string

	for turn := range cfg.MaxTurns {
		turnStart := time.Now()
		cfg.Log.Info("turn start", "turn", turn+1, "max", cfg.MaxTurns)

		// Get streaming response
		events, err := cfg.Driver.Chat(ctx)
		if err != nil {
			return "", fmt.Errorf("chat: %w", err)
		}

		// Collect response
		response, err := collectResponse(events, cfg.Handler)
		if err != nil {
			return "", err
		}

		// Append assistant response to session
		cfg.Session.Append(session.Entry{
			Role:    driver.RoleAssistant,
			Content: response.text,
			Blocks:  response.blocks,
		})

		// Append to driver's message history
		cfg.Driver.AppendAssistant(driver.RichMessage{
			Role:   driver.RoleAssistant,
			Blocks: response.blocks,
		})

		finalText = response.text

		// If no tool calls, we're done
		if len(response.toolCalls) == 0 {
			cfg.Log.Info("turn complete", "turn", turn+1,
				slog.Group("perf",
					djinnlog.RTT(time.Since(turnStart)),
					djinnlog.TokensIn(usageIn(response.usage)),
					djinnlog.TokensOut(usageOut(response.usage)),
					djinnlog.Throughput(usageOut(response.usage), time.Since(turnStart)),
				),
			)
			break
		}

		// Tools disabled (ask/plan mode): skip execution
		if !cfg.ToolsEnabled {
			cfg.Log.Info("tools disabled, skipping execution", "turn", turn+1, "tool_calls", len(response.toolCalls))
			break
		}

		// Execute tool calls
		resultBlocks, err := executeTools(ctx, cfg, response.toolCalls)
		if err != nil {
			return finalText, err
		}

		// Send tool results back to driver
		toolResultMsg := driver.RichMessage{
			Role:   driver.RoleUser,
			Blocks: resultBlocks,
		}
		if err := cfg.Driver.SendRich(ctx, toolResultMsg); err != nil {
			return finalText, fmt.Errorf("send tool results: %w", err)
		}

		// Append tool results to session
		cfg.Session.Append(session.Entry{
			Role:   driver.RoleUser,
			Blocks: resultBlocks,
		})

		// Loop continues — driver will process tool results and respond
	}

	return finalText, nil
}

type collectedResponse struct {
	text      string
	toolCalls []driver.ToolCall
	blocks    []driver.ContentBlock
	usage     *driver.Usage
}

func collectResponse(events <-chan driver.StreamEvent, handler EventHandler) (collectedResponse, error) {
	var resp collectedResponse
	var textBuilder strings.Builder
	var thinkingBuilder strings.Builder

	for evt := range events {
		switch evt.Type {
		case driver.EventText:
			textBuilder.WriteString(evt.Text)
			if handler != nil {
				handler.OnText(evt.Text)
			}

		case driver.EventThinking:
			thinkingBuilder.WriteString(evt.Thinking)
			if handler != nil {
				handler.OnThinking(evt.Thinking)
			}

		case driver.EventToolUse:
			if evt.ToolCall != nil {
				resp.toolCalls = append(resp.toolCalls, *evt.ToolCall)
				if handler != nil {
					handler.OnToolCall(*evt.ToolCall)
				}
			}

		case driver.EventDone:
			resp.usage = evt.Usage
			if handler != nil {
				handler.OnDone(evt.Usage)
			}

		case driver.EventError:
			err := fmt.Errorf("%w: %s", ErrMaxTurnsExceeded, evt.Error)
			if handler != nil {
				handler.OnError(err)
			}
			return resp, err
		}
	}

	resp.text = textBuilder.String()

	// Build content blocks for history
	if thinking := thinkingBuilder.String(); thinking != "" {
		resp.blocks = append(resp.blocks, driver.NewThinkingBlock(thinking))
	}
	if text := resp.text; text != "" {
		resp.blocks = append(resp.blocks, driver.NewTextBlock(text))
	}
	for i := range resp.toolCalls {
		resp.blocks = append(resp.blocks, driver.NewToolUseBlock(
			resp.toolCalls[i].ID,
			resp.toolCalls[i].Name,
			resp.toolCalls[i].Input,
		))
	}

	return resp, nil
}

func executeTools(ctx context.Context, cfg Config, calls []driver.ToolCall) ([]driver.ContentBlock, error) {
	if cfg.Log == nil {
		cfg.Log = djinnlog.Nop()
	}
	var resultBlocks []driver.ContentBlock

	for _, call := range calls {
		// Check approval
		if cfg.Approve != nil && !cfg.Approve(call) {
			resultBlocks = append(resultBlocks, driver.NewToolResultBlock(
				call.ID, "tool call denied by operator", true,
			))
			if cfg.Handler != nil {
				cfg.Handler.OnToolResult(call.ID, call.Name, "denied", true)
			}
			continue
		}

		// Execute
		cfg.Log.Info("tool call", "tool", call.Name)
		toolStart := time.Now()
		output, err := cfg.Tools.Execute(ctx, call.Name, call.Input)
		isError := err != nil
		if isError {
			output = err.Error()
		}

		// Truncate long outputs
		const maxOutputLen = 50000
		if len(output) > maxOutputLen {
			output = output[:maxOutputLen] + "\n... (truncated)"
		}

		resultBlocks = append(resultBlocks, driver.NewToolResultBlock(
			call.ID, output, isError,
		))

		cfg.Log.Debug("tool result", "tool", call.Name, "error", isError, djinnlog.ToolLatency(time.Since(toolStart)))
		if cfg.Handler != nil {
			cfg.Handler.OnToolResult(call.ID, call.Name, truncateForDisplay(output), isError)
		}
	}

	return resultBlocks, nil
}

func truncateForDisplay(s string) string {
	const maxDisplay = 200
	if len(s) <= maxDisplay {
		return s
	}
	return s[:maxDisplay] + "..."
}

// AutoApprove approves all tool calls. Used in headless mode.
func AutoApprove(_ driver.ToolCall) bool { return true }

// DenyAll denies all tool calls.
func DenyAll(_ driver.ToolCall) bool { return false }

// ApproveByName returns an approval function that approves specific tools.
func ApproveByName(allowed ...string) ApprovalFunc {
	set := make(map[string]bool, len(allowed))
	for _, name := range allowed {
		set[name] = true
	}
	return func(call driver.ToolCall) bool {
		return set[call.Name]
	}
}

// NilHandler is an EventHandler that discards all events.
type NilHandler struct{}

func (NilHandler) OnText(string)                                  {}
func (NilHandler) OnThinking(string)                              {}
func (NilHandler) OnToolCall(driver.ToolCall)                     {}
func (NilHandler) OnToolResult(string, string, string, bool)      {}
func (NilHandler) OnDone(*driver.Usage)                           {}
func (NilHandler) OnError(error)                                  {}

// ensure NilHandler satisfies EventHandler
var _ EventHandler = NilHandler{}

func usageIn(u *driver.Usage) int {
	if u == nil {
		return 0
	}
	return u.InputTokens
}

func usageOut(u *driver.Usage) int {
	if u == nil {
		return 0
	}
	return u.OutputTokens
}

// ensure json is used (for tool input parsing in tests)
var _ = json.Marshal
