// Package policy implements Agent Call Mediation — the ToolPolicyEnforcer
// gates every agent call from Agent Space to User Space.
//
// Agent Space (untrusted LLM) → agent call → User Space (Djinn runtime)
//
// The ToolPolicyEnforcer checks capability tokens before tool execution.
// Denied calls return errors to the agent, not crashes.
package policy

import batterypolicy "github.com/dpopsuev/battery/policy"

// CapabilityToken defines what an agent is allowed to do.
// Created at workspace load, immutable by the agent.
// Aliased from battery/policy — single source of truth.
type CapabilityToken = batterypolicy.CapabilityToken

// ToolPolicyEnforcer gates every agent call. Returns nil if allowed,
// error with reason if denied.
// Aliased from battery/policy.Enforcer — single source of truth.
type ToolPolicyEnforcer = batterypolicy.Enforcer
