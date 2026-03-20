package ari

import (
	"sync"

	ariTypes "github.com/dpopsuev/djinn/ari"
	"github.com/dpopsuev/djinn/broker"
	"github.com/dpopsuev/djinn/orchestrator"
)

// MockARIServer implements broker.OperatorPort as an in-process mock.
type MockARIServer struct {
	mu          sync.Mutex
	handler     func(ariTypes.Intent)
	permRespCh  chan ariTypes.PermissionResponse
	events      []orchestrator.Event
	permissions []ariTypes.PermissionPayload
	andons      []broker.AndonBoard
	results     []ariTypes.Result
}

// NewMockARIServer creates a new mock ARI server.
func NewMockARIServer() *MockARIServer {
	return &MockARIServer{
		permRespCh: make(chan ariTypes.PermissionResponse, 10),
	}
}

func (s *MockARIServer) OnIntent(handler func(ariTypes.Intent)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = handler
}

func (s *MockARIServer) EmitProgress(event orchestrator.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}

func (s *MockARIServer) EmitPermission(payload ariTypes.PermissionPayload) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.permissions = append(s.permissions, payload)
}

func (s *MockARIServer) EmitAndon(board broker.AndonBoard) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.andons = append(s.andons, board)
}

func (s *MockARIServer) EmitResult(result ariTypes.Result) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results = append(s.results, result)
}

func (s *MockARIServer) PermissionResponses() <-chan ariTypes.PermissionResponse {
	return s.permRespCh
}

// InjectIntent simulates receiving an intent from the operator.
func (s *MockARIServer) InjectIntent(intent ariTypes.Intent) {
	s.mu.Lock()
	h := s.handler
	s.mu.Unlock()
	if h != nil {
		h(intent)
	}
}

// RecordedEvents returns a copy of all progress events.
func (s *MockARIServer) RecordedEvents() []orchestrator.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]orchestrator.Event, len(s.events))
	copy(out, s.events)
	return out
}

// RecordedResults returns a copy of all results.
func (s *MockARIServer) RecordedResults() []ariTypes.Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ariTypes.Result, len(s.results))
	copy(out, s.results)
	return out
}

// RecordedAndons returns a copy of all andon boards.
func (s *MockARIServer) RecordedAndons() []broker.AndonBoard {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]broker.AndonBoard, len(s.andons))
	copy(out, s.andons)
	return out
}
