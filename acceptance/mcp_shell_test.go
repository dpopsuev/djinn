// mcp_shell_test.go — E2E acceptance tests for the Djinn MCP Shell.
//
// The Djinn MCP Shell is a single MCP server that facades all ToolArsenal capabilities.
// The agent connects to ONE endpoint and sees capability tools (WorkTracking,
// FileEditing, etc.). Djinn routes each call to the appropriate backend
// (Scribe MCP, Locus MCP, built-in Go), manages backend lifecycle,
// applies role-based filtering, and runs middleware (gates, policy).
//
// These tests verify the contract between the agent and the Shell.
// They do NOT test individual backends — those have their own tests.
package acceptance

import (
	"testing"
)

func TestMCPShell_AgentSeesOnlyCapabilityTools(t *testing.T) {
	// Scenario: Agent connects to single Djinn MCP endpoint
	//   Given Djinn starts with WorkTracking and FileEditing capabilities configured
	//   When an agent connects via MCP stdio
	//   Then the agent sees exactly the capability tools declared for its role
	//   And the agent does NOT see raw backend tool names (artifact, codograph)
	t.Skip("not implemented — depends on DJN-TSK-135 (Djinn MCP Shell)")
}

func TestMCPShell_CapabilityCallRoutesToBackend(t *testing.T) {
	// Scenario: Capability call routes to backend MCP server
	//   Given WorkTracking capability is backed by Scribe MCP
	//   When the agent calls WorkTracking.List
	//   Then Djinn starts Scribe as a child process (if not running)
	//   And routes the call to Scribe's artifact list tool
	//   And returns the result through the WorkTracking capability
	t.Skip("not implemented — depends on DJN-TSK-135 (Djinn MCP Shell)")
}

func TestMCPShell_BackendStartedLazilyOnFirstAccess(t *testing.T) {
	// Scenario: Backend server started lazily on first access
	//   Given Scribe MCP is declared but not running
	//   When the agent calls WorkTracking.List for the first time
	//   Then Djinn starts the Scribe process
	//   And the call succeeds
	//   And subsequent calls reuse the running process
	t.Skip("not implemented — depends on DJN-TSK-135 (Djinn MCP Shell)")
}

func TestMCPShell_RoleFiltersVisibleCapabilities(t *testing.T) {
	// Scenario: Role filters visible capabilities
	//   Given the current role is GenSec with capabilities [WorkTracking, IssueTracking]
	//   When the agent calls tools/list on the Djinn MCP
	//   Then only WorkTracking and IssueTracking tools are returned
	//   And FileEditing, ShellExecution, FileSearching tools are NOT visible
	t.Skip("not implemented — depends on DJN-TSK-135 (Djinn MCP Shell)")
}

func TestMCPShell_MiddlewareFiresOnCapabilityCall(t *testing.T) {
	// Scenario: Middleware fires on capability call
	//   Given the Executor role has a quality gate middleware
	//   When the agent calls FileEditing.WriteFile
	//   Then ToolPolicyEnforcer checks the CapabilityToken
	//   And the write is allowed or denied based on the token
	t.Skip("not implemented — depends on DJN-TSK-135 (Djinn MCP Shell)")
}

func TestMCPShell_BackendCrashAndRecover(t *testing.T) {
	// Scenario: Backend crashes and recovers
	//   Given Scribe MCP is running as a child process
	//   When the Scribe process crashes
	//   And the agent calls WorkTracking.List
	//   Then Djinn detects the crash and restarts Scribe
	//   And the call succeeds after reconnect
	t.Skip("not implemented — depends on DJN-TSK-135 (Djinn MCP Shell)")
}

func TestMCPShell_HealthReflectsBackendStatus(t *testing.T) {
	// Scenario: Health reflects backend status
	//   Given Locus MCP failed to start (binary not found)
	//   When the agent calls tools/list
	//   Then ArchitectureAnalysis tools are NOT listed (backend offline)
	//   And health shows ArchitectureAnalysis as offline, not warning
	t.Skip("not implemented — depends on DJN-TSK-135 (Djinn MCP Shell)")
}
