package builtin

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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
	store := tools.NewTaskStore(filepath.Join(t.TempDir(), "tasks.json"))
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
	var created tools.Task
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
	store := tools.NewTaskStore(filepath.Join(t.TempDir(), "tasks.json"))
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

func TestAllShellTools_NameDescription(t *testing.T) {
	store := tools.NewTaskStore(filepath.Join(t.TempDir(), "tasks.json"))
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
