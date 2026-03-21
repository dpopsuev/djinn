package taskforce

import (
	"time"

	"github.com/dpopsuev/djinn/composition"
)

// TaskForce is an assembled, immutable agent formation for a specific task.
type TaskForce struct {
	ID              string
	Band            ComplexityBand
	Formation       composition.Formation
	Budget          composition.Budget
	WatchdogConfig  WatchdogAssignment
	CreatedAt       time.Time
}
