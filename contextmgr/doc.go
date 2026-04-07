// Package session manages conversation history and context for agent
// interactions, including compaction, relay seeding, and persistence.
//
// # Internal Structure
//
// The package is organized into the following files by concern:
//
//   - session.go     Core types: Entry, Session, New(), Append(), Entries()
//   - history.go     History with optional token budget, trim-on-append
//   - store.go       Disk persistence: Save/Load/Delete/Archive (JSON files)
//   - compact.go     Context compaction: Compact(), SeedSession(), ExtractSummaryText()
//   - sanitize.go    Load-time repair: fix corrupt tool_use blocks, orphan injection
//   - monitor.go     ContextMonitor: token usage tracking, spawn/swap thresholds
//   - relay.go       RelayManager: background session spawn, seamless context swap
//   - search.go      Fuzzy session search for telescope/attach picker
//   - import.go      Claude Code JSONL session import with budget-aware compaction
//
// # Dependency Flow
//
// Store.Load() calls Sanitize() which calls Compact() — a single load path
// that ensures every loaded session is clean and within budget.
//
// RelayManager wraps ContextMonitor and Store to orchestrate the full
// spawn -> seed -> swap -> archive lifecycle for context relay.
package contextmgr
