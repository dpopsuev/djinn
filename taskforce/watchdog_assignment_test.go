package taskforce

import (
	"testing"

	"github.com/dpopsuev/djinn/watchdog"
)

func TestDefaultWatchdogAssignment_PerBand(t *testing.T) {
	tests := []struct {
		band      ComplexityBand
		wantCount int
		threshold WatchdogThreshold
	}{
		{Clear, 2, ThresholdRelaxed},
		{Complicated, 3, ThresholdStandard},
		{Complex, 5, ThresholdStrict},
		{Chaotic, 3, ThresholdRelaxed},
	}

	for _, tt := range tests {
		wda := DefaultWatchdogAssignment(tt.band)
		if len(wda.Active) != tt.wantCount {
			t.Fatalf("band %v: active = %d, want %d", tt.band, len(wda.Active), tt.wantCount)
		}
		if wda.Threshold != tt.threshold {
			t.Fatalf("band %v: threshold = %q, want %q", tt.band, wda.Threshold, tt.threshold)
		}
	}
}

func TestDefaultWatchdogAssignment_SecurityAlwaysPresent(t *testing.T) {
	for _, band := range []ComplexityBand{Clear, Complicated, Complex, Chaotic} {
		wda := DefaultWatchdogAssignment(band)
		found := false
		for _, cat := range wda.Active {
			if cat == watchdog.CategorySecurity {
				found = true
			}
		}
		if !found {
			t.Fatalf("band %v: security watchdog missing", band)
		}
	}
}
