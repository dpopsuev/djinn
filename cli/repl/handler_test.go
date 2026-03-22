package repl

import (
	"testing"

	"github.com/dpopsuev/djinn/agent"
)

// Compile-time check that BubbletaHandler implements agent.EventHandler.
var _ agent.EventHandler = &BubbletaHandler{}

func TestBubbletaHandler_ImplementsEventHandler(t *testing.T) {
	// This test passes if the compile-time check above compiles.
	// BubbletaHandler must satisfy agent.EventHandler.
	_ = &BubbletaHandler{}
}
