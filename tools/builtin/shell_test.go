package builtin

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/artifact"
	"github.com/dpopsuev/djinn/tools"
)

func TestRegisterAeonShellTools(t *testing.T) {
	reg := NewRegistry()
	dir := t.TempDir()
	RegisterAeonShellTools(reg, dir, dir)

	expected := []string{"plan", "test", "git", "arch", "discourse", "reconcile", "latency", "render"}
	for _, name := range expected {
		if _, err := reg.Get(name); err != nil {
			t.Fatalf("tool %q not registered: %v", name, err)
		}
	}

	// Original 6 + 8 shell = 14 total.
	if len(reg.Names()) != 14 {
		t.Fatalf("total tools = %d, want 14", len(reg.Names()))
	}
}

func TestPlanTool_CreateGetRoundtrip(t *testing.T) {
	store := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	tool := &PlanTool{Store: store}
	ctx := context.Background()

	// Create.
	input, _ := json.Marshal(map[string]string{"action": "create", "title": "test task"})
	out, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.Contains(out, "test task") {
		t.Fatalf("create output = %q", out)
	}

	// Extract ID.
	var created artifact.Artifact
	json.Unmarshal([]byte(out), &created)

	// Get.
	input, _ = json.Marshal(map[string]string{"action": "get", "id": created.ID})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out, created.ID) {
		t.Fatalf("get output = %q", out)
	}

	// Update.
	input, _ = json.Marshal(map[string]string{"action": "update", "id": created.ID, "status": "done"})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(out, "done") {
		t.Fatalf("update output = %q", out)
	}

	// List.
	input, _ = json.Marshal(map[string]string{"action": "list"})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, created.ID) {
		t.Fatalf("list output = %q", out)
	}

	// Topo sort.
	input, _ = json.Marshal(map[string]string{"action": "topo_sort"})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("topo_sort: %v", err)
	}
	if !strings.Contains(out, created.ID) {
		t.Fatalf("topo_sort output = %q", out)
	}
}

func TestPlanTool_InvalidAction(t *testing.T) {
	store := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	tool := &PlanTool{Store: store}

	input, _ := json.Marshal(map[string]string{"action": "nope"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
}

func TestTestTool_ParseCannedOutput(t *testing.T) {
	tool := &TestTool{WorkDir: t.TempDir()}
	ctx := context.Background()

	canned := `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/foo","Test":"TestOK"}
{"Time":"2024-01-01T00:00:01Z","Action":"output","Package":"example.com/foo","Test":"TestOK","Output":"--- PASS: TestOK\n"}
{"Time":"2024-01-01T00:00:01Z","Action":"pass","Package":"example.com/foo","Test":"TestOK","Elapsed":0.5}
{"Time":"2024-01-01T00:00:01Z","Action":"output","Package":"example.com/foo","Output":"coverage: 75.3% of statements\n"}
{"Time":"2024-01-01T00:00:01Z","Action":"pass","Package":"example.com/foo","Elapsed":0.5}`

	input, _ := json.Marshal(map[string]string{"action": "parse", "data": canned})
	out, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	var result tools.TestResult
	json.Unmarshal([]byte(out), &result)
	if result.Passed != 1 {
		t.Fatalf("passed = %d, want 1", result.Passed)
	}
	if result.Coverage != 75.3 {
		t.Fatalf("coverage = %f, want 75.3", result.Coverage)
	}
}

func TestTestTool_InvalidAction(t *testing.T) {
	tool := &TestTool{WorkDir: t.TempDir()}
	input, _ := json.Marshal(map[string]string{"action": "nope"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
}

func TestGitTool_StatusOnTempRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	dir := t.TempDir()
	// Init a git repo.
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s %v", args, out, err)
		}
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\n"), 0644)
	run("add", ".")
	run("commit", "-m", "init")

	repo := tools.NewGitRepo(dir)
	tool := &GitTool{Repo: repo}
	ctx := context.Background()

	// Status.
	input, _ := json.Marshal(map[string]string{"action": "status"})
	out, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	var status tools.GitStatus
	json.Unmarshal([]byte(out), &status)
	if !status.Clean {
		t.Fatalf("repo should be clean, got %+v", status)
	}

	// Branch.
	input, _ = json.Marshal(map[string]string{"action": "branch"})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("branch: %v", err)
	}
	if out != "main" && out != "master" {
		t.Fatalf("branch = %q", out)
	}

	// Log.
	input, _ = json.Marshal(map[string]any{"action": "log", "n": 1})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	if !strings.Contains(out, "init") {
		t.Fatalf("log output = %q", out)
	}

	// Diff (should be empty).
	input, _ = json.Marshal(map[string]string{"action": "diff"})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if out != "" {
		t.Fatalf("diff should be empty, got %q", out)
	}
}

func TestGitTool_InvalidAction(t *testing.T) {
	repo := tools.NewGitRepo(t.TempDir())
	tool := &GitTool{Repo: repo}
	input, _ := json.Marshal(map[string]string{"action": "nope"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
}

func TestDiscourseTool_TopicThreadRoundtrip(t *testing.T) {
	store := tools.NewDiscourseStore(filepath.Join(t.TempDir(), "discourse.json"))
	tool := &DiscourseTool{Store: store}
	ctx := context.Background()

	// Create topic.
	input, _ := json.Marshal(map[string]string{
		"action": "create_topic",
		"scope":  "/test",
		"title":  "Auth Module",
		"kind":   "feature",
	})
	out, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("create_topic: %v", err)
	}

	var topic tools.Topic
	json.Unmarshal([]byte(out), &topic)
	if topic.Title != "Auth Module" {
		t.Fatalf("topic title = %q", topic.Title)
	}

	// Get topic.
	input, _ = json.Marshal(map[string]string{
		"action":   "get_topic",
		"scope":    "/test",
		"topic_id": topic.ID,
	})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("get_topic: %v", err)
	}
	if !strings.Contains(out, "Auth Module") {
		t.Fatalf("get_topic output = %q", out)
	}

	// Create thread.
	input, _ = json.Marshal(map[string]string{
		"action":   "create_thread",
		"scope":    "/test",
		"topic_id": topic.ID,
	})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("create_thread: %v", err)
	}

	var thread tools.Thread
	json.Unmarshal([]byte(out), &thread)

	// Append message.
	input, _ = json.Marshal(map[string]string{
		"action":    "append",
		"scope":     "/test",
		"topic_id":  topic.ID,
		"thread_id": thread.ID,
		"role":      "user",
		"content":   "hello",
	})
	_, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	// Open count.
	input, _ = json.Marshal(map[string]string{
		"action": "open_count",
		"scope":  "/test",
	})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("open_count: %v", err)
	}
	if out != "1" {
		t.Fatalf("open_count = %q, want 1", out)
	}
}

func TestLatencyTool_Report(t *testing.T) {
	tracker := tools.NewToolLatencyTracker()
	tracker.Record("Read", 5e6)  // 5ms
	tracker.Record("Read", 10e6) // 10ms
	tracker.Record("Bash", 50e6) // 50ms

	tool := &LatencyTool{Tracker: tracker}
	ctx := context.Background()

	// Report.
	input, _ := json.Marshal(map[string]string{"action": "report"})
	out, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if !strings.Contains(out, "Read") || !strings.Contains(out, "Bash") {
		t.Fatalf("report = %q", out)
	}

	// P50.
	input, _ = json.Marshal(map[string]string{"action": "p50", "tool": "Read"})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("p50: %v", err)
	}
	if out == "0s" {
		t.Fatalf("p50 = %q, want non-zero", out)
	}

	// P95.
	input, _ = json.Marshal(map[string]string{"action": "p95", "tool": "Read"})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("p95: %v", err)
	}
	if out == "0s" {
		t.Fatalf("p95 = %q, want non-zero", out)
	}
}

func TestLatencyTool_InvalidAction(t *testing.T) {
	tracker := tools.NewToolLatencyTracker()
	tool := &LatencyTool{Tracker: tracker}
	input, _ := json.Marshal(map[string]string{"action": "nope"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
}

// --- E2E Acceptance Tests (GOL-37) ---

func setupGoProject(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	gitRun := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s %v", args, out, err)
		}
	}
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/testproject\n\ngo 1.22\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644)
	gitRun("init")
	gitRun("config", "user.email", "test@test.com")
	gitRun("config", "user.name", "test")
	gitRun("add", ".")
	gitRun("commit", "-m", "initial commit")
	return dir
}

func TestPlanTool_E2E_FullLifecycle(t *testing.T) {
	store := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	tool := &PlanTool{Store: store}
	ctx := context.Background()

	// Create two tasks.
	input, _ := json.Marshal(map[string]string{"action": "create", "title": "setup database"})
	out1, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("create 1: %v", err)
	}
	var task1 artifact.Artifact
	json.Unmarshal([]byte(out1), &task1)

	input, _ = json.Marshal(map[string]string{"action": "create", "title": "run migrations"})
	out2, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("create 2: %v", err)
	}
	var task2 artifact.Artifact
	json.Unmarshal([]byte(out2), &task2)

	if task1.ID == task2.ID {
		t.Fatal("two tasks got same ID")
	}

	// Get first task.
	input, _ = json.Marshal(map[string]string{"action": "get", "id": task1.ID})
	out, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out, "setup database") {
		t.Fatalf("get output missing title: %q", out)
	}

	// Update first to done.
	input, _ = json.Marshal(map[string]string{"action": "update", "id": task1.ID, "status": "done"})
	_, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// List — verify 2 tasks, first is done.
	input, _ = json.Marshal(map[string]string{"action": "list"})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var tasks []artifact.Artifact
	json.Unmarshal([]byte(out), &tasks)
	if len(tasks) != 2 {
		t.Fatalf("list: got %d tasks, want 2", len(tasks))
	}
	for _, task := range tasks {
		if task.ID == task1.ID && task.Status != artifact.StatusDone {
			t.Errorf("task1 status = %q, want done", task.Status)
		}
	}

	// Topo sort — both should appear.
	input, _ = json.Marshal(map[string]string{"action": "topo_sort"})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("topo_sort: %v", err)
	}
	if !strings.Contains(out, task1.ID) || !strings.Contains(out, task2.ID) {
		t.Fatalf("topo_sort missing task IDs: %q", out)
	}
}

func TestGitTool_E2E_AllActions(t *testing.T) {
	dir := setupGoProject(t)
	repo := tools.NewGitRepo(dir)
	tool := &GitTool{Repo: repo}
	ctx := context.Background()

	gitRun := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s %v", args, out, err)
		}
	}

	// Status — clean.
	input, _ := json.Marshal(map[string]string{"action": "status"})
	out, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	var status tools.GitStatus
	json.Unmarshal([]byte(out), &status)
	if !status.Clean {
		t.Fatal("repo should be clean after init")
	}

	// Dirty the repo.
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello\n"), 0o644)

	// Status — dirty.
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("status dirty: %v", err)
	}
	json.Unmarshal([]byte(out), &status)
	if status.Clean {
		t.Fatal("repo should be dirty after adding file")
	}

	// Diff.
	input, _ = json.Marshal(map[string]string{"action": "diff"})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	// Untracked files don't show in diff, but modified ones do.
	// Let's also modify an existing file.
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() { println(\"updated\") }\n"), 0o644)
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("diff 2: %v", err)
	}
	if !strings.Contains(out, "main.go") {
		t.Fatalf("diff should contain main.go: %q", out)
	}

	// Commit.
	gitRun("add", ".")
	gitRun("commit", "-m", "add new file")

	// Log.
	input, _ = json.Marshal(map[string]any{"action": "log", "n": 2})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	if !strings.Contains(out, "add new file") {
		t.Fatalf("log missing commit message: %q", out)
	}

	// Branch.
	input, _ = json.Marshal(map[string]string{"action": "branch"})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("branch: %v", err)
	}
	if out != "main" && out != "master" {
		t.Fatalf("branch = %q, want main or master", out)
	}
}

func TestTestTool_E2E_RealPassingTest(t *testing.T) {
	dir := setupGoProject(t)
	// Write a passing test.
	os.WriteFile(filepath.Join(dir, "main_test.go"), []byte(`package main

import "testing"

func TestAdd(t *testing.T) {
	if 1+1 != 2 {
		t.Fatal("math is broken")
	}
}
`), 0o644)

	tool := &TestTool{WorkDir: dir}
	input, _ := json.Marshal(map[string]any{"action": "run", "args": []string{"./..."}})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("test run: %v", err)
	}
	var result tools.TestResult
	json.Unmarshal([]byte(out), &result)
	if result.Passed < 1 {
		t.Fatalf("passed = %d, want >= 1", result.Passed)
	}
	if result.Failed != 0 {
		t.Fatalf("failed = %d, want 0", result.Failed)
	}
}

func TestTestTool_E2E_RealFailingTest(t *testing.T) {
	dir := setupGoProject(t)
	os.WriteFile(filepath.Join(dir, "main_test.go"), []byte(`package main

import "testing"

func TestBroken(t *testing.T) {
	t.Fatal("intentional failure")
}
`), 0o644)

	tool := &TestTool{WorkDir: dir}
	input, _ := json.Marshal(map[string]any{"action": "run", "args": []string{"./..."}})
	out, err := tool.Execute(context.Background(), input)
	// Test tool should NOT return error for test failures — it parses the output.
	if err != nil {
		t.Fatalf("test run: %v", err)
	}
	var result tools.TestResult
	json.Unmarshal([]byte(out), &result)
	if result.Failed < 1 {
		t.Fatalf("failed = %d, want >= 1", result.Failed)
	}
}

func TestArchTool_E2E_AnalyzeDjinn(t *testing.T) {
	// Use Djinn's own source as workspace (read-only).
	workDir := filepath.Join("..", "..")
	absDir, err := filepath.Abs(workDir)
	if err != nil {
		t.Skip("cannot resolve Djinn root")
	}

	tool := &ArchTool{WorkDir: absDir}
	ctx := context.Background()

	// Analyze.
	input, _ := json.Marshal(map[string]string{"action": "analyze"})
	out, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	var report tools.ArchReport
	json.Unmarshal([]byte(out), &report)
	if len(report.Packages) == 0 {
		t.Fatal("expected packages in arch report")
	}
}

func TestDiscourseTool_E2E_FullLifecycle(t *testing.T) {
	store := tools.NewDiscourseStore(filepath.Join(t.TempDir(), "discourse.json"))
	tool := &DiscourseTool{Store: store}
	ctx := context.Background()

	// Create topic.
	input, _ := json.Marshal(map[string]string{
		"action": "create_topic", "scope": "/e2e", "title": "Auth Module", "kind": "feature",
	})
	out, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("create_topic: %v", err)
	}
	var topic tools.Topic
	json.Unmarshal([]byte(out), &topic)
	if topic.Title != "Auth Module" {
		t.Fatalf("topic title = %q", topic.Title)
	}

	// Create thread.
	input, _ = json.Marshal(map[string]string{
		"action": "create_thread", "scope": "/e2e", "topic_id": topic.ID,
	})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("create_thread: %v", err)
	}
	var thread tools.Thread
	json.Unmarshal([]byte(out), &thread)

	// Append message.
	input, _ = json.Marshal(map[string]string{
		"action": "append", "scope": "/e2e", "topic_id": topic.ID,
		"thread_id": thread.ID, "role": "user", "content": "implement OAuth2",
	})
	_, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	// Get topic — verify message.
	input, _ = json.Marshal(map[string]string{
		"action": "get_topic", "scope": "/e2e", "topic_id": topic.ID,
	})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("get_topic: %v", err)
	}
	if !strings.Contains(out, "implement OAuth2") {
		t.Fatalf("get_topic missing message content: %q", out)
	}

	// Stale — should be empty (just created).
	input, _ = json.Marshal(map[string]string{"action": "stale", "scope": "/e2e"})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("stale: %v", err)
	}
	if out != "[]" && out != "null" {
		t.Fatalf("stale should be empty, got %q", out)
	}

	// Open count.
	input, _ = json.Marshal(map[string]string{"action": "open_count", "scope": "/e2e"})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("open_count: %v", err)
	}
	if out != "1" {
		t.Fatalf("open_count = %q, want 1", out)
	}
}

func TestReconcileTool_E2E_DriftReport(t *testing.T) {
	dir := setupGoProject(t)
	// Add a passing test.
	os.WriteFile(filepath.Join(dir, "main_test.go"), []byte(`package main

import "testing"

func TestOK(t *testing.T) {}
`), 0o644)

	store := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	// Add 2 tasks: 1 done, 1 pending.
	store.Add(artifact.Artifact{Title: "task A", Kind: artifact.KindTask, Status: artifact.StatusDone})
	store.Add(artifact.Artifact{Title: "task B", Kind: artifact.KindTask, Status: artifact.StatusPending})

	tool := &ReconcileTool{PlanStore: store, WorkDir: dir}
	input, _ := json.Marshal(map[string]string{"action": "drift"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("drift: %v", err)
	}

	var report tools.DriftReport
	json.Unmarshal([]byte(out), &report)
	if report.Functionality.Score == 0 || report.Functionality.Score == 100 {
		t.Fatalf("functionality score = %f, want between 0 and 100 exclusive", report.Functionality.Score)
	}
	if report.TasksToConvergence != 1 {
		t.Errorf("tasks to convergence = %d, want 1", report.TasksToConvergence)
	}
}

func TestLatencyTool_E2E_RecordAndQuery(t *testing.T) {
	tracker := tools.NewToolLatencyTracker()
	// Record samples: 1ms to 10ms.
	for i := 1; i <= 10; i++ {
		tracker.Record("Read", time.Duration(i)*time.Millisecond)
	}
	tracker.Record("Bash", 50*time.Millisecond)

	tool := &LatencyTool{Tracker: tracker}
	ctx := context.Background()

	// Report.
	input, _ := json.Marshal(map[string]string{"action": "report"})
	out, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if !strings.Contains(out, "Read") || !strings.Contains(out, "Bash") {
		t.Fatalf("report missing tools: %q", out)
	}

	// P50 for Read.
	input, _ = json.Marshal(map[string]string{"action": "p50", "tool": "Read"})
	out, err = tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("p50: %v", err)
	}
	if out == "0s" {
		t.Fatal("p50 should be non-zero")
	}

	// P95 for Read — should be >= P50.
	input, _ = json.Marshal(map[string]string{"action": "p95", "tool": "Read"})
	out2, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("p95: %v", err)
	}
	if out2 == "0s" {
		t.Fatal("p95 should be non-zero")
	}
}

// TSK-341: MCP upgrade path — registering a tool with the same name
// replaces the built-in seamlessly.
func TestRegistry_MCPUpgradePath(t *testing.T) {
	reg := NewRegistry()
	dir := t.TempDir()
	RegisterAeonShellTools(reg, dir, dir)

	// Built-in plan tool exists.
	original, err := reg.Get("plan")
	if err != nil {
		t.Fatalf("plan tool not found: %v", err)
	}

	// Simulate MCP server providing an upgraded "plan" tool.
	upgraded := &stubTool{name: "plan", desc: "MCP-enhanced plan tool"}
	reg.Register(upgraded)

	// The registry now returns the upgraded tool, not the built-in.
	got, err := reg.Get("plan")
	if err != nil {
		t.Fatalf("get after upgrade: %v", err)
	}
	if got.Description() == original.Description() {
		t.Fatal("upgrade should replace built-in description")
	}
	if got.Description() != "MCP-enhanced plan tool" {
		t.Fatalf("description = %q, want MCP-enhanced", got.Description())
	}

	// Other tools remain unaffected.
	if _, err := reg.Get("test"); err != nil {
		t.Fatalf("test tool missing after plan upgrade: %v", err)
	}
}

// stubTool is a minimal Tool implementation for upgrade path testing.
type stubTool struct {
	name string
	desc string
}

func (s *stubTool) Name() string                                                       { return s.name }
func (s *stubTool) Description() string                                                { return s.desc }
func (s *stubTool) InputSchema() json.RawMessage                                       { return json.RawMessage(`{}`) }
func (s *stubTool) Execute(_ context.Context, _ json.RawMessage) (string, error) { return "stub", nil }

func TestAllShellTools_NameDescription(t *testing.T) {
	store := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	discourse := tools.NewDiscourseStore(filepath.Join(t.TempDir(), "discourse.json"))
	repo := tools.NewGitRepo(t.TempDir())
	tracker := tools.NewToolLatencyTracker()

	allTools := []Tool{
		&PlanTool{Store: store},
		&TestTool{WorkDir: t.TempDir()},
		&GitTool{Repo: repo},
		&ArchTool{WorkDir: t.TempDir()},
		&DiscourseTool{Store: discourse},
		&ReconcileTool{PlanStore: store, WorkDir: t.TempDir()},
		&LatencyTool{Tracker: tracker},
	}

	for _, tool := range allTools {
		if tool.Name() == "" {
			t.Fatalf("tool has empty name")
		}
		if tool.Description() == "" {
			t.Fatalf("tool %q has empty description", tool.Name())
		}
		schema := tool.InputSchema()
		if len(schema) == 0 {
			t.Fatalf("tool %q has empty schema", tool.Name())
		}
		// Verify schema is valid JSON.
		var parsed map[string]any
		if err := json.Unmarshal(schema, &parsed); err != nil {
			t.Fatalf("tool %q schema is invalid JSON: %v", tool.Name(), err)
		}
	}
}
