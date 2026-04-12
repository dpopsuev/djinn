// workdir_contract_test.go — RED: contract tests proving all file tools
// resolve relative paths against WorkDir. MUST FAIL until GREEN task.
//
// DJN-TSK-1030
package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestWriteTool_WorkDir_ResolvesRelativePath proves Write resolves
// relative paths against WorkDir, not process CWD.
func TestWriteTool_WorkDir_ResolvesRelativePath(t *testing.T) {
	workDir := t.TempDir()
	tool := &WriteTool{WorkDir: workDir}

	input, _ := json.Marshal(writeInput{Path: "hello.go", Content: "package main\n"})
	_, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// File must be in workDir.
	if _, err := os.Stat(filepath.Join(workDir, "hello.go")); os.IsNotExist(err) {
		t.Fatal("hello.go not found in WorkDir — relative path not resolved")
	}

	// File must NOT be in CWD.
	cwd, _ := os.Getwd()
	cwdPath := filepath.Join(cwd, "hello.go")
	if _, err := os.Stat(cwdPath); err == nil {
		os.Remove(cwdPath)
		t.Fatal("hello.go found in CWD — WorkDir not honored")
	}
}

// TestWriteTool_NoWorkDir_BackwardCompat proves Write without WorkDir
// behaves exactly as before (relative to CWD).
func TestWriteTool_NoWorkDir_BackwardCompat(t *testing.T) {
	tool := &WriteTool{} // no WorkDir set

	tmpDir := t.TempDir()
	absPath := filepath.Join(tmpDir, "compat.go")
	input, _ := json.Marshal(writeInput{Path: absPath, Content: "package compat\n"})
	_, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Fatal("absolute path should work without WorkDir")
	}
}

// TestReadTool_WorkDir_ResolvesRelativePath proves Read resolves
// relative paths against WorkDir.
func TestReadTool_WorkDir_ResolvesRelativePath(t *testing.T) {
	workDir := t.TempDir()
	os.WriteFile(filepath.Join(workDir, "config.go"), []byte("package config\n"), 0644)

	tool := &ReadTool{WorkDir: workDir}
	input, _ := json.Marshal(readInput{Path: "config.go"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if out == "" {
		t.Fatal("Read returned empty — WorkDir not resolved")
	}
}

// TestEditTool_WorkDir_ResolvesRelativePath proves Edit resolves
// relative paths against WorkDir.
func TestEditTool_WorkDir_ResolvesRelativePath(t *testing.T) {
	workDir := t.TempDir()
	os.WriteFile(filepath.Join(workDir, "version.go"), []byte("var v = \"0.0.0\"\n"), 0644)

	tool := &EditTool{WorkDir: workDir}
	input, _ := json.Marshal(editInput{
		Path:      "version.go",
		OldString: "0.0.0",
		NewString: "1.0.0",
	})
	_, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(workDir, "version.go"))
	if got := string(data); got != "var v = \"1.0.0\"\n" {
		t.Fatalf("Edit result = %q, want version 1.0.0", got)
	}
}

// TestBashTool_WorkDir_SetsCmdDir proves Bash runs commands
// with cmd.Dir set to WorkDir.
func TestBashTool_WorkDir_SetsCmdDir(t *testing.T) {
	workDir := t.TempDir()
	tool := &BashTool{WorkDir: workDir}

	input, _ := json.Marshal(bashInput{Command: "pwd"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Bash: %v", err)
	}

	// pwd output should be the workDir, not the test's CWD.
	got := filepath.Clean(out[:len(out)-1]) // trim newline
	want := filepath.Clean(workDir)
	if got != want {
		t.Fatalf("Bash pwd = %q, want %q", got, want)
	}
}

// TestGlobTool_WorkDir_ResolvesPattern proves Glob resolves patterns
// relative to WorkDir.
func TestGlobTool_WorkDir_ResolvesPattern(t *testing.T) {
	workDir := t.TempDir()
	os.WriteFile(filepath.Join(workDir, "main.go"), []byte("package main\n"), 0644)
	os.WriteFile(filepath.Join(workDir, "util.go"), []byte("package main\n"), 0644)

	tool := &GlobTool{WorkDir: workDir}
	input, _ := json.Marshal(globInput{Pattern: "*.go"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}

	if out == "no files found" {
		t.Fatal("Glob found no files — WorkDir not resolved")
	}
}

// TestGrepTool_WorkDir_ResolvesPath proves Grep resolves the target
// file path against WorkDir.
func TestGrepTool_WorkDir_ResolvesPath(t *testing.T) {
	workDir := t.TempDir()
	os.WriteFile(filepath.Join(workDir, "search.go"), []byte("func hello() {}\nfunc world() {}\n"), 0644)

	tool := &GrepTool{WorkDir: workDir}
	input, _ := json.Marshal(grepInput{Pattern: "hello", Path: "search.go"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}

	if out == "no matches found" {
		t.Fatal("Grep found no matches — WorkDir not resolved")
	}
}
