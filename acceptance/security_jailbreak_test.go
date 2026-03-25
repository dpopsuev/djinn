// security_jailbreak_test.go — acceptance tests for agent call mediation.
//
// Spec: SPC-35 — Agent Call Mediation
// Covers:
//   - Write to config path → denied
//   - Edit to config path → denied
//   - Symlink to config → resolved, denied
//   - Path traversal (../) → resolved, denied
//   - Bash targeting config → denied
//   - Bash not targeting config → allowed
//   - Write outside workspace → denied
//   - Read always allowed (even outside workspace)
//   - Tool whitelist enforcement
//   - Capability token from workspace
//   - NopToolPolicyEnforcer allows everything
package acceptance

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/workspace"
)

func enforcer() *policy.DefaultToolPolicyEnforcer {
	return policy.NewDefaultToolPolicyEnforcer()
}

func TestSecurity_WriteConfigDenied(t *testing.T) {
	e := enforcer()
	home, _ := os.UserHomeDir()
	token := policy.CapabilityToken{
		DeniedPaths: []string{filepath.Join(home, ".config", "djinn")},
	}
	input, _ := json.Marshal(map[string]string{
		"path": filepath.Join(home, ".config", "djinn", "workspaces", "aeon.yaml"),
	})
	err := e.Check(token, "Write", input)
	if err == nil {
		t.Fatal("JAILBREAK: write to config path should be denied")
	}
	if !errors.Is(err, policy.ErrDeniedPath) {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestSecurity_EditConfigDenied(t *testing.T) {
	e := enforcer()
	home, _ := os.UserHomeDir()
	token := policy.CapabilityToken{
		DeniedPaths: []string{filepath.Join(home, ".config", "djinn")},
	}
	input, _ := json.Marshal(map[string]string{
		"file_path": filepath.Join(home, ".config", "djinn", "config.yaml"),
	})
	err := e.Check(token, "Edit", input)
	if err == nil {
		t.Fatal("JAILBREAK: edit to config path should be denied")
	}
}

func TestSecurity_SymlinkBypass(t *testing.T) {
	e := enforcer()

	dir := t.TempDir()
	configDir := filepath.Join(dir, "protected")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "secret.yaml"), []byte("secret"), 0644)

	symlink := filepath.Join(dir, "innocent-link")
	os.Symlink(configDir, symlink)

	token := policy.CapabilityToken{
		DeniedPaths: []string{configDir},
	}

	input, _ := json.Marshal(map[string]string{
		"path": filepath.Join(symlink, "secret.yaml"),
	})
	err := e.Check(token, "Write", input)
	if err == nil {
		t.Fatal("JAILBREAK: symlink to protected path should be denied")
	}
}

func TestSecurity_PathTraversal(t *testing.T) {
	e := enforcer()

	dir := t.TempDir()
	configDir := filepath.Join(dir, "config")
	os.MkdirAll(configDir, 0755)

	token := policy.CapabilityToken{
		DeniedPaths: []string{configDir},
	}

	traversal := filepath.Join(dir, "safe", "..", "config", "evil.yaml")
	input, _ := json.Marshal(map[string]string{"path": traversal})
	err := e.Check(token, "Write", input)
	if err == nil {
		t.Fatal("JAILBREAK: path traversal to protected path should be denied")
	}
}

func TestSecurity_BashConfigDenied(t *testing.T) {
	e := enforcer()
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "djinn")
	token := policy.CapabilityToken{
		DeniedPaths: []string{configDir},
	}

	commands := []string{
		"sed -i 's/container/none/' " + filepath.Join(configDir, "workspaces", "aeon.yaml"),
		"cp /tmp/evil.yaml " + configDir,
		"mv /tmp/evil.yaml " + filepath.Join(configDir, "config.yaml"),
		"rm -rf " + configDir,
	}

	for _, cmd := range commands {
		input, _ := json.Marshal(map[string]string{"command": cmd})
		err := e.Check(token, "Bash", input)
		if err == nil {
			t.Fatalf("JAILBREAK: bash command should be denied: %s", cmd)
		}
	}
}

func TestSecurity_BashSafeAllowed(t *testing.T) {
	e := enforcer()
	token := policy.CapabilityToken{
		DeniedPaths: []string{"/protected"},
	}

	safeCmds := []string{
		"go test ./...",
		"git status",
		"ls -la /home/user/project",
	}

	for _, cmd := range safeCmds {
		input, _ := json.Marshal(map[string]string{"command": cmd})
		err := e.Check(token, "Bash", input)
		if err != nil {
			t.Fatalf("safe command should be allowed: %s: %v", cmd, err)
		}
	}
}

func TestSecurity_WriteOutsideWorkspace(t *testing.T) {
	e := enforcer()
	token := policy.CapabilityToken{
		WritablePaths: []string{"/home/user/project"},
	}
	input, _ := json.Marshal(map[string]string{"path": "/etc/passwd"})
	err := e.Check(token, "Write", input)
	if err == nil {
		t.Fatal("JAILBREAK: write outside workspace should be denied")
	}
}

func TestSecurity_ReadAlwaysAllowed(t *testing.T) {
	e := enforcer()
	token := policy.CapabilityToken{
		WritablePaths: []string{"/home/user/project"},
	}
	input, _ := json.Marshal(map[string]string{"path": "/etc/hosts"})
	err := e.Check(token, "Read", input)
	if err != nil {
		t.Fatalf("Read should always be allowed: %v", err)
	}
}

func TestSecurity_ToolWhitelist(t *testing.T) {
	e := enforcer()
	token := policy.CapabilityToken{
		AllowedTools: []string{"Read", "Grep"},
	}

	if err := e.Check(token, "Read", nil); err != nil {
		t.Fatalf("whitelisted tool denied: %v", err)
	}

	err := e.Check(token, "Bash", nil)
	if err == nil {
		t.Fatal("non-whitelisted tool should be denied")
	}
	if !errors.Is(err, policy.ErrDeniedTool) {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestSecurity_CapabilityTokenFromWorkspace(t *testing.T) {
	ws := &workspace.Workspace{
		Name: "test",
		Repos: []workspace.Repo{
			{Path: "/home/user/project", Role: "primary"},
			{Path: "/home/user/lib", Role: "dependency"},
		},
	}

	token := ws.ToCapabilityToken()

	if len(token.WritablePaths) != 2 {
		t.Fatalf("writable = %d, want 2", len(token.WritablePaths))
	}
	if len(token.DeniedPaths) < 2 {
		t.Fatalf("denied = %d, want >= 2 (config dirs)", len(token.DeniedPaths))
	}
}

func TestSecurity_CapabilityTokenDeniesConfig(t *testing.T) {
	ws := &workspace.Workspace{
		Name:  "test",
		Repos: []workspace.Repo{{Path: "/project", Role: "primary"}},
	}
	token := ws.ToCapabilityToken()
	e := enforcer()

	// Config paths should be in DeniedPaths
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "djinn", "test.yaml")
	input, _ := json.Marshal(map[string]string{"path": configPath})

	err := e.Check(token, "Write", input)
	if err == nil {
		t.Fatal("JAILBREAK: capability token should deny config writes")
	}
}

func TestSecurity_NopToolPolicyEnforcerAllowsEverything(t *testing.T) {
	e := policy.NopToolPolicyEnforcer{}
	token := policy.CapabilityToken{DeniedPaths: []string{"/protected"}}
	input, _ := json.Marshal(map[string]string{"path": "/protected/secret"})
	if err := e.Check(token, "Write", input); err != nil {
		t.Fatal("NopToolPolicyEnforcer should allow everything")
	}
}
