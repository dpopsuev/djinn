package repl

import (
	"testing"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/tui"
)

// Compile-time check that BubbletaHandler implements agent.EventHandler.
var _ agent.EventHandler = &tui.BubbletaHandler{}

func TestBubbletaHandler_ImplementsEventHandler(t *testing.T) {
	_ = &tui.BubbletaHandler{}
}
