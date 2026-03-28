// policy.go — declarative sandbox policy from sandbox.json.
//
// Loaded from workspace root. Defines allowed commands, denied paths,
// and network access. Wire into ToolPolicyEnforcer on startup.
package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Policy file name.
const policyFileName = "sandbox.json"

// Sentinel errors.
var (
	ErrPolicyNotFound = errors.New("sandbox.json not found")
	ErrCommandDenied  = errors.New("command denied by sandbox policy")
	ErrPathDenied     = errors.New("path denied by sandbox policy")
)

// Policy defines declarative sandbox constraints loaded from sandbox.json.
type Policy struct {
	AllowedCommands []string `json:"allowed_commands"` // prefix whitelist (e.g. "git", "go test")
	DeniedPaths     []string `json:"denied_paths"`     // path prefixes denied for read/write
	AllowNetwork    bool     `json:"allow_network"`    // whether network access is permitted
}

// LoadPolicy reads sandbox.json from the workspace root.
// Returns ErrPolicyNotFound if the file doesn't exist (not an error — policy is optional).
func LoadPolicy(workDir string) (*Policy, error) {
	path := filepath.Join(workDir, policyFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrPolicyNotFound, path)
		}
		return nil, fmt.Errorf("read %s: %w", policyFileName, err)
	}

	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse %s: %w", policyFileName, err)
	}
	return &p, nil
}

// AllowCommand checks if a command is allowed by the whitelist.
// Empty whitelist = all commands allowed.
func (p *Policy) AllowCommand(cmd string) bool {
	if len(p.AllowedCommands) == 0 {
		return true
	}
	for _, prefix := range p.AllowedCommands {
		if strings.HasPrefix(cmd, prefix) {
			return true
		}
	}
	return false
}

// AllowPath checks if a path is not in the denied list.
func (p *Policy) AllowPath(path string) bool {
	for _, denied := range p.DeniedPaths {
		if strings.HasPrefix(path, denied) {
			return false
		}
	}
	return true
}
