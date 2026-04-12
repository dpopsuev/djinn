package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var _ KeyStore = (*FileStore)(nil)

// FileStore is a file-per-key KeyStore. Each key = one file in a directory.
// Atomic writes (temp + rename). Thread-safe.
type FileStore struct {
	mu   sync.RWMutex
	dir  string
	ext  string // file extension (default ".json")
	perm os.FileMode
}

// FileStoreOption configures a FileStore.
type FileStoreOption func(*FileStore)

// WithExtension sets the file extension (default ".json").
func WithExtension(ext string) FileStoreOption {
	return func(s *FileStore) { s.ext = ext }
}

// WithPermissions sets the file permissions (default 0644).
func WithPermissions(perm os.FileMode) FileStoreOption {
	return func(s *FileStore) { s.perm = perm }
}

// NewFileStore creates a file-per-key store at the given directory.
func NewFileStore(dir string, opts ...FileStoreOption) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}
	s := &FileStore{
		dir:  dir,
		ext:  ".json",
		perm: 0o644,
	}
	for _, o := range opts {
		o(s)
	}
	return s, nil
}

func (s *FileStore) Put(key string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.keyPath(key)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, s.perm); err != nil {
		return fmt.Errorf("store put %s: %w", key, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) //nolint:errcheck // best-effort cleanup of temp file
		return fmt.Errorf("store put atomic rename %s: %w", key, err)
	}
	return nil
}

func (s *FileStore) Get(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.keyPath(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, key)
		}
		return nil, fmt.Errorf("store get %s: %w", key, err)
	}
	return data, nil
}

func (s *FileStore) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("store list: %w", err)
	}

	keys := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), s.ext) {
			continue
		}
		keys = append(keys, strings.TrimSuffix(e.Name(), s.ext))
	}
	return keys, nil
}

func (s *FileStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.keyPath(key)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrNotFound, key)
		}
		return fmt.Errorf("store delete %s: %w", key, err)
	}
	return nil
}

func (s *FileStore) Close() error { return nil }

func (s *FileStore) keyPath(key string) string {
	return filepath.Join(s.dir, key+s.ext)
}
