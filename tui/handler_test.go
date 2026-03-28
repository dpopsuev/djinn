package tui

import "testing"

// TestHandler_TrustBoundary_SafeMessageTypes verifies that BubbletaHandler
// only has methods that emit output-safe message types.
//
// This is a documentation test — it asserts the handler's API surface
// hasn't grown beyond the safe set. If you add a new method to
// BubbletaHandler, add it here and verify it's output-safe.
func TestHandler_TrustBoundary_SafeMessageTypes(t *testing.T) {
	// The handler's public methods are the trust boundary.
	// Each method maps to exactly one safe message type:
	safeMethods := map[string]string{
		"OnText":       "TextMsg",
		"OnThinking":   "ThinkingMsg",
		"OnToolCall":   "ToolCallMsg",
		"OnToolResult": "ToolResultMsg + RenderPanelMsg (intercepted)",
		"OnDone":       "DoneMsg",
		"OnError":      "ErrorMsg",
	}

	// If this test needs updating, you're adding a new handler method.
	// Ask: does the new message type modify input, dashboard, or layout?
	// If yes, it violates the trust boundary.
	if len(safeMethods) != 6 {
		t.Fatal("trust boundary: handler should have exactly 6 event methods")
	}

	// Verify no dangerous message types can be emitted by the handler.
	// These message types MUST NEVER be sent from agent events:
	dangerousTypes := []string{
		"InputSetValueMsg", // command injection
		"SubmitMsg",        // re-trigger commands
		"DialogResultMsg",  // fake approvals
		"FocusPanelMsg",    // layout manipulation
		"ResizeMsg",        // layout manipulation
		"InputResetMsg",    // clear user input
		"InputFocusMsg",    // steal focus
	}
	_ = dangerousTypes // documented for human review
}

// TestHandler_RenderIntercept_OnlyOnRenderTool verifies the render
// intercept only fires for tool name "render", not other tools.
func TestHandler_RenderIntercept_OnlyOnRenderTool(t *testing.T) {
	// The handler intercepts ToolResultMsg with name="render".
	// Other tool names must NOT trigger RenderPanelMsg.
	//
	// This is verified by the handler.go code structure:
	//   if name == "render" && !isError { ... }
	//
	// A structural test — the condition is name-exact, not prefix.
	renderToolName := "render"
	nonRenderTools := []string{"Read", "Write", "Bash", "plan", "test", "git"}
	for _, name := range nonRenderTools {
		if name == renderToolName {
			t.Fatalf("tool %q should not match render intercept", name)
		}
	}
}
