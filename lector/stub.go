package lector

import "sync"

var _ Lector = (*StubLector)(nil)

// StubLector is an in-memory Lector for testing. Records all observations.
type StubLector struct {
	mu    sync.RWMutex
	files map[string]FileEntry
	syms  map[string][]Symbol // package → symbols

	// Observed tracks calls to Observer methods for test assertions.
	Reads   []string
	Writes  []string
	Deletes []string
}

// NewStubLector creates a StubLector with empty caches.
func NewStubLector() *StubLector {
	return &StubLector{
		files: make(map[string]FileEntry),
		syms:  make(map[string][]Symbol),
	}
}

// --- Index (read) ---

func (s *StubLector) FileInfo(path string) (FileEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.files[path]
	return f, ok
}

func (s *StubLector) Symbols(scope string) []Symbol {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Symbol, len(s.syms[scope]))
	copy(out, s.syms[scope])
	return out
}

func (s *StubLector) Imports(pkg string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.files[pkg]
	if !ok {
		return nil
	}
	out := make([]string, len(f.Imports))
	copy(out, f.Imports)
	return out
}

func (s *StubLector) Dependents(pkg string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var deps []string
	for path, f := range s.files {
		for _, imp := range f.Imports {
			if imp == pkg {
				deps = append(deps, path)
				break
			}
		}
	}
	return deps
}

// --- Observer (write) ---

func (s *StubLector) OnFileRead(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Reads = append(s.Reads, path)
	if _, ok := s.files[path]; !ok {
		s.files[path] = FileEntry{Path: path}
	}
}

func (s *StubLector) OnFileWrite(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Writes = append(s.Writes, path)
	s.files[path] = FileEntry{Path: path}
}

func (s *StubLector) OnFileDelete(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Deletes = append(s.Deletes, path)
	delete(s.files, path)
	delete(s.syms, path)
}

// --- Test helpers ---

// SeedFile adds a pre-indexed file entry for testing.
func (s *StubLector) SeedFile(f FileEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[f.Path] = f
}

// SeedSymbols adds pre-indexed symbols for a package scope.
func (s *StubLector) SeedSymbols(scope string, syms []Symbol) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syms[scope] = syms
}
