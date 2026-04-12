package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMCPManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.yaml")

	yaml := `mcp_servers:
  scribe:
    command: scribe serve
    auto_connect: true
    tools:
      artifact:
        requires: [read, work]
      graph:
        requires: [coordinate]
  locus:
    command: locus serve
    auto_connect: true
  lex:
    url: http://localhost:8080/
    auto_connect: false
`
	os.WriteFile(path, []byte(yaml), 0o600) //nolint:errcheck // test helper

	cfg, err := LoadMCPManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Servers) != 3 {
		t.Fatalf("servers = %d, want 3", len(cfg.Servers))
	}

	scribe := cfg.Servers["scribe"]
	if scribe.Command != "scribe serve" {
		t.Fatalf("scribe command = %q", scribe.Command)
	}
	if !scribe.AutoConnect {
		t.Fatal("scribe should auto_connect")
	}
	if len(scribe.Tools) != 2 {
		t.Fatalf("scribe tools = %d, want 2", len(scribe.Tools))
	}
	if len(scribe.Tools["artifact"].Requires) != 2 {
		t.Fatalf("artifact requires = %d, want 2", len(scribe.Tools["artifact"].Requires))
	}

	lex := cfg.Servers["lex"]
	if lex.URL != "http://localhost:8080/" {
		t.Fatalf("lex url = %q", lex.URL)
	}
	if lex.AutoConnect {
		t.Fatal("lex should NOT auto_connect")
	}
}

func TestLoadMCPSecrets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.yaml")

	yaml := `servers:
  scribe:
    env:
      SCRIBE_TOKEN: "secret-123"
      SCRIBE_DB: "/path/to/db"
  emcee:
    args: ["--api-key", "key-456"]
`
	os.WriteFile(path, []byte(yaml), 0o600) //nolint:errcheck // test helper

	secrets, err := LoadMCPSecrets(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(secrets.Servers) != 2 {
		t.Fatalf("servers = %d, want 2", len(secrets.Servers))
	}
	if secrets.Servers["scribe"].Env["SCRIBE_TOKEN"] != "secret-123" {
		t.Fatal("scribe token mismatch")
	}
	if len(secrets.Servers["emcee"].Args) != 2 {
		t.Fatalf("emcee args = %d, want 2", len(secrets.Servers["emcee"].Args))
	}
}

func TestMergeManifests(t *testing.T) {
	remote := &MCPManifest{Servers: map[string]ServerSpec{
		"scribe": {Command: "scribe serve", AutoConnect: true},
		"locus":  {Command: "locus serve", AutoConnect: true},
	}}

	project := &MCPManifest{Servers: map[string]ServerSpec{
		"scribe": {Command: "scribe serve --port 9090", AutoConnect: true},
		"emcee":  {Command: "emcee serve", AutoConnect: false},
	}}

	merged := MergeManifests(remote, project)

	if len(merged.Servers) != 3 {
		t.Fatalf("merged servers = %d, want 3 (scribe, locus, emcee)", len(merged.Servers))
	}

	// Project overrides remote for scribe.
	if merged.Servers["scribe"].Command != "scribe serve --port 9090" {
		t.Fatalf("scribe command = %q, want project override", merged.Servers["scribe"].Command)
	}

	// Remote locus preserved.
	if merged.Servers["locus"].Command != "locus serve" {
		t.Fatal("locus should come from remote")
	}

	// Project emcee added.
	if merged.Servers["emcee"].Command != "emcee serve" {
		t.Fatal("emcee should come from project")
	}
}

func TestApplySecrets(t *testing.T) {
	cfg := &MCPManifest{Servers: map[string]ServerSpec{
		"scribe": {Command: "scribe serve", Env: map[string]string{"EXISTING": "val"}},
	}}

	secrets := &MCPSecrets{Servers: map[string]SecretSpec{
		"scribe": {Env: map[string]string{"SCRIBE_TOKEN": "secret"}},
	}}

	ApplySecrets(cfg, secrets)

	scribe := cfg.Servers["scribe"]
	if scribe.Env["SCRIBE_TOKEN"] != "secret" {
		t.Fatal("secret env not applied")
	}
	if scribe.Env["EXISTING"] != "val" {
		t.Fatal("existing env should be preserved")
	}
}

func TestApplySecrets_UnknownServer(t *testing.T) {
	cfg := &MCPManifest{Servers: map[string]ServerSpec{
		"scribe": {Command: "scribe serve"},
	}}

	secrets := &MCPSecrets{Servers: map[string]SecretSpec{
		"unknown": {Env: map[string]string{"KEY": "val"}},
	}}

	ApplySecrets(cfg, secrets)

	// Should not panic, unknown server ignored.
	if len(cfg.Servers) != 1 {
		t.Fatal("should not add unknown server")
	}
}

func TestSaveMCPManifest_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.yaml")

	original := &MCPManifest{Servers: map[string]ServerSpec{
		"scribe": {Command: "scribe serve", AutoConnect: true},
	}}

	if err := SaveMCPManifest(path, original); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadMCPManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Servers["scribe"].Command != "scribe serve" {
		t.Fatal("roundtrip failed")
	}
}

func TestMergeManifests_NilSafe(t *testing.T) {
	result := MergeManifests(nil, nil)
	if len(result.Servers) != 0 {
		t.Fatal("nil merge should return empty")
	}
}
