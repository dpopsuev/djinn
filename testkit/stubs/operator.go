package stubs

import (
	"sync"

	"github.com/dpopsuev/djinn/ari"
	"github.com/dpopsuev/djinn/broker"
	"github.com/dpopsuev/djinn/orchestrator"
)

// StubOperatorPort implements broker.OperatorPort for testing.
type StubOperatorPort struct {
	mu          sync.Mutex
	intentCh    chan ari.Intent
	permRespCh  chan ari.PermissionResponse
	events      []orchestrator.Event
	permissions []ari.PermissionPayload
	andons      []broker.AndonBoard
	results     []ari.Result
	handler     func(ari.Intent)
}

// NewStubOperatorPort creates a new stub operator port.
func NewStubOperatorPort() *StubOperatorPort {
	return &StubOperatorPort{
		intentCh:   make(chan ari.Intent, 10),
		permRespCh: make(chan ari.PermissionResponse, 10),
	}
}

func (p *StubOperatorPort) OnIntent(handler func(ari.Intent)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handler = handler
}

func (p *StubOperatorPort) EmitProgress(event orchestrator.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
}

func (p *StubOperatorPort) EmitPermission(payload ari.PermissionPayload) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.permissions = append(p.permissions, payload)
}

func (p *StubOperatorPort) EmitAndon(board broker.AndonBoard) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.andons = append(p.andons, board)
}

func (p *StubOperatorPort) EmitResult(result ari.Result) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.results = append(p.results, result)
}

func (p *StubOperatorPort) PermissionResponses() <-chan ari.PermissionResponse {
	return p.permRespCh
}

// SendIntent simulates an operator sending an intent.
func (p *StubOperatorPort) SendIntent(intent ari.Intent) {
	p.mu.Lock()
	h := p.handler
	p.mu.Unlock()
	if h != nil {
		h(intent)
	}
}

// SendPermissionResponse injects a permission response.
func (p *StubOperatorPort) SendPermissionResponse(resp ari.PermissionResponse) {
	p.permRespCh <- resp
}

// Events returns a copy of all recorded events.
func (p *StubOperatorPort) Events() []orchestrator.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]orchestrator.Event, len(p.events))
	copy(out, p.events)
	return out
}

// Permissions returns a copy of all emitted permission payloads.
func (p *StubOperatorPort) Permissions() []ari.PermissionPayload {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]ari.PermissionPayload, len(p.permissions))
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
func (p *StubOperatorPort) Results() []ari.Result {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]ari.Result, len(p.results))
	copy(out, p.results)
	return out
}
