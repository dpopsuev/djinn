package taskforce

import "github.com/dpopsuev/djinn/watchdog"

// WatchdogThreshold controls how aggressively watchdogs respond.
type WatchdogThreshold string

const (
	ThresholdRelaxed  WatchdogThreshold = "relaxed"
	ThresholdStandard WatchdogThreshold = "standard"
	ThresholdStrict   WatchdogThreshold = "strict"
)

// WatchdogAssignment maps complexity bands to active watchdog categories.
type WatchdogAssignment struct {
	Active    []string          // active watchdog categories
	Threshold WatchdogThreshold // sensitivity level
}

// DefaultWatchdogAssignment returns the band-specific watchdog configuration
// per SPC-15's assignment table.
func DefaultWatchdogAssignment(band ComplexityBand) WatchdogAssignment {
	switch band {
	case Clear:
		return WatchdogAssignment{
			Active:    []string{watchdog.CategorySecurity, watchdog.CategoryBudget},
			Threshold: ThresholdRelaxed,
		}
	case Complicated:
		return WatchdogAssignment{
			Active:    []string{watchdog.CategorySecurity, watchdog.CategoryBudget, watchdog.CategoryQuality},
			Threshold: ThresholdStandard,
		}
	case Complex:
		return WatchdogAssignment{
			Active:    []string{watchdog.CategorySecurity, watchdog.CategoryBudget, watchdog.CategoryQuality, watchdog.CategoryDeadlock, watchdog.CategoryDrift},
			Threshold: ThresholdStrict,
		}
	case Chaotic:
		return WatchdogAssignment{
			Active:    []string{watchdog.CategorySecurity, watchdog.CategoryBudget, watchdog.CategoryDeadlock},
			Threshold: ThresholdRelaxed,
		}
	default:
		return WatchdogAssignment{
			Active:    []string{watchdog.CategorySecurity, watchdog.CategoryBudget},
			Threshold: ThresholdStandard,
		}
	}
}
