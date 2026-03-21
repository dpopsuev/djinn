package composition

import (
	"errors"
	"fmt"
	"time"
)

// Channel types for reporting edges.
const (
	ChannelPromotionGate = "promotion_gate"
	ChannelDelegation    = "delegation"
	ChannelObservation   = "observation"
	ChannelEscalation    = "escalation"
	ChannelBroadcast     = "broadcast"
)

// Sentinel errors for formation validation.
var (
	ErrEmptyFormation   = errors.New("formation must have at least one unit")
	ErrInvalidEdgeIndex = errors.New("edge references non-existent unit index")
	ErrNoFormationName  = errors.New("formation name is required")
)

// Edge defines a reporting relationship between two units.
type Edge struct {
	From    int    // source unit index
	To      int    // target unit index
	Channel string // one of Channel* constants
}

// Formation is a named composition of units with reporting edges.
type Formation struct {
	Name      string
	Units     []Unit
	Edges     []Edge
	Budget    Budget
	Observers []Unit
}

// Validate checks the formation's structural integrity.
func (f Formation) Validate() error {
	if f.Name == "" {
		return ErrNoFormationName
	}
	if len(f.Units) == 0 {
		return ErrEmptyFormation
	}
	for i, u := range f.Units {
		if err := u.Validate(); err != nil {
			return fmt.Errorf("unit[%d]: %w", i, err)
		}
	}
	for i, e := range f.Edges {
		if e.From < 0 || e.From >= len(f.Units) || e.To < 0 || e.To >= len(f.Units) {
			return fmt.Errorf("edge[%d]: %w: from=%d to=%d, units=%d",
				i, ErrInvalidEdgeIndex, e.From, e.To, len(f.Units))
		}
	}
	if err := ValidateScopeDisjointness(f.Units); err != nil {
		return err
	}
	return nil
}

// DivideBudget splits the formation's aggregate budget equally among units
// that have no individual budget set.
func (f *Formation) DivideBudget() {
	if f.Budget.IsZero() {
		return
	}
	var needBudget int
	for _, u := range f.Units {
		if u.Budget.IsZero() && !u.IsObserver() {
			needBudget++
		}
	}
	if needBudget == 0 {
		return
	}
	share := Budget{
		Tokens:    f.Budget.Tokens / needBudget,
		WallClock: f.Budget.WallClock / time.Duration(needBudget),
	}
	for i := range f.Units {
		if f.Units[i].Budget.IsZero() && !f.Units[i].IsObserver() {
			f.Units[i].Budget = share
		}
	}
}
