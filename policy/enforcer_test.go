package policy

import (
	"context"
	"encoding/json"
	"errors"

	"os"
	"path/filepath"
	"testing"
)

func TestPathTraversal_NestedDotDot(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer(nil)
	token := CapabilityToken{
		WritablePaths: []string{"/workspace"},
	}

	// Four levels of .. from /workspace/a/b/ escapes /workspace entirely.
	input, _ := json.Marshal(map[string]string{
		"file_path": "/workspace/a/b/../../../../etc/passwd",
	})
	err := e.Check(context.Background(), token, "Write", input)
	if err == nil {
		t.Fatal("should deny write: nested .. resolves outside /workspace")
	}
	if !errors.Is(err, ErrDeniedPath) {
		t.Fatalf("err = %v, want ErrDeniedPath", err)
	}
}

func TestPathTraversal_AbsoluteOutsideWorkspace(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer(nil)
	token := CapabilityToken{
		WritablePaths: []string{"/workspace"},
	}

	input, _ := json.Marshal(map[string]string{
		"file_path": "/etc/passwd",
	})
	err := e.Check(context.Background(), token, "Write", input)
	if err == nil {
		t.Fatal("should deny write to /etc/passwd — outside writable workspace")
	}
	if !errors.Is(err, ErrDeniedPath) {
		t.Fatalf("err = %v, want ErrDeniedPath", err)
	}
}

func TestPathTraversal_ReadAllowed(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer(nil)
	token := CapabilityToken{
		// WritablePaths is intentionally empty — Read ignores it.
	}

	input, _ := json.Marshal(map[string]string{
		"path": "/etc/passwd",
	})
	if err := e.Check(context.Background(), token, "Read", input); err != nil {
		t.Fatalf("Read with no DeniedPaths should be allowed: %v", err)
	}
}

func TestPathTraversal_SymlinkChain(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer(nil)

	dir := t.TempDir()
	workspace := filepath.Join(dir, "workspace")
	secret := filepath.Join(dir, "secret")

	os.MkdirAll(workspace, 0o755)
	os.MkdirAll(secret, 0o755)
	os.WriteFile(filepath.Join(secret, "data.txt"), []byte("top-secret"), 0o644)

	// Symlink inside workspace points outside to secret/.
	link := filepath.Join(workspace, "link")
	if err := os.Symlink(secret, link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	token := CapabilityToken{
		WritablePaths: []string{workspace},
	}

	// Agent tries to write through symlink chain.
	input, _ := json.Marshal(map[string]string{
		"file_path": filepath.Join(workspace, "link", "data.txt"),
	})
	err := e.Check(context.Background(), token, "Write", input)
	if err == nil {
		t.Fatal("should deny write through symlink that resolves outside workspace")
	}
	if !errors.Is(err, ErrDeniedPath) {
		t.Fatalf("err = %v, want ErrDeniedPath", err)
	}
}

func TestPathTraversal_DeniedPathExact(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer(nil)
	token := CapabilityToken{
		DeniedPaths: []string{"/protected"},
	}

	input, _ := json.Marshal(map[string]string{
		"path": "/protected/config.yaml",
	})
	err := e.Check(context.Background(), token, "Read", input)
	if err == nil {
		t.Fatal("should deny read of path under DeniedPaths")
	}
	if !errors.Is(err, ErrDeniedPath) {
		t.Fatalf("err = %v, want ErrDeniedPath", err)
	}
}

func TestPathTraversal_WriteThroughSymlink(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer(nil)

	dir := t.TempDir()
	workspace := filepath.Join(dir, "workspace")
	outside := filepath.Join(dir, "outside")

	os.MkdirAll(workspace, 0o755)
	os.MkdirAll(outside, 0o755)
	os.WriteFile(filepath.Join(outside, "target.txt"), []byte("external"), 0o644)

	// File-level symlink: workspace/sneaky → ../outside/target.txt
	sneaky := filepath.Join(workspace, "sneaky")
	if err := os.Symlink(filepath.Join(outside, "target.txt"), sneaky); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	token := CapabilityToken{
		WritablePaths: []string{workspace},
	}

	input, _ := json.Marshal(map[string]string{
		"file_path": sneaky,
	})
	err := e.Check(context.Background(), token, "Write", input)
	if err == nil {
		t.Fatal("should deny write through file symlink resolving outside workspace")
	}
	if !errors.Is(err, ErrDeniedPath) {
		t.Fatalf("err = %v, want ErrDeniedPath", err)
	}
}

func TestBashInjection_Semicolon(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer(nil)
	token := CapabilityToken{DeniedPaths: []string{"/protected"}}
	input := json.RawMessage(`{"command": "ls; cat /protected/secret"}`)

	err := e.Check(context.Background(), token, "Bash", input)
	if err == nil {
		t.Fatal("should deny semicolon-injected command referencing denied path")
	}
	if !errors.Is(err, ErrDeniedBash) {
		t.Fatalf("err = %v, want ErrDeniedBash", err)
	}
}

func TestBashInjection_Pipe(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer(nil)
	token := CapabilityToken{DeniedPaths: []string{"/protected"}}
	input := json.RawMessage(`{"command": "ls | cat /protected/secret"}`)

	err := e.Check(context.Background(), token, "Bash", input)
	if err == nil {
		t.Fatal("should deny pipe-injected command referencing denied path")
	}
	if !errors.Is(err, ErrDeniedBash) {
		t.Fatalf("err = %v, want ErrDeniedBash", err)
	}
}

func TestBashInjection_Subshell(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer(nil)
	token := CapabilityToken{DeniedPaths: []string{"/protected"}}
	input := json.RawMessage(`{"command": "$(cat /protected/secret)"}`)

	err := e.Check(context.Background(), token, "Bash", input)
	if err == nil {
		t.Fatal("should deny subshell command referencing denied path")
	}
	if !errors.Is(err, ErrDeniedBash) {
		t.Fatalf("err = %v, want ErrDeniedBash", err)
	}
}

func TestBashInjection_Backtick(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer(nil)
	token := CapabilityToken{DeniedPaths: []string{"/protected"}}
	input := json.RawMessage("{\"command\": \"`cat /protected/secret`\"}")

	err := e.Check(context.Background(), token, "Bash", input)
	if err == nil {
		t.Fatal("should deny backtick command referencing denied path")
	}
	if !errors.Is(err, ErrDeniedBash) {
		t.Fatalf("err = %v, want ErrDeniedBash", err)
	}
}

func TestBashInjection_And(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer(nil)
	token := CapabilityToken{DeniedPaths: []string{"/protected"}}
	input := json.RawMessage(`{"command": "true && cat /protected/secret"}`)

	err := e.Check(context.Background(), token, "Bash", input)
	if err == nil {
		t.Fatal("should deny &&-chained command referencing denied path")
	}
	if !errors.Is(err, ErrDeniedBash) {
		t.Fatalf("err = %v, want ErrDeniedBash", err)
	}
}

func TestBashInjection_VariableExpansion(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer(nil)
	token := CapabilityToken{DeniedPaths: []string{"/protected"}}
	// The literal string "/protected" appears in "f=/protected/secret",
	// so strings.Contains catches it.
	input := json.RawMessage(`{"command": "f=/protected/secret; cat $f"}`)

	err := e.Check(context.Background(), token, "Bash", input)
	if err == nil {
		t.Fatal("should deny command containing literal denied path in variable assignment")
	}
	if !errors.Is(err, ErrDeniedBash) {
		t.Fatalf("err = %v, want ErrDeniedBash", err)
	}
}

func TestBashInjection_VariableExpansionEvasion(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer(nil)
	token := CapabilityToken{DeniedPaths: []string{"/protected"}}
	// The denied string "/protected" never appears literally — it is
	// assembled at shell runtime via variable concatenation.
	// Known limitation: the belt layer (string scan) cannot catch this.
	// Defense: sandbox layer prevents actual access.
	input := json.RawMessage(`{"command": "f=/pro; f=${f}tected/secret; cat $f"}`)

	err := e.Check(context.Background(), token, "Bash", input)
	if err != nil {
		t.Fatalf("should not catch evasion via variable concatenation (known limitation): %v", err)
	}
}

func TestBashInjection_AllowedCommand(t *testing.T) {
	e := NewDefaultToolPolicyEnforcer(nil)
	token := CapabilityToken{DeniedPaths: []string{"/protected"}}
	input := json.RawMessage(`{"command": "ls -la /safe/dir"}`)

	err := e.Check(context.Background(), token, "Bash", input)
	if err != nil {
		t.Fatalf("should allow command not referencing denied path: %v", err)
	}
}
