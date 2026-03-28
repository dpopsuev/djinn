// ports.go — hexagonal port interfaces for the staffed runtime.
//
// Four ports define the contract between Djinn core and its adapters:
//   - RoleManager: switch roles, query available roles
//   - QualityGate: mechanical quality checks on executor completion
//   - AgentIdentity: human-readable labels for roles (future: Bugle colors)
//   - RoleMemory is defined in memory.go (already a port)
//
// Default implementations live alongside (DefaultConfig, NewRoleMemory).
// Future adapters: Bugle (identity), Origami (circuit roles), Sophia (memory).
package staff

import (
	"context"
	"fmt"
	"os/exec"
)

// RoleManager switches between staff roles.
// Default: PromptRoleManager reads from staff.yaml / DefaultConfig.
// Bugle adapter: roles are ECS entities with ColorIdentity.
// Origami adapter: roles are circuit nodes, transitions follow edges.
type RoleManager interface {
	Switch(role string) error
	Current() Role
	Available() []Role
}

// GateResult is the outcome of a mechanical quality check.
type GateResult struct {
	Passed      bool
	Diagnostics []Diagnostic
}

// Diagnostic is a single finding from a quality gate source.
type Diagnostic struct {
	Source  string // "limes", "locus", "scribe", "make"
	Level   string // "error", "warning", "info"
	Message string
}

// QualityGate runs mechanical quality checks on executor completion.
// Default: MakeCircuitGate runs `make circuit` and parses exit code.
// Full adapter: TripleGate calls Limes + Locus + Scribe MCP.
// QualityGate runs mechanical quality checks on executor completion.
// workDir specifies the directory to run checks in (worktree path for
// isolated tasks, repo root otherwise).
type QualityGate interface {
	Check(ctx context.Context, workDir string) (GateResult, error)
}

// AgentIdentity provides human-readable labels for roles.
// Default: StringIdentity returns the role name.
// Bugle adapter: assigns ColorIdentity from the palette.
type AgentIdentity interface {
	Label(role string) string
}

// --- Default implementations ---

// PromptRoleManager manages roles from staff config.
type PromptRoleManager struct {
	roles   map[string]Role
	current string
}

// NewPromptRoleManager creates a role manager from config.
func NewPromptRoleManager(cfg *StaffConfig) *PromptRoleManager {
	return &PromptRoleManager{
		roles:   cfg.RoleMap(),
		current: "gensec",
	}
}

func (m *PromptRoleManager) Switch(role string) error {
	if _, ok := m.roles[role]; !ok {
		return ErrRoleNotFound
	}
	m.current = role
	return nil
}

func (m *PromptRoleManager) Current() Role {
	return m.roles[m.current]
}

func (m *PromptRoleManager) Available() []Role {
	roles := make([]Role, 0, len(m.roles))
	for _, r := range m.roles {
		roles = append(roles, r)
	}
	return roles
}

// MakeCircuitGate runs `make circuit` as the quality check.
type MakeCircuitGate struct {
	Command string // default: "make"
	Target  string // default: "circuit"
}

func (g *MakeCircuitGate) Check(ctx context.Context, workDir string) (GateResult, error) {
	cmd := g.Command
	if cmd == "" {
		cmd = "make"
	}
	target := g.Target
	if target == "" {
		target = "circuit"
	}

	out, err := execCommandInDir(ctx, workDir, cmd, target)
	if err != nil {
		return GateResult{
			Passed: false,
			Diagnostics: []Diagnostic{
				{Source: "make", Level: "error", Message: string(out)},
			},
		}, nil
	}
	return GateResult{Passed: true}, nil
}

// StringIdentity returns the role name as the label.
type StringIdentity struct{}

func (StringIdentity) Label(role string) string { return role }

// Sentinel errors.
var ErrRoleNotFound = fmt.Errorf("role not found")

// execCommandInDir runs a command in the given directory.
func execCommandInDir(ctx context.Context, dir, command, target string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, target)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd.CombinedOutput()
}

// Interface compliance.
var (
	_ RoleManager   = (*PromptRoleManager)(nil)
	_ QualityGate   = (*MakeCircuitGate)(nil)
	_ AgentIdentity = (StringIdentity{})
)
