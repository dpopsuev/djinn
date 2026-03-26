package runtime

import (
	"context"
	"testing"

	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/staff"
	"github.com/dpopsuev/djinn/tools/builtin"
)

func testStaffConfig() *staff.StaffConfig {
	return &staff.StaffConfig{
		Roles: []staff.Role{
			{Name: "gensec", Mode: "plan", ToolCapabilities: []string{"WorkTracking"}},
			{Name: "executor", Mode: "agent", ToolCapabilities: []string{"WorkTracking", "FileEditing", "ShellExecution"}},
			{Name: "inspector", Mode: "plan", ToolCapabilities: []string{"FileEditing"}},
		},
		ToolCapabilities: []staff.ToolCapability{
			{Name: "WorkTracking", Backend: "scribe", Tools: []string{"artifact", "graph"}},
			{Name: "FileEditing", Backend: "builtin", Tools: []string{"Read", "Write", "Edit"}},
			{Name: "ShellExecution", Backend: "builtin", Tools: []string{"Bash"}},
		},
	}
}

func TestNew_DefaultConfig(t *testing.T) {
	d := New(Config{})

	if d.CurrentRole() != "gensec" {
		t.Fatalf("default role = %q, want gensec", d.CurrentRole())
	}
	if d.Session() == nil {
		t.Fatal("Session should not be nil")
	}
	if d.StaffConfig() == nil {
		t.Fatal("StaffConfig should not be nil")
	}
	if d.Enforcer() == nil {
		t.Fatal("Enforcer should not be nil")
	}
}

func TestNew_CustomConfig(t *testing.T) {
	cfg := Config{
		StaffCfg:      testStaffConfig(),
		Registry:      builtin.NewRegistry(),
		Enforcer:      policy.NopToolPolicyEnforcer{},
		Session:       session.New("test-id", "claude-4", "/tmp/work"),
		InitialRole:   "executor",
		SandboxHandle: "jail-42",
	}
	d := New(cfg)

	if d.CurrentRole() != "executor" {
		t.Fatalf("role = %q, want executor", d.CurrentRole())
	}
	if !d.IsSandboxed() {
		t.Fatal("should be sandboxed with handle set")
	}
	if d.SandboxHandle() != "jail-42" {
		t.Fatalf("SandboxHandle = %q, want jail-42", d.SandboxHandle())
	}
}

func TestSwitchRole(t *testing.T) {
	cfg := Config{
		StaffCfg: testStaffConfig(),
		Registry: builtin.NewRegistry(),
	}
	d := New(cfg)

	// Start as gensec — should NOT see Read.
	tools := d.ResolvedTools()
	for _, name := range tools {
		if name == "Read" {
			t.Fatal("gensec should not see Read")
		}
	}

	// Switch to executor — should see Read.
	d.SwitchRole("executor")
	if d.CurrentRole() != "executor" {
		t.Fatalf("role = %q, want executor", d.CurrentRole())
	}

	tools = d.ResolvedTools()
	found := false
	for _, name := range tools {
		if name == "Read" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("executor should see Read, got %v", tools)
	}
}

func TestSwitchRole_UpdatesToken(t *testing.T) {
	cfg := Config{
		StaffCfg:    testStaffConfig(),
		Registry:    builtin.NewRegistry(),
		InitialRole: "gensec",
	}
	d := New(cfg)

	d.SwitchRole("executor")
	token := d.Token()

	// Token should now include tools from WorkTracking + FileEditing + ShellExecution.
	allowed := make(map[string]bool)
	for _, t := range token.AllowedTools {
		allowed[t] = true
	}
	if !allowed["Read"] {
		t.Fatal("token should include Read after switch to executor")
	}
	if !allowed["Bash"] {
		t.Fatal("token should include Bash after switch to executor")
	}
	if !allowed["artifact"] {
		t.Fatal("token should include artifact after switch to executor")
	}
}

func TestIsSandboxed(t *testing.T) {
	unsandboxed := New(Config{})
	if unsandboxed.IsSandboxed() {
		t.Fatal("should not be sandboxed with empty handle")
	}

	sandboxed := New(Config{SandboxHandle: "jail-1"})
	if !sandboxed.IsSandboxed() {
		t.Fatal("should be sandboxed with handle set")
	}
}

func TestSandboxExec_Nil(t *testing.T) {
	d := New(Config{})
	if d.SandboxExec() != nil {
		t.Fatal("SandboxExec should be nil when not configured")
	}
}

func TestSandboxExec_Set(t *testing.T) {
	fn := func(_ context.Context, _ []string) (string, string, error) {
		return "out", "err", nil
	}
	d := New(Config{
		SandboxHandle: "jail-1",
		SandboxExec:   fn,
	})
	if d.SandboxExec() == nil {
		t.Fatal("SandboxExec should not be nil when configured")
	}
	out, stderr, err := d.SandboxExec()(context.Background(), []string{"echo", "hi"})
	if out != "out" || stderr != "err" || err != nil {
		t.Fatalf("unexpected result: %q %q %v", out, stderr, err)
	}
}

func TestResolvedTools_Sorted(t *testing.T) {
	cfg := Config{
		StaffCfg:    testStaffConfig(),
		Registry:    builtin.NewRegistry(),
		InitialRole: "executor",
	}
	d := New(cfg)

	tools := d.ResolvedTools()
	for i := 1; i < len(tools); i++ {
		if tools[i] < tools[i-1] {
			t.Fatalf("tools not sorted: %v", tools)
		}
	}
}

func TestSwitchRole_UnknownRole(t *testing.T) {
	d := New(Config{
		StaffCfg: testStaffConfig(),
		Registry: builtin.NewRegistry(),
	})

	d.SwitchRole("nonexistent")
	if d.CurrentRole() != "nonexistent" {
		t.Fatalf("role = %q, want nonexistent", d.CurrentRole())
	}
	// Unknown role should have no tools.
	if len(d.ResolvedTools()) != 0 {
		t.Fatalf("unknown role should see 0 tools, got %v", d.ResolvedTools())
	}
}

func TestClearance_ReturnsSameInstance(t *testing.T) {
	d := New(Config{
		StaffCfg: testStaffConfig(),
		Registry: builtin.NewRegistry(),
	})
	c1 := d.Clearance()
	c2 := d.Clearance()
	if c1 != c2 {
		t.Fatal("Clearance should return the same instance")
	}
}
