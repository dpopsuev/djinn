package stubs

import (
	"context"
	"sync"

	"github.com/dpopsuev/djinn/driver"
)

// StubDriver implements driver.Driver with canned responses and logging.
type StubDriver struct {
	mu       sync.Mutex
	messages []driver.Message
	sendLog  []driver.Message
	recvCh   chan driver.Message
	startErr error
	sendErr  error
	stopErr  error
	started  bool
	stopped  bool
	sandbox  driver.SandboxHandle
}

// NewStubDriver creates a stub driver with canned response messages.
func NewStubDriver(messages ...driver.Message) *StubDriver {
	ch := make(chan driver.Message, len(messages))
	for _, m := range messages {
		ch <- m
	}
	close(ch)
	return &StubDriver{
		messages: messages,
		recvCh:   ch,
	}
}

func (d *StubDriver) Start(ctx context.Context, sandbox driver.SandboxHandle) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.startErr != nil {
		return d.startErr
	}
	d.started = true
	d.sandbox = sandbox
	return nil
}

func (d *StubDriver) Send(ctx context.Context, msg driver.Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.sendErr != nil {
		return d.sendErr
	}
	d.sendLog = append(d.sendLog, msg)
	return nil
}

func (d *StubDriver) Recv(ctx context.Context) <-chan driver.Message {
	return d.recvCh
}

func (d *StubDriver) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.stopErr != nil {
		return d.stopErr
	}
	d.stopped = true
	return nil
}

// SendLog returns a copy of all messages sent to this driver.
func (d *StubDriver) SendLog() []driver.Message {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]driver.Message, len(d.sendLog))
	copy(out, d.sendLog)
	return out
}

// Started reports whether Start was called successfully.
func (d *StubDriver) Started() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.started
}

// Sandbox returns the sandbox handle passed to Start.
func (d *StubDriver) Sandbox() driver.SandboxHandle {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sandbox
}

// SetStartErr injects an error for the next Start call.
func (d *StubDriver) SetStartErr(err error) { d.mu.Lock(); d.startErr = err; d.mu.Unlock() }

// SetSendErr injects an error for the next Send call.
func (d *StubDriver) SetSendErr(err error) { d.mu.Lock(); d.sendErr = err; d.mu.Unlock() }

// SetStopErr injects an error for the next Stop call.
func (d *StubDriver) SetStopErr(err error) { d.mu.Lock(); d.stopErr = err; d.mu.Unlock() }
