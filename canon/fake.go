// fake.go — FakeCanon for integration tests. Real git on t.TempDir().
//
// Creates a real git repo, allows seeding files and commits,
// then serves via the same interface as RealCanon will.
//
// GOL-174, TSK-1102
package canon

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var _ Canon = (*FakeCanon)(nil)

// FakeCanon operates on a real git repo in a temp directory.
type FakeCanon struct {
	dir string
}

// NewFakeCanon creates a git repo in the given directory.
// The directory must exist. Runs git init + initial commit.
func NewFakeCanon(dir string) (*FakeCanon, error) {
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("git init: %s: %w", out, err)
		}
	}

	// Create initial commit so HEAD exists.
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# test repo\n"), 0o600); err != nil { //nolint:mnd // file perms
		return nil, err
	}
	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "initial"},
	} {
		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("%s: %s: %w", args[1], out, err)
		}
	}

	return &FakeCanon{dir: dir}, nil
}

// WriteFile creates or overwrites a file in the repo (uncommitted).
func (f *FakeCanon) WriteFile(rel, content string) error {
	path := filepath.Join(f.dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:mnd // dir perms
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600) //nolint:mnd // file perms
}

// CommitAll stages and commits everything.
func (f *FakeCanon) CommitAll(msg string) error {
	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", msg},
	} {
		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = f.dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %s: %w", args[1], out, err)
		}
	}
	return nil
}

// Dir returns the repo root.
func (f *FakeCanon) Dir() string { return f.dir }

// --- Canon interface ---

func (f *FakeCanon) ContentHash(file string) (string, error) {
	path := filepath.Join(f.dir, file)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", &FileNotFoundError{File: file}
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h), nil
}

func (f *FakeCanon) DirtyFiles() ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = f.dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		// Porcelain format: "XY filename"
		if len(line) > 3 { //nolint:mnd // porcelain format
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}
	return files, nil
}

func (f *FakeCanon) RecentCommits(n int) ([]Commit, error) {
	cmd := exec.Command("git", "log", fmt.Sprintf("-%d", n), "--format=%H|%an|%aI|%s") //nolint:gosec // test helper
	cmd.Dir = f.dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
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

func (f *FakeCanon) Blame(file string) ([]BlameLine, error) {
	path := filepath.Join(f.dir, file)
	if _, err := os.Stat(path); err != nil {
		return nil, &FileNotFoundError{File: file}
	}

	cmd := exec.Command("git", "blame", "--porcelain", file)
	cmd.Dir = f.dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git blame: %w", err)
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
		if strings.HasPrefix(line, "author ") {
			currentAuthor = strings.TrimPrefix(line, "author ")
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
