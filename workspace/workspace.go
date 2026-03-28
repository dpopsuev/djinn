// Package workspace manages named workspace manifests.
// A workspace is FS-agnostic — it's a named entity with repos,
// config, and MCP definitions stored as YAML.
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dpopsuev/djinn/policy"
	"gopkg.in/yaml.v3"
)

// Directory and file conventions.
const (
	WorkspacesDir   = "workspaces"
	ManifestExt     = ".yaml"
	DefaultDirPerm  = 0o700
	DefaultFilePerm = 0o644
)

// Sentinel errors.
var (
	ErrNotFound = errors.New("workspace not found")
	ErrNoName   = errors.New("workspace name is required")
)

// Repo roles.
const (
	RolePrimary    = "primary"
	RoleDependency = "dependency"
	RoleReference  = "reference"
)

// SandboxConfig declares the desired isolation level and backend.
type SandboxConfig struct {
	Backend string `yaml:"backend,omitempty"` // misbah, bubblewrap, podman, nsjail, firecracker
	Level   string `yaml:"level,omitempty"`   // none, namespace, container, kata
}

// Workspace is a named project configuration.
type Workspace struct {
	Name    string            `yaml:"name"`
	Sandbox SandboxConfig     `yaml:"sandbox,omitempty"`
	Repos   []Repo            `yaml:"repos"`
	Driver  string            `yaml:"driver,omitempty"`
	Model   string            `yaml:"model,omitempty"`
	Mode    string            `yaml:"mode,omitempty"`
	MCP     map[string]MCPDef `yaml:"mcp,omitempty"`
}

// Repo is a directory with a role in the workspace.
type Repo struct {
	Path string `yaml:"path"`
	Role string `yaml:"role"` // primary, dependency, reference
}

// MCPDef defines an MCP server connection.
type MCPDef struct {
	URL     string            `yaml:"url,omitempty"`
	Command string            `yaml:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

// Summary is a lightweight view for listing.
type Summary struct {
	Name     string
	Repos    int
	Primary  string
	Modified string
}

// ToCapabilityToken generates an immutable capability token from the workspace.
// WritablePaths = repo paths. DeniedPaths = Djinn config directories.
func (w *Workspace) ToCapabilityToken() policy.CapabilityToken {
	home, _ := os.UserHomeDir()
	return policy.CapabilityToken{
		WritablePaths: w.Paths(),
		DeniedPaths: []string{
			filepath.Join(home, ".config", "djinn"),
			filepath.Join(home, ".djinn"),
		},
		Tier: "mod", // default
	}
}

// Paths returns all repo paths.
func (w *Workspace) Paths() []string {
	paths := make([]string, len(w.Repos))
	for i, r := range w.Repos {
		paths[i] = r.Path
	}
	return paths
}

// PrimaryPath returns the primary repo path, or the first repo if none is primary.
func (w *Workspace) PrimaryPath() string {
	for _, r := range w.Repos {
		if r.Role == RolePrimary {
			return r.Path
		}
	}
	if len(w.Repos) > 0 {
		return w.Repos[0].Path
	}
	return ""
}

// Load loads a workspace by name (from ~/.config/djinn/workspaces/)
// or by file path (if it contains a path separator or ends in .yaml).
func Load(nameOrPath string) (*Workspace, error) {
	if nameOrPath == "" {
		return nil, ErrNoName
	}

	// Direct file path
	if strings.Contains(nameOrPath, string(filepath.Separator)) || strings.HasSuffix(nameOrPath, ManifestExt) {
		return loadFile(nameOrPath)
	}

	// Named workspace from config dir
	dir, err := workspacesDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, nameOrPath+ManifestExt)
	return loadFile(path)
}

func loadFile(path string) (*Workspace, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
		}
		return nil, err
	}
	var ws Workspace
	if err := yaml.Unmarshal(data, &ws); err != nil {
		return nil, fmt.Errorf("parse workspace: %w", err)
	}
	return &ws, nil
}

// Save persists a workspace manifest to ~/.config/djinn/workspaces/<name>.yaml.
func Save(ws *Workspace) error {
	if ws.Name == "" {
		return ErrNoName
	}
	dir, err := workspacesDir()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(ws)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, ws.Name+ManifestExt)
	return os.WriteFile(path, data, DefaultFilePerm)
}

// List returns summaries of all saved workspaces.
func List() ([]Summary, error) {
	dir, err := workspacesDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	summaries := make([]Summary, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ManifestExt) {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ManifestExt)
		ws, err := Load(name)
		if err != nil {
			continue
		}
		info, _ := e.Info()
		mod := ""
		if info != nil {
			mod = info.ModTime().Format("2006-01-02 15:04")
		}
		summaries = append(summaries, Summary{
			Name:     ws.Name,
			Repos:    len(ws.Repos),
			Primary:  ws.PrimaryPath(),
			Modified: mod,
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Modified > summaries[j].Modified
	})
	return summaries, nil
}

// Ephemeral creates an unnamed workspace from a single directory.
// Used for backward compatibility when no -w flag is given.
func Ephemeral(cwd string) *Workspace {
	return &Workspace{
		Repos: []Repo{{Path: cwd, Role: RolePrimary}},
	}
}

func workspacesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "djinn", WorkspacesDir)
	if err := os.MkdirAll(dir, DefaultDirPerm); err != nil {
		return "", err
	}
	return dir, nil
}
