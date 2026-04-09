// driver.go — TroupeChatDriver: wraps anyllm.Provider as driver.ChatDriver.
//
// Single adapter for all LLM providers (Anthropic, OpenAI, Gemini, OpenRouter).
// Manages conversation history internally. Uses Completion (blocking) for MVP;
// streaming can be added for TUI feedback later.
package troupe

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/telemetry"
)

var _ driver.ChatDriver = (*ChatDriver)(nil)

// ChatDriver wraps an anyllm.Provider as a driver.ChatDriver.
type ChatDriver struct {
	mu        sync.Mutex
	provider  anyllm.Provider
	model     string
	system    string
	maxTokens int
	messages  []anyllm.Message
	tools     []anyllm.Tool
	log       *slog.Logger
}

// Option configures a ChatDriver.
type Option func(*ChatDriver)

// WithLogger sets the structured logger.
func WithLogger(l *slog.Logger) Option {
	return func(d *ChatDriver) { d.log = l }
}

// WithSystemPrompt sets the initial system prompt.
func WithSystemPrompt(prompt string) Option {
	return func(d *ChatDriver) { d.system = prompt }
}

// WithTools sets the tool definitions sent to the provider.
func WithTools(tools []anyllm.Tool) Option {
	return func(d *ChatDriver) { d.tools = tools }
}

// WithMaxTokens sets the maximum output tokens per completion.
func WithMaxTokens(n int) Option {
	return func(d *ChatDriver) { d.maxTokens = n }
}

const defaultMaxTokens = 8192

// New creates a TroupeChatDriver wrapping the given provider.
func New(provider anyllm.Provider, model string, opts ...Option) *ChatDriver {
	d := &ChatDriver{
		provider:  provider,
		model:     model,
		maxTokens: defaultMaxTokens,
		log:       telemetry.Nop(),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *ChatDriver) Start(_ context.Context, _ driver.SandboxHandle) error { return nil }
func (d *ChatDriver) Stop(_ context.Context) error                          { return nil }

// Send appends a plain text message to conversation history.
func (d *ChatDriver) Send(_ context.Context, msg driver.Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, anyllm.Message{
		Role:    msg.Role,
		Content: msg.Content,
	})
	return nil
}

// SendRich appends structured content blocks (tool results) to conversation history.
func (d *ChatDriver) SendRich(_ context.Context, msg driver.RichMessage) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(msg.Blocks) == 0 {
		d.messages = append(d.messages, anyllm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
		return nil
	}

	for _, block := range msg.Blocks {
		switch block.Type {
		case driver.BlockToolResult:
			if block.ToolResult != nil {
				d.messages = append(d.messages, anyllm.Message{
					Role:       anyllm.RoleTool,
					Content:    block.ToolResult.Output,
					ToolCallID: block.ToolResult.ToolCallID,
				})
			}
		case driver.BlockText:
			d.messages = append(d.messages, anyllm.Message{
				Role:    msg.Role,
				Content: block.Text,
			})
		}
	}
	return nil
}

// Chat calls the provider and returns streaming events on a channel.
// Uses blocking Completion internally — events are emitted all at once.
func (d *ChatDriver) Chat(ctx context.Context) (<-chan driver.StreamEvent, error) {
	d.mu.Lock()
	maxTokens := d.maxTokens
	params := anyllm.CompletionParams{
		Model:     d.model,
		MaxTokens: &maxTokens,
		Messages:  make([]anyllm.Message, 0, len(d.messages)+1),
		Tools:     d.tools,
	}
	if d.system != "" {
		params.Messages = append(params.Messages, anyllm.Message{
			Role:    anyllm.RoleSystem,
			Content: d.system,
		})
	}
	params.Messages = append(params.Messages, d.messages...)
	d.mu.Unlock()

	d.log.DebugContext(ctx, "sending completion",
		slog.String(telemetry.KeyComponent, "troupe-driver"),
		slog.String(telemetry.KeyStatus, d.model),
		slog.Int(telemetry.KeyCount, len(params.Messages)),
	)

	ch := make(chan driver.StreamEvent, 100)
	go func() {
		defer close(ch)

		completion, err := d.provider.Completion(ctx, params)
		if err != nil {
			d.log.WarnContext(ctx, "completion failed",
				slog.String(telemetry.KeyComponent, "troupe-driver"),
				slog.String(telemetry.KeyError, err.Error()),
			)
			ch <- driver.StreamEvent{Type: driver.EventError, Error: err.Error()}
			return
		}

		if len(completion.Choices) == 0 {
			ch <- driver.StreamEvent{Type: driver.EventError, Error: "no choices in completion response"}
			return
		}

		choice := completion.Choices[0]
		msg := choice.Message

		// Emit thinking/reasoning
		if msg.Reasoning != nil && msg.Reasoning.Content != "" {
			ch <- driver.StreamEvent{Type: driver.EventThinking, Thinking: msg.Reasoning.Content}
		}

		// Emit text
		if text := msg.ContentString(); text != "" {
			ch <- driver.StreamEvent{Type: driver.EventText, Text: text}
		}

		// Emit tool calls
		for i := range msg.ToolCalls {
			tc := &msg.ToolCalls[i]
			ch <- driver.StreamEvent{
				Type: driver.EventToolUse,
				ToolCall: &driver.ToolCall{
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: json.RawMessage(tc.Function.Arguments),
				},
			}
		}

		// Emit done with usage
		var usage *driver.Usage
		if completion.Usage != nil {
			usage = &driver.Usage{
				InputTokens:  completion.Usage.PromptTokens,
				OutputTokens: completion.Usage.CompletionTokens,
			}
		}
		ch <- driver.StreamEvent{Type: driver.EventDone, Usage: usage}

		d.log.InfoContext(ctx, "completion done",
			slog.String(telemetry.KeyComponent, "troupe-driver"),
			slog.String(telemetry.KeyStatus, choice.FinishReason),
		)
	}()

	return ch, nil
}

// AppendAssistant adds the assistant's response to conversation history.
func (d *ChatDriver) AppendAssistant(msg driver.RichMessage) {
	d.mu.Lock()
	defer d.mu.Unlock()

	am := anyllm.Message{Role: anyllm.RoleAssistant}

	if len(msg.Blocks) > 0 {
		var text strings.Builder
		var toolCalls []anyllm.ToolCall
		for _, b := range msg.Blocks {
			switch b.Type {
			case driver.BlockText:
				text.WriteString(b.Text)
			case driver.BlockToolUse:
				if b.ToolCall != nil {
					toolCalls = append(toolCalls, anyllm.ToolCall{
						ID:   b.ToolCall.ID,
						Type: "function",
						Function: anyllm.FunctionCall{
							Name:      b.ToolCall.Name,
							Arguments: string(b.ToolCall.Input),
						},
					})
				}
			}
		}
		if text.Len() > 0 {
			am.Content = text.String()
		}
		if len(toolCalls) > 0 {
			am.ToolCalls = toolCalls
		}
	} else {
		am.Content = msg.Content
	}

	d.messages = append(d.messages, am)
}

// SetSystemPrompt updates the system prompt at runtime.
func (d *ChatDriver) SetSystemPrompt(prompt string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.system = prompt
}

// ContextWindow returns the model's context window in tokens.
func (d *ChatDriver) ContextWindow() int {
	if strings.Contains(d.model, "opus") {
		return 1_000_000
	}
	return 200_000
}
