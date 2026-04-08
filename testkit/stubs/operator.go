package stubs

import (
	"sync"

	"github.com/dpopsuev/djinn/broker"
)

// StubOperatorPort implements broker.OperatorPort for testing.
type StubOperatorPort struct {
	mu          sync.Mutex
	intentCh    chan broker.Intent
	permRespCh  chan broker.PermissionResponse
	events      []broker.Event
	permissions []broker.PermissionPayload
	andons      []broker.AndonBoard
	results     []broker.Result
	handler     func(broker.Intent)
}

// NewStubOperatorPort creates a new stub operator port.
func NewStubOperatorPort() *StubOperatorPort {
	return &StubOperatorPort{
		intentCh:   make(chan broker.Intent, 10),
		permRespCh: make(chan broker.PermissionResponse, 10),
	}
}

func (p *StubOperatorPort) OnIntent(handler func(broker.Intent)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handler = handler
}

func (p *StubOperatorPort) EmitProgress(event broker.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
}

func (p *StubOperatorPort) EmitPermission(payload broker.PermissionPayload) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.permissions = append(p.permissions, payload)
}

func (p *StubOperatorPort) EmitAndon(board broker.AndonBoard) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.andons = append(p.andons, board)
}

func (p *StubOperatorPort) EmitResult(result broker.Result) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.results = append(p.results, result)
}

func (p *StubOperatorPort) PermissionResponses() <-chan broker.PermissionResponse {
	return p.permRespCh
}

// SendIntent simulates an operator sending an intent.
func (p *StubOperatorPort) SendIntent(intent broker.Intent) {
	p.mu.Lock()
	h := p.handler
	p.mu.Unlock()
	if h != nil {
		h(intent)
	}
}

// SendPermissionResponse injects a permission response.
func (p *StubOperatorPort) SendPermissionResponse(resp broker.PermissionResponse) {
	p.permRespCh <- resp
}

// Events returns a copy of all recorded events.
func (p *StubOperatorPort) Events() []broker.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]broker.Event, len(p.events))
	copy(out, p.events)
	return out
}

// Permissions returns a copy of all emitted permission payloads.
func (p *StubOperatorPort) Permissions() []broker.PermissionPayload {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]broker.PermissionPayload, len(p.permissions))
	copy(out, p.permissions)
	return out
}

// Andons returns a copy of all emitted andon boards.
func (p *StubOperatorPort) Andons() []broker.AndonBoard {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]broker.AndonBoard, len(p.andons))
	copy(out, p.andons)
	return out
}

// Results returns a copy of all emitted results.
func (p *StubOperatorPort) Results() []broker.Result {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]broker.Result, len(p.results))
	copy(out, p.results)
	return out
}
