// Package config provides the Configurable interface and ConfigRegistry
// for runtime configuration using the Memento pattern.
package config

import "errors"

// Configurable is implemented by any component that can snapshot and restore
// its configuration. Each implementation owns its config shape.
type Configurable interface {
	ConfigKey() string // section name: "driver", "session", "mode"
	Snapshot() any     // returns serializable state (never secrets)
	Apply(v any) error // restores from a previously captured snapshot
}

// Sentinel errors.
var (
	ErrConfigApply = errors.New("config apply failed")
)
