package misbah

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/tier"
)

func TestSandboxPort_BuildSpec_ModTier(t *testing.T) {
	s := &SandboxPort{workspace: "/home/user/workspace"}
	spec := s.buildSpec("djinn-auth-1", tier.Scope{Level: tier.Mod, Name: "auth"})

	if spec.Metadata.Name != "djinn-auth-1" {
		t.Fatalf("Name = %q, want %q", spec.Metadata.Name, "djinn-auth-1")
	}
	if spec.Version != specVersion {
		t.Fatalf("Version = %q, want %q", spec.Version, specVersion)
	}
	if !spec.Namespaces.User {
		t.Fatal("User namespace should be enabled")
	}
	if !spec.Namespaces.Mount {
		t.Fatal("Mount namespace should be enabled")
	}
	if len(spec.Mounts) != 2 {
		t.Fatalf("Mounts = %d, want 2", len(spec.Mounts))
	}

	// Mod tier gets RW workspace
	wsMount := spec.Mounts[0]
	if wsMount.Source != "/home/user/workspace" {
		t.Fatalf("workspace source = %q", wsMount.Source)
	}
	if wsMount.Destination != defaultCwd {
		t.Fatalf("workspace dest = %q", wsMount.Destination)
	}
	hasRW := false
	for _, opt := range wsMount.Options {
		if opt == "rw" {
			hasRW = true
		}
	}
	if !hasRW {
		t.Fatalf("Mod tier should have rw mount, got %v", wsMount.Options)
	}

	// Tier config
	if spec.TierConfig == nil {
		t.Fatal("TierConfig should be set")
	}
	if spec.TierConfig.Tier != "mod" {
		t.Fatalf("TierConfig.Tier = %q, want %q", spec.TierConfig.Tier, "mod")
	}
	if len(spec.TierConfig.WritablePaths) != 1 || spec.TierConfig.WritablePaths[0] != "auth" {
		t.Fatalf("WritablePaths = %v, want [auth]", spec.TierConfig.WritablePaths)
	}
}

func TestSandboxPort_BuildSpec_EcoTier(t *testing.T) {
	s := &SandboxPort{workspace: "/workspace"}
	spec := s.buildSpec("djinn-eco-1", tier.Scope{Level: tier.Eco, Name: "root"})

	// Eco tier gets RO workspace
	wsMount := spec.Mounts[0]
	hasRO := false
	for _, opt := range wsMount.Options {
		if opt == "ro" {
			hasRO = true
		}
	}
	if !hasRO {
		t.Fatalf("Eco tier should have ro mount, got %v", wsMount.Options)
	}
}

func TestSandboxPort_BuildSpec_Validation(t *testing.T) {
	s := &SandboxPort{workspace: "/workspace"}
	spec := s.buildSpec("djinn-test-1", tier.Scope{Level: tier.Mod, Name: "auth"})

	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestMountOptionsForTier(t *testing.T) {
	tests := []struct {
		level tier.TierLevel
		wantRW bool
	}{
		{tier.Eco, false},
		{tier.Sys, false},
		{tier.Com, true},
		{tier.Mod, true},
	}
	for _, tt := range tests {
		opts := mountOptionsForTier(tt.level)
		hasRW := false
		for _, opt := range opts {
			if opt == "rw" {
				hasRW = true
			}
		}
		if hasRW != tt.wantRW {
			t.Fatalf("tier %s: rw=%v, want %v (opts=%v)", tt.level, hasRW, tt.wantRW, opts)
		}
	}
}

func TestSandboxPort_BuildSpec_SysTier(t *testing.T) {
	s := &SandboxPort{workspace: "/workspace"}
	spec := s.buildSpec("djinn-sys-1", tier.Scope{Level: tier.Sys, Name: "sys"})
	wsMount := spec.Mounts[0]
	hasRO := false
	for _, opt := range wsMount.Options {
		if opt == "ro" {
			hasRO = true
		}
	}
	if !hasRO {
		t.Fatalf("Sys tier should have ro mount, got %v", wsMount.Options)
	}
}

func TestSandboxPort_BuildSpec_ComTier(t *testing.T) {
	s := &SandboxPort{workspace: "/workspace"}
	spec := s.buildSpec("djinn-com-1", tier.Scope{Level: tier.Com, Name: "com"})
	wsMount := spec.Mounts[0]
	hasRW := false
	for _, opt := range wsMount.Options {
		if opt == "rw" {
			hasRW = true
		}
	}
	if !hasRW {
		t.Fatalf("Com tier should have rw mount, got %v", wsMount.Options)
	}
}

func TestSandboxPort_BuildSpec_EmptyName(t *testing.T) {
	s := &SandboxPort{workspace: "/workspace"}
	spec := s.buildSpec("djinn-empty-1", tier.Scope{Level: tier.Eco})
	if spec.TierConfig != nil {
		t.Fatal("empty scope name should not set TierConfig")
	}
}

func TestSandboxPort_BuildSpec_TmpfsMount(t *testing.T) {
	s := &SandboxPort{workspace: "/workspace"}
	spec := s.buildSpec("djinn-tmp-1", tier.Scope{Level: tier.Mod, Name: "test"})
	if len(spec.Mounts) < 2 {
		t.Fatal("should have at least 2 mounts")
	}
	tmpMount := spec.Mounts[1]
	if tmpMount.Destination != "/tmp" {
		t.Fatalf("second mount dest = %q, want /tmp", tmpMount.Destination)
	}
	if tmpMount.Type != mountTypeTmpfs {
		t.Fatalf("second mount type = %q, want tmpfs", tmpMount.Type)
	}
}

func TestSandboxPort_New(t *testing.T) {
	s := New("/tmp/test.sock", "/workspace")
	if s == nil {
		t.Fatal("New returned nil")
	}
	s.Close()
}

func TestMountOptionsForTier_Default(t *testing.T) {
	// Unknown tier level should default to ro
	opts := mountOptionsForTier(99)
	hasRO := false
	for _, opt := range opts {
		if opt == "ro" {
			hasRO = true
		}
	}
	if !hasRO {
		t.Fatalf("unknown tier should default to ro, got %v", opts)
	}
}

// Integration test — only runs if Misbah daemon socket exists.
func TestSandboxPort_Integration(t *testing.T) {
	socketPath := os.Getenv("MISBAH_DAEMON_SOCKET")
	if socketPath == "" {
		socketPath = "/run/misbah/permission.sock"
	}

	// Skip if daemon not running
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Skipf("Misbah daemon not available at %s: %v", socketPath, err)
	}
	conn.Close()

	workspace := t.TempDir()
	s := New(socketPath, workspace)
	defer s.Close()

	ctx := context.Background()

	id, err := s.Create(ctx, tier.Scope{Level: tier.Mod, Name: "test"})
	if err != nil {
		if strings.Contains(err.Error(), "not supported") ||
			strings.Contains(err.Error(), "permission denied") ||
			strings.Contains(err.Error(), "network isolation") ||
			strings.Contains(err.Error(), "veth") ||
			strings.Contains(err.Error(), "bind:") {
			t.Skipf("skipping: infrastructure issue: %v", err)
		}
		t.Fatalf("Create: %v", err)
	}
	if id == "" {
		t.Fatal("Create returned empty ID")
	}
	t.Logf("created sandbox: %s", id)

	if err := s.Destroy(ctx, id); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	t.Log("destroyed sandbox")
}
