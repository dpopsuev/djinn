// manifest.go — MCP server registry manifest (GOL-148).
// Three-tier merge: remote Git URL > project djinn.yaml > local secrets.
// Static config (persisted) + runtime state (ephemeral).
// Each MCP tool declares requires: [capabilities] for RBAC filtering.
package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dpopsuev/djinn/telemetry"
	"gopkg.in/yaml.v3"
)

// MCPManifest is the static MCP server configuration.
type MCPManifest struct {
	Servers map[string]ServerSpec `yaml:"mcp_servers"`
}

// ServerSpec describes an MCP server.
type ServerSpec struct {
	Command     string              `yaml:"command,omitempty"`      // stdio transport
	Args        []string            `yaml:"args,omitempty"`         // command args
	URL         string              `yaml:"url,omitempty"`          // HTTP transport
	AutoConnect bool                `yaml:"auto_connect,omitempty"` // connect on boot
	Env         map[string]string   `yaml:"env,omitempty"`          // environment variables
	Tools       map[string]ToolSpec `yaml:"tools,omitempty"`        // per-tool RBAC requires
}

// ToolSpec declares capability requirements for an MCP tool.
type ToolSpec struct {
	Requires []string `yaml:"requires,omitempty"` // capability names
}

// MCPSecrets holds machine-local secrets (0600 perms, NOT in Git).
type MCPSecrets struct {
	Servers map[string]SecretSpec `yaml:"servers"`
}

// SecretSpec holds per-server secrets.
type SecretSpec struct {
	Env  map[string]string `yaml:"env,omitempty"`  // secret env vars
	Args []string          `yaml:"args,omitempty"` // secret args
}

// LoadMCPManifest reads an MCP manifest from a YAML file.
func LoadMCPManifest(path string) (*MCPManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load mcp manifest %s: %w", path, err)
	}
	var cfg MCPManifest
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse mcp manifest %s: %w", path, err)
	}
	if cfg.Servers == nil {
		cfg.Servers = make(map[string]ServerSpec)
	}
	return &cfg, nil
}

// LoadMCPSecrets reads secrets from a YAML file.
func LoadMCPSecrets(path string) (*MCPSecrets, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load mcp secrets %s: %w", path, err)
	}
	var secrets MCPSecrets
	if err := yaml.Unmarshal(data, &secrets); err != nil {
		return nil, fmt.Errorf("parse mcp secrets %s: %w", path, err)
	}
	if secrets.Servers == nil {
		secrets.Servers = make(map[string]SecretSpec)
	}
	return &secrets, nil
}

// SaveMCPManifest writes a manifest to YAML (atomic: temp + rename).
func SaveMCPManifest(path string, cfg *MCPManifest) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal mcp manifest: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write mcp manifest temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) //nolint:errcheck // best-effort cleanup of temp file
		return fmt.Errorf("atomic rename mcp manifest: %w", err)
	}
	return nil
}

// MergeManifests merges multiple manifests with priority (last wins).
// remote → project → local. Per-server merge: later overrides earlier.
func MergeManifests(manifests ...*MCPManifest) *MCPManifest {
	result := &MCPManifest{Servers: make(map[string]ServerSpec)}
	for _, m := range manifests {
		if m == nil {
			continue
		}
		for name, spec := range m.Servers {
			result.Servers[name] = spec
		}
	}
	return result
}

// ApplySecrets merges secrets into a manifest's server specs.
// Secret env vars are added to server env (secrets override on conflict).
// Secret args are appended to server args.
func ApplySecrets(cfg *MCPManifest, secrets *MCPSecrets) {
	if secrets == nil {
		return
	}
	for name, sec := range secrets.Servers {
		spec, ok := cfg.Servers[name]
		if !ok {
			continue
		}
		if spec.Env == nil {
			spec.Env = make(map[string]string)
		}
		for k, v := range sec.Env {
			spec.Env[k] = v
		}
		spec.Args = append(spec.Args, sec.Args...)
		cfg.Servers[name] = spec
	}
}

// LogManifest logs the merged manifest (YELLOW).
func LogManifest(ctx context.Context, log *slog.Logger, cfg *MCPManifest) {
	autoConnect := 0
	for _, s := range cfg.Servers {
		if s.AutoConnect {
			autoConnect++
		}
	}
	log.InfoContext(ctx, "mcp manifest loaded",
		slog.Int(telemetry.KeyCount, len(cfg.Servers)),
		slog.Int(telemetry.KeyEntries, autoConnect),
	)
}
