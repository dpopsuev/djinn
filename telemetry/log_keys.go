// keys.go — Standardized slog field key constants.
//
// Use these instead of raw strings in slog calls. Enables consistent
// field names across all packages, grep-able, and refactor-safe.
//
// Orange (error paths): KeyError, KeyReason
// Yellow (success paths): KeyAction, KeyStatus, KeyDuration, KeyDecision
package telemetry

// LogKey is a named type for slog field keys. Prevents mixing with arbitrary strings.
type LogKey = string

// Identity keys — WHO is acting.
const (
	KeyComponent LogKey = "component" // package/subsystem name (e.g., "clutch", "policy", "broker")
	KeyAgent     LogKey = "agent"     // agent identifier
	KeyRole      LogKey = "role"      // agent role (executor, inspector, gensec)
)

// Action keys — WHAT is happening.
const (
	KeyAction   LogKey = "action"   // verb (e.g., "attach", "deny", "spawn", "merge")
	KeyTool     LogKey = "tool"     // tool name being called
	KeyDecision LogKey = "decision" // allow/deny/warn/skip
	KeyStatus   LogKey = "status"   // lifecycle state (e.g., "running", "paused", "done")
)

// Context keys — WHERE/WHY it happened.
const (
	KeyPath   LogKey = "path"   // file path, socket path, worktree path
	KeyReason LogKey = "reason" // why a decision was made or action taken
	KeyError  LogKey = "error"  // error description
)

// Metrics keys — HOW MUCH / HOW LONG.
const (
	KeyDuration  LogKey = "duration"   // elapsed time
	KeyTokensIn  LogKey = "tokens_in"  // input token count
	KeyTokensOut LogKey = "tokens_out" // output token count
	KeyTurn      LogKey = "turn"       // conversation turn number
)

// TUI keys — panel/field identification.
const (
	KeyPanel LogKey = "panel" // TUI panel identifier
	KeyField LogKey = "field" // field within a panel
)

// Waste detection keys — agent waste classification.
const (
	KeyWasteKind LogKey = "waste_kind" // WasteKind string (transportation, defect, etc.)
)

// Performance group key — nests perf metrics under "perf".
const KeyPerf LogKey = "perf"

// SymbolGraph keys.
const (
	KeyCount    LogKey = "count"    // item count
	KeyCallers  LogKey = "callers"  // caller count for a symbol
	KeyProvider LogKey = "provider" // provider name/index
	KeyFrom     LogKey = "from"     // transition source
	KeyTo       LogKey = "to"       // transition target
)

// Domain keys — workstream, sandbox, VCS.
const (
	KeyBackend      LogKey = "backend"        // sandbox backend name
	KeyLevel        LogKey = "level"          // sandbox isolation level
	KeyExitCode     LogKey = "exit_code"      // process exit code
	KeyBranch       LogKey = "branch"         // git branch name
	KeyTaskID       LogKey = "task_id"        // task identifier
	KeyWorkstreamID LogKey = "workstream_id"  // workstream identifier
	KeyIntentID     LogKey = "intent_id"      // operator intent identifier
	KeySource       LogKey = "source"         // event source (e.g., watchdog name)
	KeyQueuePos     LogKey = "queue_position" // position in queue
	KeyCache        LogKey = "cache"          // cache name (L1, L2)
	KeyScope        LogKey = "scope"          // cache scope (agent ID)
	KeyKey          LogKey = "key"            // cache key
	KeyBytes        LogKey = "bytes"          // data size in bytes
	KeyEntries      LogKey = "entries"        // entry count
	KeyScopes       LogKey = "scopes"         // scope count
	KeyVessel       LogKey = "vessel"         // vessel identifier
	KeyWorkDir      LogKey = "work_dir"       // workspace directory path
)
