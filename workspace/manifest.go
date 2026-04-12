// manifest.go — two-tier workspace manifest YAML (GOL-145).
// Ecosystem (virtual grouping) → project (real host path).
// Read on start, persist on mount/unmount.
// Like VSCode .code-workspace files.
package workspace

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/dpopsuev/djinn/telemetry"
)

// ManifestConfig represents the workspace manifest YAML.
// Two tiers: ecosystem scope → project mounts. No recursive nesting.
type ManifestConfig struct {
	Scope  string               `yaml:"scope"`  // ecosystem path (e.g. "/djinn")
	Mounts map[string]MountSpec `yaml:"mounts"` // project name → spec
}

// MountSpec describes a single project mount.
type MountSpec struct {
	Host     string `yaml:"host"`               // host filesystem path
	ReadOnly bool   `yaml:"readonly,omitempty"` // true for reference mounts
	Scope    string `yaml:"scope,omitempty"`    // scope type (operations, global, etc.)
}

// LoadManifest reads a workspace manifest from a YAML file.
func LoadManifest(path string) (*ManifestConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load manifest %s: %w", path, err)
	}
	var cfg ManifestConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", path, err)
	}
	if cfg.Mounts == nil {
		cfg.Mounts = make(map[string]MountSpec)
	}
	return &cfg, nil
}

// SaveManifest writes a workspace manifest to a YAML file (atomic: temp + rename).
func SaveManifest(path string, cfg *ManifestConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write manifest temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) //nolint:errcheck // best-effort cleanup of temp file
		return fmt.Errorf("atomic rename manifest: %w", err)
	}
	return nil
}

// PopulateMountTable loads a manifest into a MountTable.
// Each mount becomes: /{scope}/{projectName} → host path.
func PopulateMountTable(cfg *ManifestConfig, table *MountTable, log *slog.Logger) error {
	for name, spec := range cfg.Mounts {
		virtualPath := filepath.Join(cfg.Scope, name)

		scopeType := ScopeType(spec.Scope)
		if scopeType == "" {
			scopeType = ScopeOperations
		}

		if err := table.Mount(virtualPath, spec.Host, spec.ReadOnly, scopeType); err != nil {
			if log != nil {
				log.WarnContext(context.Background(), "manifest mount failed",
					slog.String(telemetry.KeyPath, virtualPath),
					slog.String(telemetry.KeySource, spec.Host),
					slog.String(telemetry.KeyError, err.Error()),
				)
			}
			return fmt.Errorf("mount %s: %w", virtualPath, err)
		}
	}

	if log != nil {
		log.InfoContext(context.Background(), "manifest loaded",
			slog.String(telemetry.KeyScope, cfg.Scope),
			slog.Int(telemetry.KeyCount, len(cfg.Mounts)),
		)
	}

	return nil
}
