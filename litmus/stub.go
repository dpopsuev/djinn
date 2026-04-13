// stub.go — StubLitmus for unit tests. Canned data, no exec.
//
// GOL-175, TSK-1118
package litmus

import "sync"

var _ Litmus = (*StubLitmus)(nil)

// StubLitmus returns canned data. No exec, no filesystem.
type StubLitmus struct {
	mu     sync.RWMutex
	tests  map[string]TestResultEntry
	builds map[string]BuildResultEntry

	// Call tracking.
	RecordCalls     int
	InvalidateCalls int
}

// NewStubLitmus creates an empty stub.
func NewStubLitmus() *StubLitmus {
	return &StubLitmus{
		tests:  make(map[string]TestResultEntry),
		builds: make(map[string]BuildResultEntry),
	}
}

// SeedTest sets a canned test result for a package.
func (s *StubLitmus) SeedTest(pkg string, result TestResultEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tests[pkg] = result
}

// SeedBuild sets a canned build result for a package.
func (s *StubLitmus) SeedBuild(pkg string, result BuildResultEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.builds[pkg] = result
}

func (s *StubLitmus) TestResult(pkg string) (TestResultEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.tests[pkg]
	return r, ok
}

func (s *StubLitmus) BuildResult(pkg string) (BuildResultEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.builds[pkg]
	return r, ok
}

func (s *StubLitmus) RecordTest(pkg string, result TestResultEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tests[pkg] = result
	s.RecordCalls++
}

func (s *StubLitmus) RecordBuild(pkg string, result BuildResultEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.builds[pkg] = result
	s.RecordCalls++
}

func (s *StubLitmus) Invalidate(pkg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tests, pkg)
	delete(s.builds, pkg)
	s.InvalidateCalls++
}

func (s *StubLitmus) InvalidateAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tests = make(map[string]TestResultEntry)
	s.builds = make(map[string]BuildResultEntry)
	s.InvalidateCalls++
}
