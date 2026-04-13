// Package hook provides the unified event dispatch system for Djinn.
//
// YAML-defined, deterministic, zero-LLM hooks that fire on events.
// Three phases: pre_tool_use (synchronous gate, can deny), post_tool_use
// (synchronous recorder), event (async, reacts to EventLog events).
//
// Replaces 4 scattered mechanisms: agent/hook.go (shell hooks),
// MetricsHandler (SignalBus bridge), SignalBus.OnSignal (pub/sub),
// SlotTable.SpawnOn (spawn triggers).
//
// GOL-161
package hook
