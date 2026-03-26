package sandbox

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// ═══ RED ═══

func TestLoadPolicy_NotFound(t *testing.T) {
	_, err := LoadPolicy(t.TempDir())
	if !errors.Is(err, ErrPolicyNotFound) {
		t.Fatalf("err = %v, want ErrPolicyNotFound", err)
	}
}

func TestLoadPolicy_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, policyFileName), []byte("not json"), 0644)
	_, err := LoadPolicy(dir)
	if err == nil {
		t.Fatal("should fail on invalid JSON")
	}
}

// ═══ GREEN ═══

func TestLoadPolicy_Valid(t *testing.T) {
	dir := t.TempDir()
	p := Policy{
		AllowedCommands: []string{"git", "go test"},
		DeniedPaths:     []string{"/etc", "/root"},
		AllowNetwork:    false,
	}
	data, _ := json.Marshal(p)
	os.WriteFile(filepath.Join(dir, policyFileName), data, 0644)

	loaded, err := LoadPolicy(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.AllowedCommands) != 2 {
		t.Fatalf("commands = %d", len(loaded.AllowedCommands))
	}
	if loaded.AllowNetwork {
		t.Fatal("should be false")
	}
}

func TestPolicy_AllowCommand(t *testing.T) {
	p := &Policy{AllowedCommands: []string{"git", "go test"}}
	if !p.AllowCommand("git status") {
		t.Fatal("git status should be allowed")
	}
	if !p.AllowCommand("go test ./...") {
		t.Fatal("go test should be allowed")
	}
	if p.AllowCommand("rm -rf /") {
		t.Fatal("rm should be denied")
	}
}

func TestPolicy_AllowCommand_EmptyWhitelist(t *testing.T) {
	p := &Policy{AllowedCommands: nil}
	if !p.AllowCommand("anything") {
		t.Fatal("empty whitelist should allow all")
	}
}

func TestPolicy_AllowPath(t *testing.T) {
	p := &Policy{DeniedPaths: []string{"/etc", "/root"}}
	if p.AllowPath("/etc/passwd") {
		t.Fatal("/etc/passwd should be denied")
	}
	if !p.AllowPath("/home/user/code") {
		t.Fatal("/home should be allowed")
	}
}

// ═══ BLUE ═══

func TestPolicy_AllowPath_EmptyDenied(t *testing.T) {
	p := &Policy{DeniedPaths: nil}
	if !p.AllowPath("/anything") {
		t.Fatal("empty denied list should allow all")
	}
}
