package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	sessionFileExt = ".json"
	sessionDirPerm = 0700
	sessionFilePerm = 0600
)

// Sentinel errors.
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrNameConflict    = errors.New("session name already exists")
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

// Store persists sessions to disk as JSON files.
type Store struct {
	dir string
}

// NewStore creates a session store at the given directory.
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, sessionDirPerm); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

// Save writes a session to disk using atomic write (temp + rename).
// Uses Name as filename if set, otherwise ID.
func (s *Store) Save(sess *Session) error {
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	path := s.sessionPath(sess)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, sessionFilePerm); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("atomic rename: %w", err)
	}
	return nil
}

// Load reads a session from disk by name or ID.
func (s *Store) Load(nameOrID string) (*Session, error) {
	path := filepath.Join(s.dir, nameOrID+sessionFileExt)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
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
		s.Save(&sess) //nolint:errcheck
	}

	return &sess, nil
}

// LoadRaw reads a session WITHOUT sanitizing — for debug inspection.
func (s *Store) LoadRaw(nameOrID string) (*Session, error) {
	path := filepath.Join(s.dir, nameOrID+sessionFileExt)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
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
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read session dir: %w", err)
	}

	var summaries []SessionSummary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), sessionFileExt) {
			continue
		}

		nameOrID := strings.TrimSuffix(e.Name(), sessionFileExt)
		sess, err := s.Load(nameOrID)
		if err != nil {
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

// Delete removes a session file by name or ID.
func (s *Store) Delete(nameOrID string) error {
	path := filepath.Join(s.dir, nameOrID+sessionFileExt)
	err := os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrSessionNotFound, nameOrID)
		}
		return err
	}
	return nil
}

// Archive moves a session to the archive/ subdirectory.
// Archived sessions are excluded from List() but can be retrieved.
func (s *Store) Archive(sess *Session) error {
	archiveDir := filepath.Join(s.dir, "archive")
	if err := os.MkdirAll(archiveDir, sessionDirPerm); err != nil {
		return fmt.Errorf("create archive dir: %w", err)
	}

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	name := sess.Name
	if name == "" {
		name = sess.ID
	}
	path := filepath.Join(archiveDir, name+sessionFileExt)
	if err := os.WriteFile(path, data, sessionFilePerm); err != nil {
		return fmt.Errorf("write archive: %w", err)
	}

	// Remove from active directory.
	activePath := s.sessionPath(sess)
	os.Remove(activePath) //nolint:errcheck

	return nil
}

// ListArchived returns summaries of all archived sessions.
func (s *Store) ListArchived() ([]SessionSummary, error) {
	archiveDir := filepath.Join(s.dir, "archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read archive dir: %w", err)
	}

	var summaries []SessionSummary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), sessionFileExt) {
			continue
		}

		nameOrID := strings.TrimSuffix(e.Name(), sessionFileExt)
		sess, err := s.LoadArchived(nameOrID)
		if err != nil {
			continue
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

// LoadArchived reads a session from the archive directory.
func (s *Store) LoadArchived(nameOrID string) (*Session, error) {
	path := filepath.Join(s.dir, "archive", nameOrID+sessionFileExt)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, nameOrID)
		}
		return nil, fmt.Errorf("read archived session: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("unmarshal archived session %s: %w", nameOrID, err)
	}

	if sess.History == nil {
		sess.History = NewHistory(0)
	}

	return &sess, nil
}

func (s *Store) sessionPath(sess *Session) string {
	name := sess.Name
	if name == "" {
		name = sess.ID
	}
	return filepath.Join(s.dir, name+sessionFileExt)
}
