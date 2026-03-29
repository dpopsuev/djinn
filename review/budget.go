// budget.go — BudgetHeuristic interface + BudgetMonitor (TSK-437).
//
// The contract all 4 heuristic tiers implement. BudgetMonitor runs registered
// heuristics against a DiffSnapshot and returns exceeded signals.
package review

import "context"

// Signal represents a single heuristic measurement.
type Signal struct {
	Metric    string  `json:"metric"`    // e.g. "files_touched"
	Value     float64 `json:"value"`     // current measurement
	Threshold float64 `json:"threshold"` // configured limit
	Exceeded  bool    `json:"exceeded"`  // value >= threshold
	Detail    string  `json:"detail"`    // human-readable context
}

// DiffSnapshot captures the current state of changes for heuristic evaluation.
type DiffSnapshot struct {
	ChangedFiles []string // modified files
	AddedFiles   []string // newly created files
	DeletedFiles []string // removed files
	LOCDelta     int      // net lines changed
	PackagesHit  []string // unique directories touched
	WorkDir      string   // repo root for file content access + LSP
}

// BudgetHeuristic evaluates a diff against a specific quality dimension.
// Each tier (git diff, symbols, Locus graph, quality) implements this interface.
type BudgetHeuristic interface {
	Name() string
	Evaluate(ctx context.Context, diff *DiffSnapshot) ([]Signal, error)
}

// BudgetMonitor orchestrates multiple heuristics.
type BudgetMonitor struct {
	heuristics []BudgetHeuristic
}

// NewBudgetMonitor creates a monitor with no registered heuristics.
func NewBudgetMonitor() *BudgetMonitor {
	return &BudgetMonitor{}
}

// Register adds a heuristic to the monitor.
func (bm *BudgetMonitor) Register(h BudgetHeuristic) {
	bm.heuristics = append(bm.heuristics, h)
}

// Check runs all registered heuristics and returns signals with any exceeded.
func (bm *BudgetMonitor) Check(ctx context.Context, diff *DiffSnapshot) []Signal {
	var all []Signal
	for _, h := range bm.heuristics {
		signals, err := h.Evaluate(ctx, diff)
		if err != nil {
			all = append(all, Signal{
				Metric:   h.Name() + "_error",
				Exceeded: false,
				Detail:   "heuristic failed: " + err.Error(),
			})
			continue
		}
		all = append(all, signals...)
	}
	return all
}

// Exceeded returns only the signals that breached their threshold.
func Exceeded(signals []Signal) []Signal {
	out := make([]Signal, 0, len(signals))
	for i := range signals {
		if signals[i].Exceeded {
			out = append(out, signals[i])
		}
	}
	return out
}
