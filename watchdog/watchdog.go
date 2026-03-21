// Package watchdog defines lateral independent observers that monitor
// agent workstreams for budget violations, deadlocks, security issues,
// quality drift, and task drift. Watchdogs consume and emit signals
// on the SignalBus.
package watchdog

import (
	"context"
	"errors"
)

// Watchdog categories.
const (
	CategorySecurity = "security"
	CategoryBudget   = "budget"
	CategoryDeadlock = "deadlock"
	CategoryQuality  = "quality"
	CategoryDrift    = "drift"
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
