package assertions

import (
	"testing"

	"github.com/dpopsuev/djinn/broker"
)

func TestAssertCordoned(t *testing.T) {
	reg := broker.NewCordonRegistry()
	reg.Set("auth", "broken")
	AssertCordoned(t, reg, "auth")
}

func TestAssertNotCordoned(t *testing.T) {
	reg := broker.NewCordonRegistry()
	AssertNotCordoned(t, reg, "auth")
}

func TestAssertCordonCount(t *testing.T) {
	reg := broker.NewCordonRegistry()
	AssertCordonCount(t, reg, 0)

	reg.Set("auth", "r1")
	reg.Set("payments", "r2")
	AssertCordonCount(t, reg, 2)

	reg.Clear("auth")
	AssertCordonCount(t, reg, 1)
}
