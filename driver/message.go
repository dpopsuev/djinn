package driver

import (
	"encoding/json"
	"strings"
)

// Content block types within a message.
const (
	BlockText       = "text"
	BlockToolUse    = "tool_use"
	BlockToolResult = "tool_result"
	BlockThinking   = "thinking"
)

// ContentBlock is a single piece of content within a message.
// A message may contain multiple blocks (e.g., text + tool calls).
type ContentBlock struct {
	Type       string      `json:"type"`
	Text       string      `json:"text,omitempty"`
	ToolCall   *ToolCall   `json:"tool_call,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
	Thinking   string      `json:"thinking,omitempty"`
}

// ToolCall represents a request from the model to execute a tool.
type ToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolResult is the outcome of executing a tool.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output"`
	IsError    bool   `json:"is_error,omitempty"`
}

// Usage tracks token consumption for a response.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// StreamEvent types for real-time processing.
const (
	EventText     = "text"
	EventThinking = "thinking"
	EventToolUse  = "tool_use"
	EventDone     = "done"
	EventError    = "error"
)

// StreamEvent represents a single event in a streaming response.
type StreamEvent struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	Thinking string    `json:"thinking,omitempty"`
	ToolCall *ToolCall `json:"tool_call,omitempty"`
	Usage    *Usage    `json:"usage,omitempty"`
	Error    string    `json:"error,omitempty"`
}

// RichMessage extends Message with structured content blocks.
// Backward compatible: if Blocks is empty, Content is the plain text.
type RichMessage struct {
	Role    string         `json:"role"`
	Content string         `json:"content,omitempty"`
	Blocks  []ContentBlock `json:"blocks,omitempty"`
	Usage   *Usage         `json:"usage,omitempty"`
}

// TextContent returns the concatenated text from all text blocks,
// falling back to Content for plain messages.
func (m RichMessage) TextContent() string {
	if len(m.Blocks) == 0 {
		return m.Content
	}
	var sb strings.Builder
	for _, b := range m.Blocks {
		if b.Type == BlockText {
			sb.WriteString(b.Text)
		}
	}
	return sb.String()
}

// ToolCalls extracts all tool call blocks from the message.
func (m RichMessage) ToolCalls() []ToolCall {
	var calls []ToolCall
	for _, b := range m.Blocks {
		if b.Type == BlockToolUse && b.ToolCall != nil {
			calls = append(calls, *b.ToolCall)
		}
	}
	return calls
}

// HasToolCalls returns true if the message contains any tool calls.
func (m RichMessage) HasToolCalls() bool {
	for _, b := range m.Blocks {
		if b.Type == BlockToolUse {
			return true
		}
	}
	return false
}

// ThinkingContent returns the concatenated thinking blocks.
func (m RichMessage) ThinkingContent() string {
	var sb strings.Builder
	for _, b := range m.Blocks {
		if b.Type == BlockThinking {
			sb.WriteString(b.Thinking)
		}
	}
	return sb.String()
}

// ToMessage converts a RichMessage to a plain Message (lossy — drops blocks).
func (m RichMessage) ToMessage() Message {
	return Message{
		Role:    m.Role,
		Content: m.TextContent(),
	}
}

// NewTextBlock creates a text content block.
func NewTextBlock(text string) ContentBlock {
	return ContentBlock{Type: BlockText, Text: text}
}

// NewToolUseBlock creates a tool call content block.
func NewToolUseBlock(id, name string, input json.RawMessage) ContentBlock {
	return ContentBlock{
		Type:     BlockToolUse,
		ToolCall: &ToolCall{ID: id, Name: name, Input: input},
	}
}

// NewToolResultBlock creates a tool result content block.
func NewToolResultBlock(toolCallID, output string, isError bool) ContentBlock {
	return ContentBlock{
		Type:       BlockToolResult,
		ToolResult: &ToolResult{ToolCallID: toolCallID, Output: output, IsError: isError},
	}
}

// NewThinkingBlock creates a thinking content block.
func NewThinkingBlock(thinking string) ContentBlock {
	return ContentBlock{Type: BlockThinking, Thinking: thinking}
}
