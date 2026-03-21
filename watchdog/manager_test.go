package watchdog

import (
	"context"
	"testing"

	"github.com/dpopsuev/djinn/signal"
)

func TestManager_Lifecycle(t *testing.T) {
	bus := signal.NewSignalBus()
	mgr := NewManager(bus)

	mgr.Register(NewSecurityWatchdog())
	mgr.Register(NewQualityWatchdog())

	if len(mgr.Watchdogs()) != 2 {
		t.Fatalf("Watchdogs = %d, want 2", len(mgr.Watchdogs()))
	}

	ctx := context.Background()
	if err := mgr.StartAll(ctx); err != nil {
		t.Fatalf("StartAll: %v", err)
	}
	if err := mgr.StopAll(ctx); err != nil {
		t.Fatalf("StopAll: %v", err)
	}
}

func TestManager_Empty(t *testing.T) {
	bus := signal.NewSignalBus()
	mgr := NewManager(bus)

	ctx := context.Background()
	if err := mgr.StartAll(ctx); err != nil {
		t.Fatalf("StartAll empty: %v", err)
	}
	if err := mgr.StopAll(ctx); err != nil {
		t.Fatalf("StopAll empty: %v", err)
	}
}
