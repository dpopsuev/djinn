package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config file locations.
const (
	GlobalConfigDir  = ".djinn"
	GlobalConfigFile = ".djinn/config.yaml"
	ProjectConfig    = "djinn.yaml"
	EnvConfigVar     = "DJINN_CONFIG"
)

// Discover finds config files in priority order (lowest → highest).
// Returns only paths that exist.
func Discover(workdir string) []string {
	var paths []string

	// 1. User global: ~/.djinn/config.yaml
	if home, err := os.UserHomeDir(); err == nil {
		global := filepath.Join(home, GlobalConfigFile)
		if _, err := os.Stat(global); err == nil {
			paths = append(paths, global)
		}
	}

	// 2. Project local: ./djinn.yaml
	local := filepath.Join(workdir, ProjectConfig)
	if _, err := os.Stat(local); err == nil {
		paths = append(paths, local)
	}

	// 3. Environment variable: $DJINN_CONFIG
	if envPath := os.Getenv(EnvConfigVar); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			paths = append(paths, envPath)
		}
	}

	return paths
}

// LoadAll discovers and loads config files in priority order,
// then applies the optional explicit path (highest priority).
func LoadAll(r *Registry, workdir, explicit string) error {
	for _, path := range Discover(workdir) {
		if err := r.LoadFile(path); err != nil {
			return fmt.Errorf("load %s: %w", path, err)
		}
	}
	if explicit != "" {
		if err := r.LoadFile(explicit); err != nil {
			return fmt.Errorf("load %s: %w", explicit, err)
		}
	}
	return nil
}
