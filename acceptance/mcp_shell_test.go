// mcp_shell_test.go — E2E acceptance tests for the Djinn MCP Shell.
//
// The Djinn MCP Shell is a single MCP server that facades all Spine slots.
// The agent connects to ONE endpoint and sees slot tools (WorkTracker,
// CodeEditor, etc.). Djinn routes each call to the appropriate backend
// (Scribe MCP, Locus MCP, built-in Go), manages backend lifecycle,
// applies role-based filtering, and runs middleware (gates, policy).
//
// These tests verify the contract between the agent and the Shell.
// They do NOT test individual backends — those have their own tests.
package acceptance

import (
	"testing"
)

func TestMCPShell_AgentSeesOnlySlotTools(t *testing.T) {
	// Scenario: Agent connects to single Djinn MCP endpoint
	//   Given Djinn starts with WorkTracker and CodeEditor slots configured
	//   When an agent connects via MCP stdio
	//   Then the agent sees exactly the slot tools declared for its role
	//   And the agent does NOT see raw backend tool names (artifact, codograph)
	t.Skip("not implemented — depends on DJN-TSK-135 (Djinn MCP Shell)")
}

func TestMCPShell_SlotCallRoutesToBackend(t *testing.T) {
	// Scenario: Slot call routes to backend MCP server
	//   Given WorkTracker slot is backed by Scribe MCP
	//   When the agent calls WorkTracker.List
	//   Then Djinn starts Scribe as a child process (if not running)
	//   And routes the call to Scribe's artifact list tool
	//   And returns the result through the WorkTracker slot
	t.Skip("not implemented — depends on DJN-TSK-135 (Djinn MCP Shell)")
}

func TestMCPShell_BackendStartedLazilyOnFirstAccess(t *testing.T) {
	// Scenario: Backend server started lazily on first access
	//   Given Scribe MCP is declared but not running
	//   When the agent calls WorkTracker.List for the first time
	//   Then Djinn starts the Scribe process
	//   And the call succeeds
	//   And subsequent calls reuse the running process
	t.Skip("not implemented — depends on DJN-TSK-135 (Djinn MCP Shell)")
}

func TestMCPShell_RoleFiltersVisibleSlots(t *testing.T) {
	// Scenario: Role filters visible slots
	//   Given the current role is GenSec with slots [WorkTracker, IssueTracker]
	//   When the agent calls tools/list on the Djinn MCP
	//   Then only WorkTracker and IssueTracker tools are returned
	//   And CodeEditor, Shell, FileSearch tools are NOT visible
	t.Skip("not implemented — depends on DJN-TSK-135 (Djinn MCP Shell)")
}

func TestMCPShell_MiddlewareFiresOnSlotCall(t *testing.T) {
	// Scenario: Middleware fires on slot call
	//   Given the Executor role has a quality gate middleware
	//   When the agent calls CodeEditor.WriteFile
	//   Then PolicyEnforcer checks the CapabilityToken
	//   And the write is allowed or denied based on the token
	t.Skip("not implemented — depends on DJN-TSK-135 (Djinn MCP Shell)")
}

func TestMCPShell_BackendCrashAndRecover(t *testing.T) {
	// Scenario: Backend crashes and recovers
	//   Given Scribe MCP is running as a child process
	//   When the Scribe process crashes
	//   And the agent calls WorkTracker.List
	//   Then Djinn detects the crash and restarts Scribe
	//   And the call succeeds after reconnect
	t.Skip("not implemented — depends on DJN-TSK-135 (Djinn MCP Shell)")
}

func TestMCPShell_HealthReflectsBackendStatus(t *testing.T) {
	// Scenario: Health reflects backend status
	//   Given Locus MCP failed to start (binary not found)
	//   When the agent calls tools/list
	//   Then ArchExplorer tools are NOT listed (backend offline)
	//   And health shows ArchExplorer as offline, not warning
	t.Skip("not implemented — depends on DJN-TSK-135 (Djinn MCP Shell)")
}
