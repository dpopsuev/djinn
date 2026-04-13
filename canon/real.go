// real.go — RealCanon: production VCS cache with mtime verification.
//
// Wraps exec git for VCS queries. Caches content hashes with mtime
// verification — stat() is free (one syscall), rehash only when
// file mtime changes. Writes to L2 cache for cross-agent sharing.
//
// GOL-174, TSK-1104, TSK-1105
package canon

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	djinncache "github.com/dpopsuev/djinn/cache"
	"github.com/dpopsuev/djinn/telemetry"
)

var _ Canon = (*RealCanon)(nil)

// hashEntry caches a content hash with its mtime for staleness detection.
type hashEntry struct {
	hash  string
	mtime time.Time
}

// RealCanon is the production Canon backed by exec git + L2 cache.
type RealCanon struct {
	dir string
	l2  djinncache.Cache
	log *slog.Logger

	mu     sync.RWMutex
	hashes map[string]hashEntry // file → cached hash + mtime
}

// NewRealCanon creates a production Canon for the given workspace directory.
func NewRealCanon(dir string, l2 djinncache.Cache, log *slog.Logger) *RealCanon {
	if log == nil {
		log = telemetry.Nop()
	}
	return &RealCanon{
		dir:    dir,
		l2:     l2,
		log:    log,
		hashes: make(map[string]hashEntry),
	}
}

// ContentHash returns SHA256 of file contents. Cached with mtime verification.
func (c *RealCanon) ContentHash(file string) (string, error) {
	path := filepath.Join(c.dir, file)

	info, err := os.Stat(path)
	if err != nil {
		c.log.WarnContext(context.Background(), "canon: file not found",
			slog.String(telemetry.KeyPath, file),
			slog.String(telemetry.KeyError, err.Error()),
		)
		return "", &FileNotFoundError{File: file}
	}

	// Check in-memory cache: mtime match = trust hash.
	c.mu.RLock()
	entry, ok := c.hashes[file]
	c.mu.RUnlock()

	if ok && entry.mtime.Equal(info.ModTime()) {
		c.log.DebugContext(context.Background(), "canon: cache hit",
			slog.String(telemetry.KeyPath, file),
		)
		return entry.hash, nil
	}

	c.log.DebugContext(context.Background(), "canon: cache miss, rehashing",
		slog.String(telemetry.KeyPath, file),
	)

	// Cache miss or stale — rehash from disk.
	data, err := os.ReadFile(path)
	if err != nil {
		c.log.WarnContext(context.Background(), "canon: read failed",
			slog.String(telemetry.KeyPath, file),
			slog.String(telemetry.KeyError, err.Error()),
		)
		return "", fmt.Errorf("canon: read %s: %w", file, err)
	}

	h := fmt.Sprintf("%x", sha256.Sum256(data))

	// Update in-memory cache.
	c.mu.Lock()
	c.hashes[file] = hashEntry{hash: h, mtime: info.ModTime()}
	c.mu.Unlock()

	// Write to L2 for cross-agent sharing.
	if c.l2 != nil {
		c.l2.Put("canon", "hash:"+file, []byte(h))
	}

	return h, nil
}

// DirtyFiles returns files with uncommitted changes.
func (c *RealCanon) DirtyFiles() ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = c.dir
	out, err := cmd.Output()
	if err != nil {
		c.log.WarnContext(context.Background(), "canon: git status failed",
			slog.String(telemetry.KeyError, err.Error()),
		)
		return nil, fmt.Errorf("canon: git status: %w", err)
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		if len(line) > 3 { //nolint:mnd // porcelain format: "XY filename"
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}
	return files, nil
}

// RecentCommits returns the last n commits.
func (c *RealCanon) RecentCommits(n int) ([]Commit, error) {
	cmd := exec.Command("git", "log", fmt.Sprintf("-%d", n), "--format=%H|%an|%aI|%s") //nolint:gosec // n is int, not user input
	cmd.Dir = c.dir
	out, err := cmd.Output()
	if err != nil {
		c.log.WarnContext(context.Background(), "canon: git log failed",
			slog.String(telemetry.KeyError, err.Error()),
		)
		return nil, fmt.Errorf("canon: git log: %w", err)
	}

	commits := make([]Commit, 0, n)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4) //nolint:mnd // format fields
		if len(parts) < 4 {                   //nolint:mnd // format fields
			continue
		}
		date, _ := time.Parse(time.RFC3339, parts[2])
		commits = append(commits, Commit{
			Hash:    parts[0],
			Author:  parts[1],
			Date:    date,
			Message: parts[3],
		})
	}
	return commits, nil
}

// Blame returns per-line commit attribution for a file.
func (c *RealCanon) Blame(file string) ([]BlameLine, error) {
	path := filepath.Join(c.dir, file)
	if _, err := os.Stat(path); err != nil {
		c.log.WarnContext(context.Background(), "canon: blame file not found",
			slog.String(telemetry.KeyPath, file),
		)
		return nil, &FileNotFoundError{File: file}
	}

	cmd := exec.Command("git", "blame", "--porcelain", file)
	cmd.Dir = c.dir
	out, err := cmd.Output()
	if err != nil {
		c.log.WarnContext(context.Background(), "canon: git blame failed",
			slog.String(telemetry.KeyPath, file),
			slog.String(telemetry.KeyError, err.Error()),
		)
		return nil, fmt.Errorf("canon: git blame %s: %w", file, err)
	}

	var lines []BlameLine
	lineNum := 0
	var currentHash, currentAuthor string
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) >= 40 && line[0] != '\t' { //nolint:mnd // sha length
			parts := strings.Fields(line)
			if len(parts) >= 1 && len(parts[0]) == 40 { //nolint:mnd // sha length
				currentHash = parts[0]
			}
		}
		if after, found := strings.CutPrefix(line, "author "); found {
			currentAuthor = after
		}
		if strings.HasPrefix(line, "\t") {
			lineNum++
			lines = append(lines, BlameLine{
				Line:   lineNum,
				Hash:   currentHash,
				Author: currentAuthor,
			})
		}
	}
	return lines, nil
}
