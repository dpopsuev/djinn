// canon.go — Canon interface and core types.
//
// GOL-174, TSK-1102
package canon

import "time"

// Canon provides cached VCS state. Implementations must be thread-safe.
type Canon interface {
	// ContentHash returns the SHA256 hash of a file's contents.
	// Cached with mtime verification — stat() is free, rehash only on change.
	ContentHash(file string) (string, error)

	// DirtyFiles returns files with uncommitted changes (modified, untracked, staged).
	DirtyFiles() ([]string, error)

	// RecentCommits returns the last n commits.
	RecentCommits(n int) ([]Commit, error)

	// Blame returns per-line commit attribution for a file.
	Blame(file string) ([]BlameLine, error)
}

// Commit represents a git commit.
type Commit struct {
	Hash    string    `json:"hash"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
	Message string    `json:"message"`
}

// BlameLine maps a source line to its originating commit.
type BlameLine struct {
	Line   int    `json:"line"`
	Hash   string `json:"hash"`
	Author string `json:"author"`
}
