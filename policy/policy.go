// Package policy implements Agent Call Mediation — the ToolPolicyEnforcer
// gates every agent call from Agent Space to User Space.
//
// Agent Space (untrusted LLM) → agent call → User Space (Djinn runtime)
//
// The ToolPolicyEnforcer checks capability tokens before tool execution.
// Denied calls return errors to the agent, not crashes.
package policy

import "encoding/json"

// CapabilityToken defines what an agent is allowed to do.
// Created at workspace load, immutable by the agent.
type CapabilityToken struct {
	WritablePaths []string // workspace repo paths the agent can write to
	DeniedPaths   []string // config paths, always denied regardless of other rules
	AllowedTools  []string // tool whitelist (empty = all tools allowed)
	Tier          string   // eco, mod, sys — privilege level
}

// ToolPolicyEnforcer gates every agent call. Returns nil if allowed,
// error with reason if denied.
type ToolPolicyEnforcer interface {
	Check(token CapabilityToken, tool string, input json.RawMessage) error
}
