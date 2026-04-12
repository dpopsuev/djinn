package vezir

import (
	"context"
	"sync"
)

var _ Vezir = (*StubVezir)(nil)

// StubVezir implements Vezir for testing. Records restart calls.
type StubVezir struct {
	mu       sync.Mutex
	health   HealthReport
	Restarts []string
}

func NewStubVezir() *StubVezir {
	return &StubVezir{
		health: HealthReport{
			Substrate: ProcessState{Running: true},
			TUI:       ProcessState{Running: true},
		},
	}
}

func (v *StubVezir) Start(_ context.Context) error { select {} }

func (v *StubVezir) Health() HealthReport {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.health
}

func (v *StubVezir) Restart(_ context.Context, process string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.Restarts = append(v.Restarts, process)
	return nil
}
