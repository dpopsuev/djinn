// persist.go — shared atomic JSON file persistence for tool stores.
package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// atomicSaveJSON marshals data as indented JSON and writes it atomically
// (write to temp then rename). Used by TaskStore and DiscourseStore.
func atomicSaveJSON(path string, data any, label string) error {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", label, err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) //nolint:errcheck // best-effort cleanup
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
