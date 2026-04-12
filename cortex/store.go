package cortex

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	djinnstore "github.com/dpopsuev/djinn/store"
)

const (
	sessionFileExt  = ".json"
	sessionDirPerm  = 0o700
	sessionFilePerm = 0o600
)

// Sentinel errors.
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrNameConflict    = errors.New("session name already exists")
	ErrArchiveNoDir    = errors.New("archive requires file-backed store with directory")
)

// SessionSummary is a lightweight view of a session for listing.
type SessionSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Driver    string    `json:"driver,omitempty"`
	Model     string    `json:"model"`
	WorkDir   string    `json:"work_dir"`
	Turns     int       `json:"turns"`
	Tokens    int       `json:"tokens"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store persists sessions via a KeyStore backend.
// Default backend is FileStore (file-per-key JSON). Can be swapped for
// MemStore (testkit) or ReliquaryStore (future).
type Store struct {
	dir     string
	backend djinnstore.KeyStore
}

// NewStore creates a session store at the given directory using FileStore.
func NewStore(dir string) (*Store, error) {
	fs, err := djinnstore.NewFileStore(dir, djinnstore.WithPermissions(sessionFilePerm))
	if err != nil {
		return nil, fmt.Errorf("create session store: %w", err)
	}
	return &Store{dir: dir, backend: fs}, nil
}

// NewStoreWithBackend creates a session store with a custom KeyStore backend.
// Use for testing (MemStore) or Reliquary integration.
func NewStoreWithBackend(backend djinnstore.KeyStore) *Store {
	return &Store{backend: backend}
}

// Save writes a session via the KeyStore backend.
// Uses Name as key if set, otherwise ID.
func (s *Store) Save(sess *Session) error {
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	return s.backend.Put(s.sessionKey(sess), data)
}

// Load reads a session by name or ID via the KeyStore backend.
func (s *Store) Load(nameOrID string) (*Session, error) {
	data, err := s.backend.Get(nameOrID)
	if err != nil {
		if errors.Is(err, djinnstore.ErrNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, nameOrID)
		}
		return nil, fmt.Errorf("read session: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("unmarshal session %s: %w", nameOrID, err)
	}

	// Ensure History is initialized (may be nil from old files)
	if sess.History == nil {
		sess.History = NewHistory(0)
	}

	// Sanitize: repair corrupt entries, auto-compact oversized sessions (DJN-BUG-14).
	if repaired := Sanitize(&sess); repaired {
		// Persist sanitized version immediately so fixes survive crashes (DJN-BUG-18).
		s.Save(&sess) //nolint:errcheck // best-effort persist
	}

	return &sess, nil
}

// LoadRaw reads a session WITHOUT sanitizing — for debug inspection.
func (s *Store) LoadRaw(nameOrID string) (*Session, error) {
	data, err := s.backend.Get(nameOrID)
	if err != nil {
		if errors.Is(err, djinnstore.ErrNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, nameOrID)
		}
		return nil, fmt.Errorf("read session: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("unmarshal session %s: %w", nameOrID, err)
	}

	if sess.History == nil {
		sess.History = NewHistory(0)
	}

	return &sess, nil
}

// List returns summaries of all sessions, sorted by most recently updated.
func (s *Store) List() ([]SessionSummary, error) {
	return s.listFrom(s.backend, s.Load)
}

// listFrom reads sessions from a KeyStore and returns sorted summaries.
func (s *Store) listFrom(ks djinnstore.KeyStore, loader func(string) (*Session, error)) ([]SessionSummary, error) {
	keys, err := ks.List()
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	summaries := make([]SessionSummary, 0, len(keys))
	for _, key := range keys {
		sess, loadErr := loader(key)
		if loadErr != nil {
			continue // skip corrupt files
		}

		summaries = append(summaries, SessionSummary{
			ID:        sess.ID,
			Name:      sess.Name,
			Driver:    sess.Driver,
			Model:     sess.Model,
			WorkDir:   sess.WorkDir,
			Turns:     sess.History.Len(),
			Tokens:    sess.TotalTokens(),
			CreatedAt: sess.CreatedAt,
			UpdatedAt: sess.UpdatedAt,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].UpdatedAt.After(summaries[j].UpdatedAt)
	})

	return summaries, nil
}

// Delete removes a session by name or ID.
func (s *Store) Delete(nameOrID string) error {
	err := s.backend.Delete(nameOrID)
	if err != nil {
		if errors.Is(err, djinnstore.ErrNotFound) {
			return fmt.Errorf("%w: %s", ErrSessionNotFound, nameOrID)
		}
		return err
	}
	return nil
}

// Archive moves a session to the archive/ subdirectory.
// Archived sessions are excluded from List() but can be retrieved.
func (s *Store) Archive(sess *Session) error {
	if s.dir == "" {
		return ErrArchiveNoDir
	}
	archiveDir := filepath.Join(s.dir, "archive")
	archiveStore, err := djinnstore.NewFileStore(archiveDir, djinnstore.WithPermissions(sessionFilePerm))
	if err != nil {
		return fmt.Errorf("create archive store: %w", err)
	}

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	key := s.sessionKey(sess)
	if err := archiveStore.Put(key, data); err != nil {
		return fmt.Errorf("write archive: %w", err)
	}

	// Remove from active store.
	s.backend.Delete(key) //nolint:errcheck // best-effort cleanup

	return nil
}

// ListArchived returns summaries of all archived sessions.
func (s *Store) ListArchived() ([]SessionSummary, error) {
	if s.dir == "" {
		return nil, nil
	}
	archiveDir := filepath.Join(s.dir, "archive")
	archiveStore, err := djinnstore.NewFileStore(archiveDir, djinnstore.WithPermissions(sessionFilePerm))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return s.listFrom(archiveStore, func(key string) (*Session, error) {
		return s.loadFrom(archiveStore, key)
	})
}

// LoadArchived reads a session from the archive directory.
func (s *Store) LoadArchived(nameOrID string) (*Session, error) {
	if s.dir == "" {
		return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, nameOrID)
	}
	archiveDir := filepath.Join(s.dir, "archive")
	archiveStore, err := djinnstore.NewFileStore(archiveDir, djinnstore.WithPermissions(sessionFilePerm))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, nameOrID)
	}
	return s.loadFrom(archiveStore, nameOrID)
}

// loadFrom reads and unmarshals a session from any KeyStore.
func (s *Store) loadFrom(ks djinnstore.KeyStore, nameOrID string) (*Session, error) {
	data, err := ks.Get(nameOrID)
	if err != nil {
		if errors.Is(err, djinnstore.ErrNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, nameOrID)
		}
		return nil, fmt.Errorf("read session: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("unmarshal session %s: %w", nameOrID, err)
	}
	if sess.History == nil {
		sess.History = NewHistory(0)
	}
	return &sess, nil
}

// sessionKey returns the key for a session (Name if set, otherwise ID).
func (s *Store) sessionKey(sess *Session) string {
	if sess.Name != "" {
		return sess.Name
	}
	return sess.ID
}
