package assertions

import (
	"testing"

	"github.com/dpopsuev/djinn/broker"
)

// AssertCordoned checks that the given scope is cordoned.
func AssertCordoned(t *testing.T, reg *broker.CordonRegistry, scope string) {
	t.Helper()
	if !reg.Overlaps(scope) {
		t.Fatalf("scope %q should be cordoned, but is not", scope)
	}
}

// AssertNotCordoned checks that the given scope is not cordoned.
func AssertNotCordoned(t *testing.T, reg *broker.CordonRegistry, scope string) {
	t.Helper()
	if reg.Overlaps(scope) {
		t.Fatalf("scope %q should not be cordoned, but is", scope)
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
