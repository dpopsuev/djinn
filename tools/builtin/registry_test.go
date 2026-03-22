package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistry_AllBuiltinsRegistered(t *testing.T) {
	r := NewRegistry()
	expected := []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"}
	for _, name := range expected {
		if _, err := r.Get(name); err != nil {
			t.Fatalf("tool %q not registered: %v", name, err)
		}
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("NonExistent")
	if !errors.Is(err, ErrToolNotFound) {
		t.Fatalf("err = %v, want ErrToolNotFound", err)
	}
}

func TestRegistry_Execute(t *testing.T) {
	r := NewRegistry()

	// Create a temp file to read
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello\nworld\n"), 0644)

	input, _ := json.Marshal(readInput{Path: path})
	output, err := r.Execute(context.Background(), "Read", input)
	if err != nil {
		t.Fatalf("Execute Read: %v", err)
	}
	if !strings.Contains(output, "hello") {
		t.Fatalf("output = %q, expected to contain 'hello'", output)
	}
}

func TestReadTool(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "code.go")
	os.WriteFile(path, []byte("package main\n\nfunc main() {}\n"), 0644)

	tool := &ReadTool{}
	input, _ := json.Marshal(readInput{Path: path})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !strings.Contains(out, "package main") {
		t.Fatalf("output missing content: %q", out)
	}
	// Should have line numbers
	if !strings.Contains(out, "1\t") {
		t.Fatalf("output missing line numbers: %q", out)
	}
}

func TestReadTool_WithOffset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lines.txt")
	os.WriteFile(path, []byte("line1\nline2\nline3\nline4\n"), 0644)

	tool := &ReadTool{}
	input, _ := json.Marshal(readInput{Path: path, Offset: 2, Limit: 2})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !strings.Contains(out, "line2") || !strings.Contains(out, "line3") {
		t.Fatalf("offset/limit not working: %q", out)
	}
	if strings.Contains(out, "line1") || strings.Contains(out, "line4") {
		t.Fatalf("offset/limit leaked extra lines: %q", out)
	}
}

func TestWriteTool(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.txt")

	tool := &WriteTool{}
	input, _ := json.Marshal(writeInput{Path: path, Content: "created"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !strings.Contains(out, "wrote") {
		t.Fatalf("output = %q", out)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "created" {
		t.Fatalf("file content = %q", string(data))
	}
}

func TestEditTool(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "edit.go")
	os.WriteFile(path, []byte("func old() {}\n"), 0644)

	tool := &EditTool{}
	input, _ := json.Marshal(editInput{Path: path, OldString: "old", NewString: "new"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if !strings.Contains(out, "replaced") {
		t.Fatalf("output = %q", out)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "func new()") {
		t.Fatalf("edit not applied: %q", string(data))
	}
}

func TestEditTool_NoMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nope.go")
	os.WriteFile(path, []byte("func main() {}\n"), 0644)

	tool := &EditTool{}
	input, _ := json.Marshal(editInput{Path: path, OldString: "nothere", NewString: "x"})
	_, err := tool.Execute(context.Background(), input)
	if !errors.Is(err, ErrNoMatch) {
		t.Fatalf("err = %v, want ErrNoMatch", err)
	}
}

func TestBashTool(t *testing.T) {
	tool := &BashTool{}
	input, _ := json.Marshal(bashInput{Command: "echo hello"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Bash: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("output = %q", out)
	}
}

func TestBashTool_ExitCode(t *testing.T) {
	tool := &BashTool{}
	input, _ := json.Marshal(bashInput{Command: "exit 42"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Bash: %v", err)
	}
	if !strings.Contains(out, "EXIT CODE: 42") {
		t.Fatalf("output = %q, expected exit code 42", out)
	}
}

func TestGlobTool(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), nil, 0644)
	os.WriteFile(filepath.Join(dir, "b.go"), nil, 0644)
	os.WriteFile(filepath.Join(dir, "c.txt"), nil, 0644)

	tool := &GlobTool{}
	input, _ := json.Marshal(globInput{Pattern: "*.go", Path: dir})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if !strings.Contains(out, "a.go") || !strings.Contains(out, "b.go") {
		t.Fatalf("output = %q, expected .go files", out)
	}
	if strings.Contains(out, "c.txt") {
		t.Fatalf("output should not contain .txt: %q", out)
	}
}

func TestGrepTool(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "search.go")
	os.WriteFile(path, []byte("package main\n\nfunc hello() {}\nfunc goodbye() {}\n"), 0644)

	tool := &GrepTool{}
	input, _ := json.Marshal(grepInput{Pattern: "func.*\\(\\)", Path: path})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if !strings.Contains(out, "hello") || !strings.Contains(out, "goodbye") {
		t.Fatalf("output = %q", out)
	}
}

func TestGrepTool_NoMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	os.WriteFile(path, []byte("nothing here\n"), 0644)

	tool := &GrepTool{}
	input, _ := json.Marshal(grepInput{Pattern: "missing", Path: path})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if out != "no matches found" {
		t.Fatalf("output = %q", out)
	}
}
