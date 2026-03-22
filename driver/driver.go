package driver

import "context"

// SandboxHandle is a string identifier for a sandbox instance.
type SandboxHandle = string

// DriverConfig holds configuration for creating a driver.
type DriverConfig struct {
	Model       string
	MaxTokens   int
	Temperature float64
}

// Message roles.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message represents a message exchanged between the runtime and an agent.
type Message struct {
	Role    string // one of Role* constants
	Content string
}

// Driver is the interface for headless/subprocess LLM communication.
type Driver interface {
	Start(ctx context.Context, sandbox SandboxHandle) error
	Send(ctx context.Context, msg Message) error
	Recv(ctx context.Context) <-chan Message
	Stop(ctx context.Context) error
}

// ChatDriver is the interface for interactive REPL-style LLM communication.
// Supports streaming, tool calling, rich messages, and conversation history.
// Any model backend (Claude, OpenAI, Ollama, Gemini) implements this.
type ChatDriver interface {
	Start(ctx context.Context, sandbox SandboxHandle) error
	Stop(ctx context.Context) error
	Send(ctx context.Context, msg Message) error
	SendRich(ctx context.Context, msg RichMessage) error
	Chat(ctx context.Context) (<-chan StreamEvent, error)
	AppendAssistant(msg RichMessage)
	SetSystemPrompt(prompt string) // update prompt at runtime (workspace switch)
}
