package policy

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultToolPolicyEnforcer_AllowedWrite(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer()
	token := CapabilityToken{
		WritablePaths: []string{"/home/user/project"},
	}
	input, _ := json.Marshal(map[string]string{"path": "/home/user/project/main.go"})
	if err := e.Check(token, "Write", input); err != nil {
		t.Fatalf("should allow: %v", err)
	}
}

func TestDefaultToolPolicyEnforcer_DeniedPath(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer()
	home, _ := os.UserHomeDir()
	token := CapabilityToken{
		DeniedPaths: []string{filepath.Join(home, ".config", "djinn")},
	}
	input, _ := json.Marshal(map[string]string{"path": filepath.Join(home, ".config", "djinn", "workspaces", "aeon.yaml")})
	err := e.Check(token, "Write", input)
	if err == nil {
		t.Fatal("should deny write to config path")
	}
	if !errors.Is(err, ErrDeniedPath) {
		t.Fatalf("err = %v, want ErrDeniedPath", err)
	}
}

func TestDefaultToolPolicyEnforcer_DeniedEdit(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer()
	home, _ := os.UserHomeDir()
	token := CapabilityToken{
		DeniedPaths: []string{filepath.Join(home, ".config", "djinn")},
	}
	input, _ := json.Marshal(map[string]string{"file_path": filepath.Join(home, ".config", "djinn", "config.yaml")})
	err := e.Check(token, "Edit", input)
	if err == nil {
		t.Fatal("should deny edit to config path")
	}
}

func TestDefaultToolPolicyEnforcer_SymlinkBypass(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer()

	// Create a real config dir and a symlink to it
	dir := t.TempDir()
	configDir := filepath.Join(dir, "config")
	os.MkdirAll(configDir, 0o755)
	os.WriteFile(filepath.Join(configDir, "secret.yaml"), []byte("secret"), 0o644)

	symlink := filepath.Join(dir, "innocent")
	os.Symlink(configDir, symlink)

	token := CapabilityToken{
		DeniedPaths: []string{configDir},
	}

	// Agent tries to write via symlink
	input, _ := json.Marshal(map[string]string{"path": filepath.Join(symlink, "secret.yaml")})
	err := e.Check(token, "Write", input)
	if err == nil {
		t.Fatal("should deny write through symlink to protected path")
	}
}

func TestDefaultToolPolicyEnforcer_PathTraversal(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer()
	dir := t.TempDir()
	configDir := filepath.Join(dir, "config")
	os.MkdirAll(configDir, 0o755)

	token := CapabilityToken{
		DeniedPaths: []string{configDir},
	}

	// Path traversal: dir/other/../config/file
	traversal := filepath.Join(dir, "other", "..", "config", "file.yaml")
	input, _ := json.Marshal(map[string]string{"path": traversal})
	err := e.Check(token, "Write", input)
	if err == nil {
		t.Fatal("should deny path traversal to protected path")
	}
}

func TestDefaultToolPolicyEnforcer_BashDenied(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer()
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "djinn")
	token := CapabilityToken{
		DeniedPaths: []string{configDir},
	}

	input, _ := json.Marshal(map[string]string{"command": "sed -i 's/container/none/' " + filepath.Join(configDir, "workspaces", "aeon.yaml")})
	err := e.Check(token, "Bash", input)
	if err == nil {
		t.Fatal("should deny bash command targeting config")
	}
	if !errors.Is(err, ErrDeniedBash) {
		t.Fatalf("err = %v, want ErrDeniedBash", err)
	}
}

func TestDefaultToolPolicyEnforcer_BashAllowed(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer()
	token := CapabilityToken{
		DeniedPaths: []string{"/protected"},
	}
	input, _ := json.Marshal(map[string]string{"command": "go test ./..."})
	if err := e.Check(token, "Bash", input); err != nil {
		t.Fatalf("should allow: %v", err)
	}
}

func TestDefaultToolPolicyEnforcer_WriteOutsideWorkspace(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer()
	token := CapabilityToken{
		WritablePaths: []string{"/home/user/project"},
	}
	input, _ := json.Marshal(map[string]string{"path": "/etc/passwd"})
	err := e.Check(token, "Write", input)
	if err == nil {
		t.Fatal("should deny write outside workspace")
	}
}

func TestDefaultToolPolicyEnforcer_ReadAlwaysAllowed(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer()
	token := CapabilityToken{
		WritablePaths: []string{"/home/user/project"},
	}
	// Read should work even outside writable paths (read is not write)
	input, _ := json.Marshal(map[string]string{"path": "/etc/hosts"})
	if err := e.Check(token, "Read", input); err != nil {
		t.Fatalf("Read should be allowed: %v", err)
	}
}

func TestDefaultToolPolicyEnforcer_ToolWhitelist(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer()
	token := CapabilityToken{
		AllowedTools: []string{"Read", "Grep"},
	}
	if err := e.Check(token, "Read", nil); err != nil {
		t.Fatalf("Read should be allowed: %v", err)
	}
	err := e.Check(token, "Bash", nil)
	if err == nil {
		t.Fatal("Bash should be denied")
	}
	if !errors.Is(err, ErrDeniedTool) {
		t.Fatalf("err = %v, want ErrDeniedTool", err)
	}
}

func TestDefaultToolPolicyEnforcer_EmptyWhitelistAllowsAll(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer()
	token := CapabilityToken{}
	if err := e.Check(token, "Bash", nil); err != nil {
		t.Fatalf("empty whitelist should allow all: %v", err)
	}
}

func TestNopToolPolicyEnforcer_AllowsEverything(t *testing.T) {
	e := NopToolPolicyEnforcer{}
	token := CapabilityToken{DeniedPaths: []string{"/protected"}}
	input, _ := json.Marshal(map[string]string{"path": "/protected/secret"})
	if err := e.Check(token, "Write", input); err != nil {
		t.Fatalf("NopToolPolicyEnforcer should allow everything: %v", err)
	}
}
