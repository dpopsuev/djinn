package assertions

import (
	"testing"

	"github.com/dpopsuev/djinn/broker"
)

func TestAssertCordoned(t *testing.T) {
	reg := broker.NewCordonRegistry()
	reg.Set([]string{"auth"}, "broken", "agent-1")
	AssertCordoned(t, reg, "auth")
}

func TestAssertNotCordoned(t *testing.T) {
	reg := broker.NewCordonRegistry()
	AssertNotCordoned(t, reg, "auth")
}

func TestAssertCordonCount(t *testing.T) {
	reg := broker.NewCordonRegistry()
	AssertCordonCount(t, reg, 0)

	reg.Set([]string{"auth"}, "r1", "agent-1")
	reg.Set([]string{"payments"}, "r2", "agent-2")
	AssertCordonCount(t, reg, 2)

	reg.Clear([]string{"auth"})
	AssertCordonCount(t, reg, 1)
}
