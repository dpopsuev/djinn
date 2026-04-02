// keys.go — Standardized slog field key constants.
//
// Use these instead of raw strings in slog calls. Enables consistent
// field names across all packages, grep-able, and refactor-safe.
//
// Orange (error paths): KeyError, KeyReason
// Yellow (success paths): KeyAction, KeyStatus, KeyDuration, KeyDecision
package djinnlog

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
