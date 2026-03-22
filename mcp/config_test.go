package mcp

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateConfig_HTTPServer(t *testing.T) {
	servers := []Server{
		{Name: "scribe", Type: TypeHTTP, URL: "http://localhost:8080/"},
	}

	data, err := GenerateConfig(servers)
	if err != nil {
		t.Fatalf("GenerateConfig: %v", err)
	}

	var cfg claudeConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	entry, ok := cfg.MCPServers["scribe"]
	if !ok {
		t.Fatal("scribe server not in config")
	}
	if entry.URL != "http://localhost:8080/" {
		t.Fatalf("URL = %q, want %q", entry.URL, "http://localhost:8080/")
	}
	if entry.Command != "" {
		t.Fatalf("Command should be empty for HTTP, got %q", entry.Command)
	}
}

func TestGenerateConfig_StdioServer(t *testing.T) {
	servers := []Server{
		{Name: "lex", Type: TypeStdio, Command: "lex", Args: []string{"serve"}},
	}

	data, err := GenerateConfig(servers)
	if err != nil {
		t.Fatalf("GenerateConfig: %v", err)
	}

	var cfg claudeConfigFile
	json.Unmarshal(data, &cfg)

	entry := cfg.MCPServers["lex"]
	if entry.Command != "lex" {
		t.Fatalf("Command = %q, want %q", entry.Command, "lex")
	}
	if len(entry.Args) != 1 || entry.Args[0] != "serve" {
		t.Fatalf("Args = %v, want [serve]", entry.Args)
	}
	if entry.URL != "" {
		t.Fatalf("URL should be empty for stdio, got %q", entry.URL)
	}
}

func TestGenerateConfig_StdioWithEnvAndCWD(t *testing.T) {
	servers := []Server{
		{
			Name:    "locus",
			Type:    TypeStdio,
			Command: "podman",
			Args:    []string{"run", "--rm", "-i", "locus:latest", "serve"},
			Env:     map[string]string{"DATA_DIR": "/data"},
			CWD:     "/workspace",
		},
	}

	data, err := GenerateConfig(servers)
	if err != nil {
		t.Fatalf("GenerateConfig: %v", err)
	}

	var cfg claudeConfigFile
	json.Unmarshal(data, &cfg)

	entry := cfg.MCPServers["locus"]
	if entry.Env["DATA_DIR"] != "/data" {
		t.Fatalf("Env[DATA_DIR] = %q", entry.Env["DATA_DIR"])
	}
	if entry.CWD != "/workspace" {
		t.Fatalf("CWD = %q", entry.CWD)
	}
}

func TestGenerateConfig_MultipleServers(t *testing.T) {
	servers := DefaultServers()

	data, err := GenerateConfig(servers)
	if err != nil {
		t.Fatalf("GenerateConfig: %v", err)
	}

	var cfg claudeConfigFile
	json.Unmarshal(data, &cfg)

	if len(cfg.MCPServers) != len(servers) {
		t.Fatalf("server count = %d, want %d", len(cfg.MCPServers), len(servers))
	}
}

func TestGenerateConfig_NoServers(t *testing.T) {
	_, err := GenerateConfig(nil)
	if !errors.Is(err, ErrNoServers) {
		t.Fatalf("err = %v, want ErrNoServers", err)
	}
}

func TestGenerateConfig_UnknownType(t *testing.T) {
	servers := []Server{{Name: "bad", Type: "grpc"}}
	_, err := GenerateConfig(servers)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestWriteConfigFile(t *testing.T) {
	dir := t.TempDir()
	servers := []Server{
		{Name: "scribe", Type: TypeHTTP, URL: "http://localhost:8080/"},
	}

	path, err := WriteConfigFile(dir, servers)
	if err != nil {
		t.Fatalf("WriteConfigFile: %v", err)
	}

	expected := filepath.Join(dir, configFileName)
	if path != expected {
		t.Fatalf("path = %q, want %q", path, expected)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var cfg claudeConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal written file: %v", err)
	}

	if _, ok := cfg.MCPServers["scribe"]; !ok {
		t.Fatal("scribe not in written config")
	}
}

func TestDefaultServers(t *testing.T) {
	servers := DefaultServers()
	if len(servers) == 0 {
		t.Fatal("DefaultServers returned empty")
	}

	names := make(map[string]bool)
	for _, s := range servers {
		if s.Name == "" {
			t.Fatal("server with empty name")
		}
		if s.Type != TypeHTTP && s.Type != TypeStdio {
			t.Fatalf("server %q has unknown type %q", s.Name, s.Type)
		}
		names[s.Name] = true
	}

	if !names["scribe"] {
		t.Fatal("scribe missing from defaults")
	}
}
