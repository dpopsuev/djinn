// Package referee provides event-driven scoring of agent behavior.
//
// Referee subscribes to Troupe's EventLog and scores events using
// YAML-defined weighted rules (Scorecard). It watches and judges —
// it does NOT orchestrate.
//
// Scorecard: YAML rules with weights per event kind.
// Result: pass/fail based on threshold + chrono dump on failure.
//
// GOL-164
package referee
