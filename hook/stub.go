// stub.go — StubDispatcher for testing.
//
// Records all dispatches for test assertions.
// Forge rule: every interface ships with a testkit stub.
//
// GOL-161, TSK-1068
package hook

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/dpopsuev/battery/middleware"
)

var (
	_ middleware.Gate     = (*StubDispatcher)(nil)
	_ middleware.Recorder = (*StubDispatcher)(nil)
)

// StubDispatcher records all hook dispatches for test assertions.
type StubDispatcher struct {
	mu       sync.Mutex
	checks   []StubCheck
	records  []StubRecord
	denyTool map[string]string // tool → reason
}

// StubCheck records a gate check call.
type StubCheck struct {
	Tool  string
	Input json.RawMessage
}

// StubRecord records a post-tool recording call.
type StubRecord struct {
	Tool    string
	Output  string
	Elapsed time.Duration
}

// NewStubDispatcher creates an empty stub.
func NewStubDispatcher() *StubDispatcher {
	return &StubDispatcher{denyTool: make(map[string]string)}
}

// SetDeny configures the stub to deny a specific tool.
func (s *StubDispatcher) SetDeny(tool, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.denyTool[tool] = reason
}

// Check implements middleware.Gate.
func (s *StubDispatcher) Check(_ context.Context, tool string, input json.RawMessage) (middleware.Verdict, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks = append(s.checks, StubCheck{Tool: tool, Input: input})
	if reason, ok := s.denyTool[tool]; ok {
		return middleware.Verdict{Allowed: false, Reason: reason}, nil
	}
	return middleware.Verdict{Allowed: true}, nil
}

// Record implements middleware.Recorder.
func (s *StubDispatcher) Record(_ context.Context, tool string, _ json.RawMessage, output string, _ error, elapsed time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, StubRecord{Tool: tool, Output: output, Elapsed: elapsed})
}

// Checks returns recorded gate checks.
func (s *StubDispatcher) Checks() []StubCheck {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]StubCheck, len(s.checks))
	copy(out, s.checks)
	return out
}

// Records returns recorded post-tool observations.
func (s *StubDispatcher) Records() []StubRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]StubRecord, len(s.records))
	copy(out, s.records)
	return out
}
