// Package watchdog defines lateral independent observers that monitor
// agent workstreams for budget violations, deadlocks, security issues,
// quality drift, and task drift. Watchdogs consume and emit signals
// on the SignalBus.
package telemetry

import (
	"context"
	"errors"
)

// Watchdog-specific categories (CategorySecurity, CategoryBudget, CategoryDrift
// are defined in sig_signal.go — shared with signal system).
const (
	CategoryDeadlock = "deadlock"
	CategoryQuality  = "quality"
)

// Watchdog levels (host vs container).
const (
	LevelHost      = "host"
	LevelContainer = "container"
)

// Sentinel errors for watchdog operations.
var (
	ErrBudgetExceeded   = errors.New("budget exceeded")
	ErrDeadlockDetected = errors.New("deadlock detected: no signals within timeout")
)

// Watchdog is the interface for all lateral observers.
type Watchdog interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Name() string
	Category() string
}
