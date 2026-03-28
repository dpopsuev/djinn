package staff

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGatherIntelligence_EmptyWorkDir(t *testing.T) {
	b := GatherIntelligence(context.Background(), "")
	if b == nil {
		t.Fatal("expected non-nil briefing")
	}
	if len(b.Recommendations) == 0 {
		t.Fatal("expected recommendation for empty workdir")
	}
	found := false
	for _, r := range b.Recommendations {
		if r == "no working directory specified" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'no working directory specified' in recommendations, got %v", b.Recommendations)
	}
}

func TestGatherIntelligence_WithGoFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a package with a Go file and a test file.
	pkg := filepath.Join(dir, "mypackage")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "main.go"), []byte("package mypackage\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "main_test.go"), []byte("package mypackage\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := GatherIntelligence(context.Background(), dir)
	if len(b.ExistingPackages) == 0 {
		t.Fatal("expected at least one package")
	}
	if b.FailingTests == 0 {
		t.Fatal("expected test file count > 0")
	}
}

func TestGatherIntelligence_NoTestFiles(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := GatherIntelligence(context.Background(), dir)
	if b.FailingTests != 0 {
		t.Fatalf("expected 0 test files, got %d", b.FailingTests)
	}
	// Should recommend adding tests.
	found := false
	for _, r := range b.Recommendations {
		if r == "no test files detected — add tests" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'no test files' recommendation, got %v", b.Recommendations)
	}
}

func TestGatherIntelligence_SkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()

	hidden := filepath.Join(dir, ".git")
	if err := os.MkdirAll(hidden, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hidden, "internal.go"), []byte("package git\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := GatherIntelligence(context.Background(), dir)
	for _, pkg := range b.ExistingPackages {
		if pkg == ".git" {
			t.Fatal("should skip hidden directories")
		}
	}
}

func TestGatherIntelligence_SkipsVendor(t *testing.T) {
	dir := t.TempDir()

	vendor := filepath.Join(dir, "vendor", "dep")
	if err := os.MkdirAll(vendor, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendor, "lib.go"), []byte("package dep\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := GatherIntelligence(context.Background(), dir)
	for _, pkg := range b.ExistingPackages {
		if pkg == "vendor" || pkg == "vendor/dep" {
			t.Fatalf("should skip vendor directory, found %q", pkg)
		}
	}
}

func TestGatherIntelligence_CanceledContext(t *testing.T) {
	dir := t.TempDir()

	// Create some files.
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	b := GatherIntelligence(ctx, dir)
	// Should still return a valid briefing (not panic).
	if b == nil {
		t.Fatal("expected non-nil briefing even with canceled context")
	}
}

func TestGatherIntelligence_ExtendRecommendation(t *testing.T) {
	dir := t.TempDir()

	pkg := filepath.Join(dir, "auth")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "auth.go"), []byte("package auth\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "auth_test.go"), []byte("package auth\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := GatherIntelligence(context.Background(), dir)
	found := false
	for _, r := range b.Recommendations {
		if r == "existing packages found — extend, don't rewrite" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'extend' recommendation, got %v", b.Recommendations)
	}
}
