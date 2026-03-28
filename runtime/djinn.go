// Package runtime provides the Djinn facade — the single entry point
// for all domain subsystems. The TUI holds one *Djinn instead of
// 11 separate references.
//
// Djinn wires staff roles, capability clearance, policy enforcement,
// sandbox execution, and session management into one cohesive surface.
package runtime

import (
	"context"
	"sort"

	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/staff"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// Config holds the inputs needed to construct a Djinn facade.
type Config struct {
	StaffCfg *staff.StaffConfig
	Registry *builtin.Registry
	Enforcer policy.ToolPolicyEnforcer
	Token    policy.CapabilityToken
	Session  *session.Session

	// Initial role name (must exist in StaffCfg).
	InitialRole string

	// Sandbox settings — empty SandboxHandle means unsandboxed.
	SandboxHandle string
	SandboxExec   func(ctx context.Context, cmd []string) (stdout, stderr string, err error)
}

// Djinn wraps all domain subsystems into one human-readable facade.
// The TUI holds one *Djinn instead of 11 separate references.
type Djinn struct {
	staffCfg      *staff.StaffConfig
	clearance     *staff.ToolClearance
	enforcer      policy.ToolPolicyEnforcer
	token         policy.CapabilityToken
	session       *session.Session
	currentRole   string
	sandboxHandle string
	sandboxExec   func(ctx context.Context, cmd []string) (string, string, error)
}

// New creates a Djinn facade from the given configuration.
// It initializes the ToolClearance with the initial role and
// wires all subsystems together.
func New(cfg Config) *Djinn {
	if cfg.StaffCfg == nil {
		cfg.StaffCfg = staff.DefaultConfig()
	}
	if cfg.Registry == nil {
		cfg.Registry = builtin.NewRegistry()
	}
	if cfg.Enforcer == nil {
		cfg.Enforcer = policy.NopToolPolicyEnforcer{}
	}
	if cfg.Session == nil {
		cfg.Session = session.New("default", "default", ".")
	}
	if cfg.InitialRole == "" {
		cfg.InitialRole = "gensec"
	}

	d := &Djinn{
		staffCfg:      cfg.StaffCfg,
		clearance:     staff.NewToolClearance(cfg.StaffCfg, cfg.Registry, cfg.InitialRole),
		enforcer:      cfg.Enforcer,
		token:         cfg.Token,
		session:       cfg.Session,
		currentRole:   cfg.InitialRole,
		sandboxHandle: cfg.SandboxHandle,
		sandboxExec:   cfg.SandboxExec,
	}
	return d
}

// SwitchRole changes the active role, updating ToolClearance and the
// capability token's AllowedTools to match the new role's capabilities.
func (d *Djinn) SwitchRole(role string) {
	d.currentRole = role
	d.clearance.SetRole(role)

	// Rebuild the token's AllowedTools from the new role.
	d.token.AllowedTools = d.staffCfg.ResolveToolNames(
		d.roleCapabilities(role),
	)
}

// CurrentRole returns the active role name.
func (d *Djinn) CurrentRole() string {
	return d.currentRole
}

// ResolvedTools returns the sorted list of tool names available
// to the current role after capability resolution.
func (d *Djinn) ResolvedTools() []string {
	names := d.clearance.Names()
	sort.Strings(names)
	return names
}

// IsSandboxed returns true if the Djinn instance is running inside
// a sandbox (SandboxHandle is set).
func (d *Djinn) IsSandboxed() bool {
	return d.sandboxHandle != ""
}

// Session returns the current conversation session.
func (d *Djinn) Session() *session.Session {
	return d.session
}

// Clearance returns the ToolClearance for direct use by the agent loop.
func (d *Djinn) Clearance() *staff.ToolClearance {
	return d.clearance
}

// Enforcer returns the ToolPolicyEnforcer.
func (d *Djinn) Enforcer() policy.ToolPolicyEnforcer {
	return d.enforcer
}

// Token returns the current CapabilityToken.
func (d *Djinn) Token() policy.CapabilityToken {
	return d.token
}

// StaffConfig returns the staff configuration.
func (d *Djinn) StaffConfig() *staff.StaffConfig {
	return d.staffCfg
}

// SandboxHandle returns the sandbox handle (empty if unsandboxed).
func (d *Djinn) SandboxHandle() string {
	return d.sandboxHandle
}

// SandboxExec returns the sandbox execution function, or nil if unsandboxed.
func (d *Djinn) SandboxExec() func(ctx context.Context, cmd []string) (string, string, error) {
	return d.sandboxExec
}

// roleCapabilities returns the ToolCapabilities list for the given role.
func (d *Djinn) roleCapabilities(role string) []string {
	roleMap := d.staffCfg.RoleMap()
	if r, ok := roleMap[role]; ok {
		return r.ToolCapabilities
	}
	return nil
}
