// signal_consumer.go — Bridge budget signals to the signal bus (TSK-474).
//
// When BudgetMonitor detects exceeded signals, this emits them on the
// Djinn signal bus. The SignalInterpreter (GOL-44) picks them up and
// routes to GenSec for stochastic interpretation.
package review

import (
	"fmt"
	"strings"

	"github.com/dpopsuev/djinn/signal"
)

// EmitBudgetSignals publishes exceeded budget signals to the signal bus.
func EmitBudgetSignals(bus *signal.SignalBus, workstream string, signals []Signal) {
	exceeded := Exceeded(signals)
	if len(exceeded) == 0 {
		return
	}

	// Determine overall severity: any single exceeded = Yellow, 3+ = Red.
	level := signal.Yellow
	if len(exceeded) >= 3 { //nolint:mnd // 3 exceeded signals is a reasonable threshold for Red
		level = signal.Red
	}

	details := make([]string, 0, len(exceeded))
	for i := range exceeded {
		details = append(details, fmt.Sprintf("%s: %.0f (threshold: %.0f)", exceeded[i].Metric, exceeded[i].Value, exceeded[i].Threshold))
	}

	bus.Emit(signal.Signal{
		Workstream: workstream,
		Level:      level,
		Source:     "review-budget",
		Category:   signal.CategoryBudget,
		Message:    "budget exceeded: " + strings.Join(details, ", "),
	})
}
