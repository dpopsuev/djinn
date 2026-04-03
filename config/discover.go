package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
//
// Walk order:
//  1. User global: ~/.djinn/config.yaml (lowest priority)
//  2. Parent-directory walk: from root down to workdir, each djinn.yaml found
//     (shallowest first, so deepest/most-specific overrides shallowest)
//  3. Environment variable: $DJINN_CONFIG (highest priority)
func Discover(workdir string) []string {
	var paths []string

	// 1. User global: ~/.djinn/config.yaml
	if home, err := os.UserHomeDir(); err == nil {
		global := filepath.Join(home, GlobalConfigFile)
		if _, err := os.Stat(global); err == nil {
			paths = append(paths, global)
		}
	}

	// 2. Walk from workdir upward to root, collecting all djinn.yaml files.
	// We walk upward and then reverse so shallowest comes first (lower priority)
	// and deepest (most specific) comes last (higher priority).
	var walkPaths []string
	for dir := filepath.Clean(workdir); dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, ProjectConfig)
		if _, err := os.Stat(candidate); err == nil {
			walkPaths = append(walkPaths, candidate)
		}
	}
	// Also check root "/" itself.
	rootCandidate := filepath.Join(string(filepath.Separator), ProjectConfig)
	if _, err := os.Stat(rootCandidate); err == nil {
		walkPaths = append(walkPaths, rootCandidate)
	}
	// walkPaths is deepest-first (workdir → root). Reverse to shallowest-first.
	slices.Reverse(walkPaths)
	paths = append(paths, walkPaths...)

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
