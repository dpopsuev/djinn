// dump.go — workspace snapshot for post-mortem audit.
//
// DumpWorkspace walks a directory and captures text file contents.
// Used by Arena, Substrate, and any agent lifecycle that produces
// files for later inspection. Observable by default — no opt-in.
package telemetry

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// MaxDumpFileSize is the per-file size limit for workspace dumps.
// Files larger than this are skipped (likely binaries or logs).
const MaxDumpFileSize = 64 * 1024

// DumpWorkspace walks root and returns a map of relative path → content
// for all text files under the size limit. Binaries, executables, and
// dotfiles are excluded.
func DumpWorkspace(root string) map[string]string {
	artifacts := make(map[string]string)

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(root, path)

		// Skip binaries, executables, dotfiles
		if isBinaryName(rel) {
			return nil
		}

		info, err := d.Info()
		if err != nil || info.Size() > MaxDumpFileSize {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		artifacts[rel] = string(content)
		return nil
	})

	return artifacts
}

// isBinaryName returns true for file names that are likely compiled binaries
// or hidden files that shouldn't be captured in a dump.
func isBinaryName(rel string) bool {
	base := filepath.Base(rel)
	if strings.HasPrefix(base, ".") {
		return true
	}
	switch base {
	case "server", "main", "app", "binary":
		return true
	}
	for _, ext := range []string{".exe", ".o", ".a", ".so", ".dylib", ".test"} {
		if strings.HasSuffix(base, ext) {
			return true
		}
	}
	return false
}
