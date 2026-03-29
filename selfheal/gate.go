// gate.go — SelfHealGate validates fixes via trace comparison (TSK-490).
//
// After a fix is built and tested, the gate compares trace metrics
// before and after to verify the fix actually improved things.
package selfheal

import (
	"github.com/dpopsuev/djinn/trace"
)

// GateVerdict is the result of fix validation.
type GateVerdict string

const (
	GatePass   GateVerdict = "pass"   // fix improved metrics
	GateFail   GateVerdict = "fail"   // fix didn't help or regressed
	GateUnsure GateVerdict = "unsure" // insufficient data to judge
)

// GateResult captures the validation outcome.
type GateResult struct {
	Verdict         GateVerdict       `json:"verdict"`
	Reason          string            `json:"reason"`
	ErrorRateBefore float64           `json:"error_rate_before"`
	ErrorRateAfter  float64           `json:"error_rate_after"`
	Diff            *trace.DiffResult `json:"diff,omitempty"`
}

// Validate compares a pre-fix archive against the current ring state.
// Returns pass if error rate decreased or no new errors appeared.
func Validate(before *trace.Archive, afterRing *trace.Ring) *GateResult {
	after := trace.Export(afterRing, "")
	diff := trace.Diff(before, after)

	result := &GateResult{
		ErrorRateBefore: diff.ErrorRateBefore,
		ErrorRateAfter:  diff.ErrorRateAfter,
		Diff:            diff,
	}

	// Not enough data to judge.
	if diff.EventCountAfter < 5 { //nolint:mnd // minimum events for meaningful comparison
		result.Verdict = GateUnsure
		result.Reason = "insufficient trace data after fix"
		return result
	}

	// Error rate improved.
	if diff.ErrorRateAfter < diff.ErrorRateBefore {
		result.Verdict = GatePass
		result.Reason = "error rate improved"
		return result
	}

	// New errors appeared.
	if len(diff.NewErrors) > 0 {
		result.Verdict = GateFail
		result.Reason = "new errors appeared after fix"
		return result
	}

	// Error rate unchanged but no new errors — marginal pass.
	if diff.ErrorRateAfter <= diff.ErrorRateBefore {
		result.Verdict = GatePass
		result.Reason = "no regression detected"
		return result
	}

	result.Verdict = GateFail
	result.Reason = "error rate increased"
	return result
}
