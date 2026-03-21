package assertions

import (
	"testing"

	"github.com/dpopsuev/djinn/broker"
)

// AssertCordoned checks that the given paths overlap with an active cordon.
func AssertCordoned(t *testing.T, reg *broker.CordonRegistry, paths ...string) {
	t.Helper()
	if len(reg.Overlaps(paths)) == 0 {
		t.Fatalf("paths %v should be cordoned, but are not", paths)
	}
}

// AssertNotCordoned checks that the given paths do not overlap with any active cordon.
func AssertNotCordoned(t *testing.T, reg *broker.CordonRegistry, paths ...string) {
	t.Helper()
	if len(reg.Overlaps(paths)) > 0 {
		t.Fatalf("paths %v should not be cordoned, but are", paths)
	}
}

// AssertCordonCount checks that the number of active cordons matches.
func AssertCordonCount(t *testing.T, reg *broker.CordonRegistry, want int) {
	t.Helper()
	got := len(reg.Active())
	if got != want {
		t.Fatalf("cordon count = %d, want %d", got, want)
	}
}
