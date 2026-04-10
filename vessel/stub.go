package vessel

import (
	"context"
	"sync"

	"github.com/dpopsuev/battery/tool"
	"github.com/dpopsuev/troupe/signal"
)

var _ Vessel = (*StubVessel)(nil)

// StubVessel implements Vessel for testing. Records Close calls.
type StubVessel struct {
	mu       sync.Mutex
	tools    tool.Executor
	eventLog signal.EventLog
	workDir  string
	Closed   bool
}

// NewStubVessel creates a test vessel.
func NewStubVessel(tools tool.Executor, log signal.EventLog, workDir string) *StubVessel {
	return &StubVessel{tools: tools, eventLog: log, workDir: workDir}
}

func (v *StubVessel) Tools() tool.Executor      { return v.tools }
func (v *StubVessel) EventLog() signal.EventLog { return v.eventLog }
func (v *StubVessel) WorkDir() string           { return v.workDir }

func (v *StubVessel) Close(_ context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.Closed = true
	return nil
}
