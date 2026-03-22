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
