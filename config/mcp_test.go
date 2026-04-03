package config

import (
	"testing"
)

func TestMCPConfigurable_Apply(t *testing.T) {
	c := &MCPConfigurable{}
	err := c.Apply(map[string]any{
		"scribe": map[string]any{
			"command": "scribe",
			"args":    []any{"serve"},
		},
		"locus": map[string]any{
			"url": "http://localhost:8090/",
			"env": map[string]any{
				"LOCUS_DEBUG": "true",
			},
		},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(c.Servers) != 2 {
		t.Fatalf("servers = %d, want 2", len(c.Servers))
	}

	scribe := c.Servers["scribe"]
	if scribe.Command != "scribe" {
		t.Fatalf("scribe.Command = %q", scribe.Command)
	}
	if len(scribe.Args) != 1 || scribe.Args[0] != "serve" {
		t.Fatalf("scribe.Args = %v", scribe.Args)
	}

	locus := c.Servers["locus"]
	if locus.URL != "http://localhost:8090/" {
		t.Fatalf("locus.URL = %q", locus.URL)
	}
	if locus.Env["LOCUS_DEBUG"] != "true" {
		t.Fatalf("locus.Env = %v", locus.Env)
	}
}

func TestMCPConfigurable_Apply_WrongType(t *testing.T) {
	c := &MCPConfigurable{}
	err := c.Apply("not a map")
	if err == nil {
		t.Fatal("should reject non-map")
	}
}

func TestMCPConfigurable_Apply_WrongEntryType(t *testing.T) {
	c := &MCPConfigurable{}
	err := c.Apply(map[string]any{
		"bad": "not a map entry",
	})
	if err == nil {
		t.Fatal("should reject non-map entry")
	}
}

func TestMCPConfigurable_Snapshot(t *testing.T) {
	c := &MCPConfigurable{
		Servers: map[string]MCPServerEntry{
			"scribe": {
				Command: "scribe",
				Args:    []string{"serve"},
			},
			"locus": {
				URL: "http://localhost:8090/",
				Env: map[string]string{"KEY": "val"},
			},
		},
	}
	snap := c.Snapshot()
	m, ok := snap.(map[string]any)
	if !ok {
		t.Fatalf("snapshot type = %T", snap)
	}
	if len(m) != 2 {
		t.Fatalf("snapshot entries = %d", len(m))
	}

	scribe, ok := m["scribe"].(map[string]any)
	if !ok {
		t.Fatalf("scribe type = %T", m["scribe"])
	}
	if scribe["command"] != "scribe" {
		t.Fatalf("scribe.command = %v", scribe["command"])
	}

	locus, ok := m["locus"].(map[string]any)
	if !ok {
		t.Fatalf("locus type = %T", m["locus"])
	}
	if locus["url"] != "http://localhost:8090/" {
		t.Fatalf("locus.url = %v", locus["url"])
	}
}

func TestMCPConfigurable_Snapshot_Nil(t *testing.T) {
	c := &MCPConfigurable{}
	snap := c.Snapshot()
	m, ok := snap.(map[string]any)
	if !ok {
		t.Fatalf("snapshot type = %T", snap)
	}
	if len(m) != 0 {
		t.Fatalf("snapshot entries = %d, want 0", len(m))
	}
}

func TestMCPConfigurable_ConfigKey(t *testing.T) {
	c := &MCPConfigurable{}
	if c.ConfigKey() != "mcp_servers" {
		t.Fatalf("key = %q", c.ConfigKey())
	}
}

func TestMCPConfigurable_Roundtrip(t *testing.T) {
	orig := &MCPConfigurable{
		Servers: map[string]MCPServerEntry{
			"test": {
				Command: "test-server",
				Args:    []string{"-v", "serve"},
				Env:     map[string]string{"PORT": "8080"},
			},
		},
	}

	// Snapshot and re-apply to a fresh instance.
	c2 := &MCPConfigurable{}
	if err := c2.Apply(orig.Snapshot()); err != nil {
		t.Fatalf("roundtrip Apply: %v", err)
	}
	if len(c2.Servers) != 1 {
		t.Fatalf("servers = %d", len(c2.Servers))
	}
	entry := c2.Servers["test"]
	if entry.Command != "test-server" {
		t.Fatalf("command = %q", entry.Command)
	}
	if len(entry.Args) != 2 || entry.Args[0] != "-v" {
		t.Fatalf("args = %v", entry.Args)
	}
	if entry.Env["PORT"] != "8080" {
		t.Fatalf("env = %v", entry.Env)
	}
}

func TestMCPConfigurable_ApplyMerges(t *testing.T) {
	c := &MCPConfigurable{}
	// First apply.
	if err := c.Apply(map[string]any{
		"server1": map[string]any{"command": "s1"},
	}); err != nil {
		t.Fatal(err)
	}
	// Second apply should merge, not replace.
	if err := c.Apply(map[string]any{
		"server2": map[string]any{"command": "s2"},
	}); err != nil {
		t.Fatal(err)
	}
	if len(c.Servers) != 2 {
		t.Fatalf("servers = %d, want 2 (merge)", len(c.Servers))
	}
}
