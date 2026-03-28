//go:build linux

package namespace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/sandbox"
)

func setupWorkspace(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func createSandbox(t *testing.T, workDir string) (*NamespaceSandbox, sandbox.Handle) {
	t.Helper()
	sb := New(workDir)
	handle, err := sb.Create(context.Background(), "", nil)
	if err != nil {
		if errors.Is(err, ErrUnsupported) {
			t.Skip("fuse-overlayfs not available")
		}
		t.Fatal(err)
	}
	t.Cleanup(func() { sb.Destroy(context.Background(), handle) })
	return sb, handle
}

// --- TSK-421: Sandbox interface ---

func TestSandbox_CreateDestroy(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{"a.go": "package a"})
	sb := New(dir)

	handle, err := sb.Create(context.Background(), "", nil)
	if err != nil {
		if errors.Is(err, ErrUnsupported) {
			t.Skip("fuse-overlayfs not available")
		}
		t.Fatal(err)
	}

	if handle == "" {
		t.Fatal("empty handle")
	}
	if sb.Name() != "namespace" {
		t.Fatalf("name = %q, want namespace", sb.Name())
	}

	if err := sb.Destroy(context.Background(), handle); err != nil {
		t.Fatal(err)
	}

	// Double destroy should error.
	if err := sb.Destroy(context.Background(), handle); err == nil {
		t.Fatal("double destroy should error")
	}
}

func TestSandbox_ExecReadFile(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{"hello.txt": "world"})
	sb, handle := createSandbox(t, dir)

	result, err := sb.Exec(context.Background(), handle, []string{"cat", "hello.txt"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", result.ExitCode, result.Stderr)
	}
	if strings.TrimSpace(result.Stdout) != "world" {
		t.Fatalf("stdout = %q, want %q", result.Stdout, "world")
	}
}

func TestSandbox_ExecWriteFile(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{"main.go": "original"})
	sb, handle := createSandbox(t, dir)

	// Write through sandbox.
	result, err := sb.Exec(context.Background(), handle,
		[]string{"bash", "-c", "echo -n modified > main.go"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("write failed: %s", result.Stderr)
	}

	// Real file untouched.
	actual, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if string(actual) != "original" {
		t.Fatalf("real file modified: %q", actual)
	}

	// Read back through sandbox — should see write.
	result, err = sb.Exec(context.Background(), handle, []string{"cat", "main.go"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(result.Stdout) != "modified" {
		t.Fatalf("sandbox read = %q, want modified", result.Stdout)
	}
}

func TestSandbox_ExecCreateFile(t *testing.T) {
	dir := setupWorkspace(t, nil)
	sb, handle := createSandbox(t, dir)

	result, err := sb.Exec(context.Background(), handle,
		[]string{"bash", "-c", "echo -n content > new.go"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("create failed: %s", result.Stderr)
	}

	// Not in real workspace.
	if _, err := os.Stat(filepath.Join(dir, "new.go")); !os.IsNotExist(err) {
		t.Fatal("new file leaked to real workspace")
	}
}

func TestSandbox_ExecBadHandle(t *testing.T) {
	sb := New(t.TempDir())
	_, err := sb.Exec(context.Background(), "bogus", []string{"echo"}, 10)
	if err == nil {
		t.Fatal("should error on bad handle")
	}
}

func TestSandbox_ExecListDir(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{"a.go": "a", "b.go": "b"})
	sb, handle := createSandbox(t, dir)

	result, err := sb.Exec(context.Background(), handle, []string{"ls"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Stdout, "a.go") || !strings.Contains(result.Stdout, "b.go") {
		t.Fatalf("ls output missing files: %q", result.Stdout)
	}
}

func TestSandbox_ExecGrepContent(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{
		"main.go": "package main\nfunc hello() {}\n",
	})
	sb, handle := createSandbox(t, dir)

	result, err := sb.Exec(context.Background(), handle,
		[]string{"grep", "-n", "hello", "main.go"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Stdout, "hello") {
		t.Fatalf("grep didn't find hello: %q", result.Stdout)
	}
}

func TestSandbox_ExecShellCommand(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{"data.txt": "one\ntwo\nthree\n"})
	sb, handle := createSandbox(t, dir)

	result, err := sb.Exec(context.Background(), handle,
		[]string{"wc", "-l", "data.txt"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Stdout, "3") {
		t.Fatalf("wc -l = %q, want 3", result.Stdout)
	}
}

// --- TSK-422: Diff and Commit ---

func TestSandbox_DiffShowsChanges(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{"a.go": "a", "b.go": "b"})
	sb, handle := createSandbox(t, dir)

	// Modify a.go, leave b.go.
	sb.Exec(context.Background(), handle,
		[]string{"bash", "-c", "echo -n changed > a.go"}, 10)

	diff, err := sb.Diff(handle)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range diff {
		if f == "a.go" {
			found = true
		}
		if f == "b.go" {
			t.Fatal("b.go should not be in diff — it wasn't modified")
		}
	}
	if !found {
		t.Fatalf("a.go not in diff: %v", diff)
	}
}

func TestSandbox_DiffShowsNewFiles(t *testing.T) {
	dir := setupWorkspace(t, nil)
	sb, handle := createSandbox(t, dir)

	sb.Exec(context.Background(), handle,
		[]string{"bash", "-c", "echo -n new > created.go"}, 10)

	diff, err := sb.Diff(handle)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff) != 1 || diff[0] != "created.go" {
		t.Fatalf("diff = %v, want [created.go]", diff)
	}
}

func TestSandbox_CommitSelectivePromotion(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{"a.go": "old-a", "b.go": "old-b"})
	sb, handle := createSandbox(t, dir)

	// Modify both.
	sb.Exec(context.Background(), handle,
		[]string{"bash", "-c", "echo -n new-a > a.go && echo -n new-b > b.go"}, 10)

	// Commit only a.go.
	if err := sb.Commit(handle, []string{"a.go"}); err != nil {
		t.Fatal(err)
	}

	// a.go promoted, b.go unchanged.
	gotA, _ := os.ReadFile(filepath.Join(dir, "a.go"))
	if string(gotA) != "new-a" {
		t.Fatalf("a.go = %q, want new-a", gotA)
	}
	gotB, _ := os.ReadFile(filepath.Join(dir, "b.go"))
	if string(gotB) != "old-b" {
		t.Fatalf("b.go = %q, want old-b (unchanged)", gotB)
	}
}

// --- TSK-423: Multi-sandbox isolation ---

func TestSandbox_TwoSandboxesIndependent(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{"main.go": "original"})
	sb := New(dir)

	handleA, err := sb.Create(context.Background(), "", nil)
	if err != nil {
		if errors.Is(err, ErrUnsupported) {
			t.Skip("fuse-overlayfs not available")
		}
		t.Fatal(err)
	}
	defer sb.Destroy(context.Background(), handleA)

	handleB, err := sb.Create(context.Background(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sb.Destroy(context.Background(), handleB)

	// A writes "foo", B writes "bar".
	sb.Exec(context.Background(), handleA,
		[]string{"bash", "-c", "echo -n foo > main.go"}, 10)
	sb.Exec(context.Background(), handleB,
		[]string{"bash", "-c", "echo -n bar > main.go"}, 10)

	// A sees "foo".
	resultA, _ := sb.Exec(context.Background(), handleA, []string{"cat", "main.go"}, 10)
	if strings.TrimSpace(resultA.Stdout) != "foo" {
		t.Fatalf("A sees %q, want foo", resultA.Stdout)
	}

	// B sees "bar".
	resultB, _ := sb.Exec(context.Background(), handleB, []string{"cat", "main.go"}, 10)
	if strings.TrimSpace(resultB.Stdout) != "bar" {
		t.Fatalf("B sees %q, want bar", resultB.Stdout)
	}

	// Real file untouched.
	actual, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if string(actual) != "original" {
		t.Fatalf("real file = %q, want original", actual)
	}
}

func TestSandbox_DestroyOnePreservesOther(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{"x.go": "x"})
	sb := New(dir)

	handleA, err := sb.Create(context.Background(), "", nil)
	if err != nil {
		if errors.Is(err, ErrUnsupported) {
			t.Skip("fuse-overlayfs not available")
		}
		t.Fatal(err)
	}

	handleB, err := sb.Create(context.Background(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sb.Destroy(context.Background(), handleB)

	// Write in both.
	sb.Exec(context.Background(), handleA,
		[]string{"bash", "-c", "echo -n a-data > x.go"}, 10)
	sb.Exec(context.Background(), handleB,
		[]string{"bash", "-c", "echo -n b-data > x.go"}, 10)

	// Destroy A.
	if err := sb.Destroy(context.Background(), handleA); err != nil {
		t.Fatal(err)
	}

	// B still works.
	result, err := sb.Exec(context.Background(), handleB, []string{"cat", "x.go"}, 10)
	if err != nil {
		t.Fatalf("B exec after A destroy: %v", err)
	}
	if strings.TrimSpace(result.Stdout) != "b-data" {
		t.Fatalf("B = %q, want b-data", result.Stdout)
	}
}

// --- TSK-360: E2E Round-Trip — THE Acceptance Test ---

func TestE2E_UniversalSevenRoundTrip(t *testing.T) {
	// Setup: workspace with files.
	dir := setupWorkspace(t, map[string]string{
		"main.go":     "package main\n\nfunc main() {}\n",
		"config.yaml": "key: value\n",
		"README.md":   "# Hello\n",
	})

	sb, handle := createSandbox(t, dir)

	ctx := context.Background()

	// 1. READ — agent reads a file through overlay.
	result, err := sb.Exec(ctx, handle, []string{"cat", "main.go"}, 10)
	if err != nil {
		t.Fatalf("READ: %v", err)
	}
	if !strings.Contains(result.Stdout, "package main") {
		t.Fatalf("READ: got %q, want package main", result.Stdout)
	}

	// 2. WRITE — agent writes a file, real workspace untouched.
	result, err = sb.Exec(ctx, handle,
		[]string{"bash", "-c", "echo -n 'package main\n\nfunc hello() {}' > main.go"}, 10)
	if err != nil {
		t.Fatalf("WRITE: %v", err)
	}
	actual, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if !strings.Contains(string(actual), "func main()") {
		t.Fatalf("WRITE: real file modified: %q", actual)
	}

	// 3. EDIT (via sed) — agent edits a file.
	result, err = sb.Exec(ctx, handle,
		[]string{"sed", "-i", "s/value/replaced/", "config.yaml"}, 10)
	if err != nil {
		t.Fatalf("EDIT: %v", err)
	}

	// 4. SEARCH CONTENT (grep) — agent searches through overlay.
	result, err = sb.Exec(ctx, handle, []string{"grep", "-r", "replaced", "."}, 10)
	if err != nil {
		t.Fatalf("SEARCH: %v", err)
	}
	if !strings.Contains(result.Stdout, "replaced") {
		t.Fatalf("SEARCH: grep didn't find edited content: %q", result.Stdout)
	}

	// 5. FIND FILES (glob via find) — agent lists files.
	result, err = sb.Exec(ctx, handle, []string{"find", ".", "-name", "*.go"}, 10)
	if err != nil {
		t.Fatalf("GLOB: %v", err)
	}
	if !strings.Contains(result.Stdout, "main.go") {
		t.Fatalf("GLOB: didn't find main.go: %q", result.Stdout)
	}

	// 6. LIST DIRECTORY — agent lists directory.
	result, err = sb.Exec(ctx, handle, []string{"ls", "-la"}, 10)
	if err != nil {
		t.Fatalf("LISTDIR: %v", err)
	}
	if !strings.Contains(result.Stdout, "README.md") {
		t.Fatalf("LISTDIR: missing README.md: %q", result.Stdout)
	}

	// 7. SHELL EXECUTE — agent runs arbitrary command.
	result, err = sb.Exec(ctx, handle, []string{"wc", "-l", "main.go"}, 10)
	if err != nil {
		t.Fatalf("EXEC: %v", err)
	}
	// Agent wrote a single-line file, so wc should work.
	if result.ExitCode != 0 {
		t.Fatalf("EXEC: exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// DIFF — Djinn sees what changed.
	diff, err := sb.Diff(handle)
	if err != nil {
		t.Fatalf("DIFF: %v", err)
	}
	if len(diff) == 0 {
		t.Fatal("DIFF: no changes detected — should see main.go and config.yaml")
	}
	diffMap := make(map[string]bool)
	for _, f := range diff {
		diffMap[f] = true
	}
	if !diffMap["main.go"] {
		t.Fatalf("DIFF: missing main.go: %v", diff)
	}
	if !diffMap["config.yaml"] {
		t.Fatalf("DIFF: missing config.yaml: %v", diff)
	}

	// COMMIT — Djinn promotes only config.yaml.
	if err := sb.Commit(handle, []string{"config.yaml"}); err != nil {
		t.Fatalf("COMMIT: %v", err)
	}

	// config.yaml promoted.
	committed, _ := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if !strings.Contains(string(committed), "replaced") {
		t.Fatalf("COMMIT: config.yaml not promoted: %q", committed)
	}

	// main.go NOT promoted — still original.
	notCommitted, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if !strings.Contains(string(notCommitted), "func main()") {
		t.Fatalf("COMMIT: main.go was promoted but shouldn't be: %q", notCommitted)
	}

	t.Log("E2E PASS: All Universal 7 operations captured through overlay. Agent accepted all results as normal.")
}

// --- Registry ---

func TestSandbox_RegisteredInRegistry(t *testing.T) {
	sb, err := sandbox.Get("namespace")
	if err != nil {
		t.Fatal(err)
	}
	if sb.Name() != "namespace" {
		t.Fatalf("name = %q", sb.Name())
	}
}
