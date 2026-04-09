package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDumpWorkspace_CapturesTextFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0o644)

	artifacts := DumpWorkspace(dir)

	if len(artifacts) != 2 {
		t.Fatalf("artifacts = %d, want 2", len(artifacts))
	}
	if artifacts["main.go"] != "package main" {
		t.Fatalf("main.go = %q", artifacts["main.go"])
	}
	if artifacts["go.mod"] != "module test" {
		t.Fatalf("go.mod = %q", artifacts["go.mod"])
	}
}

func TestDumpWorkspace_SkipsBinaries(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("code"), 0o644)
	os.WriteFile(filepath.Join(dir, "server"), []byte("binary"), 0o755)
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("secret"), 0o644)
	os.WriteFile(filepath.Join(dir, "lib.so"), []byte("shared"), 0o644)

	artifacts := DumpWorkspace(dir)

	if _, ok := artifacts["server"]; ok {
		t.Fatal("should skip 'server' binary")
	}
	if _, ok := artifacts[".hidden"]; ok {
		t.Fatal("should skip dotfiles")
	}
	if _, ok := artifacts["lib.so"]; ok {
		t.Fatal("should skip .so files")
	}
	if artifacts["main.go"] != "code" {
		t.Fatal("should keep main.go")
	}
}

func TestDumpWorkspace_SkipsLargeFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "small.txt"), []byte("ok"), 0o644)
	os.WriteFile(filepath.Join(dir, "huge.txt"), []byte(strings.Repeat("x", MaxDumpFileSize+1)), 0o644)

	artifacts := DumpWorkspace(dir)

	if _, ok := artifacts["small.txt"]; !ok {
		t.Fatal("should keep small.txt")
	}
	if _, ok := artifacts["huge.txt"]; ok {
		t.Fatal("should skip files over MaxDumpFileSize")
	}
}

func TestDumpWorkspace_Subdirectories(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "pkg"), 0o755)
	os.WriteFile(filepath.Join(dir, "pkg", "lib.go"), []byte("package pkg"), 0o644)

	artifacts := DumpWorkspace(dir)

	if artifacts[filepath.Join("pkg", "lib.go")] != "package pkg" {
		t.Fatalf("pkg/lib.go = %q", artifacts[filepath.Join("pkg", "lib.go")])
	}
}

func TestDumpWorkspace_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	artifacts := DumpWorkspace(dir)
	if len(artifacts) != 0 {
		t.Fatalf("artifacts = %d, want 0", len(artifacts))
	}
}
