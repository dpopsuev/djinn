// stub.go — StubCanon for unit tests. Canned data, no git.
//
// Forge rule: every interface ships with a testkit stub.
//
// GOL-174, TSK-1102
package canon

import "sync"

var _ Canon = (*StubCanon)(nil)

// StubCanon returns canned data. No git, no filesystem.
type StubCanon struct {
	mu      sync.RWMutex
	hashes  map[string]string // file → hash
	dirty   []string
	commits []Commit
	blames  map[string][]BlameLine

	// Call tracking.
	HashCalls  int
	DirtyCalls int
}

// NewStubCanon creates an empty stub.
func NewStubCanon() *StubCanon {
	return &StubCanon{
		hashes: make(map[string]string),
		blames: make(map[string][]BlameLine),
	}
}

// SeedHash sets a canned hash for a file.
func (s *StubCanon) SeedHash(file, hash string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hashes[file] = hash
}

// SeedDirty sets the dirty files list.
func (s *StubCanon) SeedDirty(files []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty = files
}

// SeedCommits sets the commit history.
func (s *StubCanon) SeedCommits(commits []Commit) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commits = commits
}

// SeedBlame sets blame data for a file.
func (s *StubCanon) SeedBlame(file string, lines []BlameLine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blames[file] = lines
}

func (s *StubCanon) ContentHash(file string) (string, error) {
	s.mu.Lock()
	s.HashCalls++
	s.mu.Unlock()

	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.hashes[file]
	if !ok {
		return "", &FileNotFoundError{File: file}
	}
	return h, nil
}

func (s *StubCanon) DirtyFiles() ([]string, error) {
	s.mu.Lock()
	s.DirtyCalls++
	s.mu.Unlock()

	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.dirty))
	copy(out, s.dirty)
	return out, nil
}

func (s *StubCanon) RecentCommits(n int) ([]Commit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if n > len(s.commits) {
		n = len(s.commits)
	}
	out := make([]Commit, n)
	copy(out, s.commits[:n])
	return out, nil
}

func (s *StubCanon) Blame(file string) ([]BlameLine, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lines, ok := s.blames[file]
	if !ok {
		return nil, &FileNotFoundError{File: file}
	}
	out := make([]BlameLine, len(lines))
	copy(out, lines)
	return out, nil
}

// FileNotFoundError is returned when a file is not in the cache.
type FileNotFoundError struct {
	File string
}

func (e *FileNotFoundError) Error() string {
	return "canon: file not found: " + e.File
}
