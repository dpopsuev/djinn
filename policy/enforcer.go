package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sentinel errors.
var (
	ErrDeniedPath   = errors.New("access denied: protected path")
	ErrDeniedTool   = errors.New("access denied: tool not allowed")
	ErrDeniedBash   = errors.New("access denied: command references protected path")
)

// DefaultEnforcer checks file paths and tool permissions.
type DefaultEnforcer struct{}

// NewDefaultEnforcer creates the standard enforcer.
func NewDefaultEnforcer() *DefaultEnforcer {
	return &DefaultEnforcer{}
}

func (e *DefaultEnforcer) Check(token CapabilityToken, tool string, input json.RawMessage) error {
	// Check tool whitelist
	if len(token.AllowedTools) > 0 {
		allowed := false
		for _, t := range token.AllowedTools {
			if t == tool {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("%w: %s", ErrDeniedTool, tool)
		}
	}

	// Check file-path tools
	switch tool {
	case "Write", "Edit", "Read":
		return e.checkFilePath(token, tool, input)
	case "Bash":
		return e.checkBash(token, input)
	}

	return nil
}

func (e *DefaultEnforcer) checkFilePath(token CapabilityToken, tool string, input json.RawMessage) error {
	var params struct {
		Path    string `json:"path"`
		FilePath string `json:"file_path"`
	}
	_ = json.Unmarshal(input, &params) //nolint:errcheck // best-effort parse for path extraction

	path := params.Path
	if path == "" {
		path = params.FilePath
	}
	if path == "" {
		return nil
	}

	resolved, err := resolvePath(path)
	if err != nil {
		// If we can't resolve, deny — fail closed
		return fmt.Errorf("%w: cannot resolve %q", ErrDeniedPath, path)
	}

	// Check denied paths
	for _, denied := range token.DeniedPaths {
		deniedResolved, _ := resolvePath(denied)
		if deniedResolved == "" {
			deniedResolved = denied
		}
		if strings.HasPrefix(resolved, deniedResolved) {
			return fmt.Errorf("%w: %s is protected", ErrDeniedPath, path)
		}
	}

	// For Write/Edit, check writable paths (if specified)
	if (tool == "Write" || tool == "Edit") && len(token.WritablePaths) > 0 {
		writable := false
		for _, wp := range token.WritablePaths {
			wpResolved, _ := resolvePath(wp)
			if wpResolved == "" {
				wpResolved = wp
			}
			if strings.HasPrefix(resolved, wpResolved) {
				writable = true
				break
			}
		}
		if !writable {
			return fmt.Errorf("%w: %s is outside workspace", ErrDeniedPath, path)
		}
	}

	return nil
}

func (e *DefaultEnforcer) checkBash(token CapabilityToken, input json.RawMessage) error {
	var params struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal(input, &params) //nolint:errcheck // best-effort parse for command extraction

	if params.Command == "" {
		return nil
	}

	// Scan command for denied path references
	for _, denied := range token.DeniedPaths {
		if strings.Contains(params.Command, denied) {
			return fmt.Errorf("%w: command references %s", ErrDeniedBash, denied)
		}
		// Also check with home expansion
		home, _ := os.UserHomeDir()
		if home != "" {
			expanded := strings.Replace(denied, "~", home, 1)
			if strings.Contains(params.Command, expanded) {
				return fmt.Errorf("%w: command references %s", ErrDeniedBash, denied)
			}
		}
	}

	return nil
}

// resolvePath follows symlinks and returns the absolute canonical path.
func resolvePath(path string) (string, error) {
	// Expand ~
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}

	// Absolute
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// Follow symlinks
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// File might not exist yet (Write creating new file)
		// Resolve parent directory instead
		dir := filepath.Dir(abs)
		resolvedDir, dirErr := filepath.EvalSymlinks(dir)
		if dirErr != nil {
			return abs, nil // best effort: return abs
		}
		return filepath.Join(resolvedDir, filepath.Base(abs)), nil
	}

	return resolved, nil
}

// NopEnforcer allows everything. Used when no policy is configured.
type NopEnforcer struct{}

func (NopEnforcer) Check(_ CapabilityToken, _ string, _ json.RawMessage) error { return nil }

// Ensure interface compliance.
var (
	_ Enforcer = (*DefaultEnforcer)(nil)
	_ Enforcer = NopEnforcer{}
)
