// agent_config.go — Single source of truth for test AI agent configuration.
//
// All E2E tests that spawn an AI agent use DefaultTestAgent() instead of
// hardcoding driver names. Override via DJINN_TEST_AGENT env var.
// Default: "cursor" (Cursor Agent CLI).
package testkit

import "os"

// Supported test agent values.
const (
	AgentCursor = "cursor"
	AgentClaude = "claude"
	AgentOllama = "ollama"
	AgentCodex  = "codex"
	AgentGemini = "gemini"
)

// DefaultTestAgent returns the configured test agent name.
// Reads DJINN_TEST_AGENT env var. Default: "cursor".
func DefaultTestAgent() string {
	if agent := os.Getenv("DJINN_TEST_AGENT"); agent != "" {
		return agent
	}
	return AgentCursor
}

// DriverForAgent maps an agent name to the Djinn driver name.
func DriverForAgent(agent string) string {
	switch agent {
	case AgentClaude:
		return "claude"
	case AgentOllama:
		return "ollama"
	case AgentCodex:
		return "codex"
	case AgentGemini:
		return "gemini"
	default:
		return "cursor"
	}
}

// AgentBinary returns the expected CLI binary name for the agent.
func AgentBinary(agent string) string {
	switch agent {
	case AgentClaude:
		return "claude"
	case AgentOllama:
		return "ollama"
	case AgentCodex:
		return "codex"
	case AgentGemini:
		return "gemini"
	default:
		return "cursor" // Cursor Agent CLI
	}
}
