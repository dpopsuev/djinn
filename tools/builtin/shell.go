// shell.go — Aeon Shell: 7 built-in tools that wrap Go packages in tools/.
//
// These are the agent's "GNU tools" — plan, test, git, arch, discourse,
// reconcile, latency. Each implements builtin.Tool and delegates to the
// underlying Go library.
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/dpopsuev/djinn/tools"
)

// ── PlanTool ────────────────────────────────────────────────────────────

// PlanTool wraps tools.TaskStore for in-process task tracking.
type PlanTool struct {
	Store *tools.TaskStore
}

func (t *PlanTool) Name() string        { return "plan" }
func (t *PlanTool) Description() string { return "In-process task tracker: create, get, update, list, topo_sort tasks" }
func (t *PlanTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {"type": "string", "enum": ["create", "get", "update", "list", "topo_sort"]},
			"title":  {"type": "string"},
			"id":     {"type": "string"},
			"status": {"type": "string"}
		},
		"required": ["action"]
	}`)
}

func (t *PlanTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var req struct {
		Action string `json:"action"`
		Title  string `json:"title"`
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("plan: %w", err)
	}

	switch req.Action {
	case "create":
		if req.Title == "" {
			return "", fmt.Errorf("plan create: title required")
		}
		task := t.Store.Create(req.Title)
		out, _ := json.Marshal(task)
		return string(out), nil

	case "get":
		if req.ID == "" {
			return "", fmt.Errorf("plan get: id required")
		}
		task, ok := t.Store.Get(req.ID)
		if !ok {
			return "", fmt.Errorf("plan get: task %q not found", req.ID)
		}
		out, _ := json.Marshal(task)
		return string(out), nil

	case "update":
		if req.ID == "" || req.Status == "" {
			return "", fmt.Errorf("plan update: id and status required")
		}
		if err := t.Store.Update(req.ID, req.Status); err != nil {
			return "", fmt.Errorf("plan update: %w", err)
		}
		return fmt.Sprintf("updated %s to %s", req.ID, req.Status), nil

	case "list":
		tasks := t.Store.List()
		out, _ := json.Marshal(tasks)
		return string(out), nil

	case "topo_sort":
		tasks := t.Store.TopoSort()
		out, _ := json.Marshal(tasks)
		return string(out), nil

	default:
		return "", fmt.Errorf("plan: unknown action %q", req.Action)
	}
}

// ── TestTool ────────────────────────────────────────────────────────────

// TestTool wraps `go test -json` execution and tools.ParseGoTestJSON.
type TestTool struct {
	WorkDir string
}

func (t *TestTool) Name() string        { return "test" }
func (t *TestTool) Description() string { return "Run go tests and parse results: run or parse" }
func (t *TestTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action":  {"type": "string", "enum": ["run", "parse"]},
			"args":    {"type": "array", "items": {"type": "string"}},
			"data":    {"type": "string"}
		},
		"required": ["action"]
	}`)
}

func (t *TestTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req struct {
		Action string   `json:"action"`
		Args   []string `json:"args"`
		Data   string   `json:"data"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("test: %w", err)
	}

	switch req.Action {
	case "run":
		args := []string{"test", "-json"}
		if len(req.Args) > 0 {
			args = append(args, req.Args...)
		} else {
			args = append(args, "./...")
		}
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Dir = t.WorkDir
		out, err := cmd.CombinedOutput()
		// Parse even on error — go test exits non-zero when tests fail.
		result, parseErr := tools.ParseGoTestJSON(strings.NewReader(string(out)))
		if parseErr != nil {
			if err != nil {
				return string(out), fmt.Errorf("test run: %w", err)
			}
			return string(out), fmt.Errorf("test parse: %w", parseErr)
		}
		j, _ := json.Marshal(result)
		return string(j), nil

	case "parse":
		if req.Data == "" {
			return "", fmt.Errorf("test parse: data required")
		}
		result, err := tools.ParseGoTestJSON(strings.NewReader(req.Data))
		if err != nil {
			return "", fmt.Errorf("test parse: %w", err)
		}
		j, _ := json.Marshal(result)
		return string(j), nil

	default:
		return "", fmt.Errorf("test: unknown action %q", req.Action)
	}
}

// ── GitTool ─────────────────────────────────────────────────────────────

// GitTool wraps tools.GitRepo for structured git operations.
type GitTool struct {
	Repo *tools.GitRepo
}

func (t *GitTool) Name() string        { return "git" }
func (t *GitTool) Description() string { return "Structured git operations: status, diff, log, branch" }
func (t *GitTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {"type": "string", "enum": ["status", "diff", "log", "branch"]},
			"n":      {"type": "integer"}
		},
		"required": ["action"]
	}`)
}

func (t *GitTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req struct {
		Action string `json:"action"`
		N      int    `json:"n"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("git: %w", err)
	}

	switch req.Action {
	case "status":
		status, err := t.Repo.Status(ctx)
		if err != nil {
			return "", fmt.Errorf("git status: %w", err)
		}
		j, _ := json.Marshal(status)
		return string(j), nil

	case "diff":
		diff, err := t.Repo.Diff(ctx)
		if err != nil {
			return "", fmt.Errorf("git diff: %w", err)
		}
		return diff, nil

	case "log":
		n := req.N
		if n <= 0 {
			n = 10
		}
		commits, err := t.Repo.Log(ctx, n)
		if err != nil {
			return "", fmt.Errorf("git log: %w", err)
		}
		j, _ := json.Marshal(commits)
		return string(j), nil

	case "branch":
		branch, err := t.Repo.CurrentBranch(ctx)
		if err != nil {
			return "", fmt.Errorf("git branch: %w", err)
		}
		return branch, nil

	default:
		return "", fmt.Errorf("git: unknown action %q", req.Action)
	}
}

// ── ArchTool ────────────────────────────────────────────────────────────

// ArchTool wraps tools.AnalyzeImports for import graph analysis.
type ArchTool struct {
	WorkDir string
}

func (t *ArchTool) Name() string        { return "arch" }
func (t *ArchTool) Description() string { return "Import graph analysis: analyze imports, detect cycles, check layer violations" }
func (t *ArchTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {"type": "string", "enum": ["analyze", "violations"]},
			"layers": {"type": "array", "items": {"type": "string"}}
		},
		"required": ["action"]
	}`)
}

func (t *ArchTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req struct {
		Action string   `json:"action"`
		Layers []string `json:"layers"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("arch: %w", err)
	}

	switch req.Action {
	case "analyze":
		report, err := tools.AnalyzeImports(ctx, t.WorkDir)
		if err != nil {
			return "", fmt.Errorf("arch analyze: %w", err)
		}
		j, _ := json.Marshal(report)
		return string(j), nil

	case "violations":
		if len(req.Layers) < 2 {
			return "", fmt.Errorf("arch violations: at least 2 layers required")
		}
		report, err := tools.AnalyzeImports(ctx, t.WorkDir)
		if err != nil {
			return "", fmt.Errorf("arch violations: %w", err)
		}
		violations := tools.CheckLayerViolations(report, req.Layers)
		j, _ := json.Marshal(violations)
		return string(j), nil

	default:
		return "", fmt.Errorf("arch: unknown action %q", req.Action)
	}
}

// ── DiscourseTool ───────────────────────────────────────────────────────

// DiscourseTool wraps tools.DiscourseStore for forum-style conversation persistence.
type DiscourseTool struct {
	Store *tools.DiscourseStore
}

func (t *DiscourseTool) Name() string        { return "discourse" }
func (t *DiscourseTool) Description() string { return "Forum-style conversation persistence: topics, threads, messages" }
func (t *DiscourseTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action":    {"type": "string", "enum": ["create_topic", "get_topic", "create_thread", "append", "stale", "open_count"]},
			"scope":     {"type": "string"},
			"topic_id":  {"type": "string"},
			"thread_id": {"type": "string"},
			"title":     {"type": "string"},
			"kind":      {"type": "string"},
			"role":      {"type": "string"},
			"content":   {"type": "string"},
			"threshold": {"type": "string"}
		},
		"required": ["action"]
	}`)
}

func (t *DiscourseTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var req struct {
		Action    string `json:"action"`
		Scope     string `json:"scope"`
		TopicID   string `json:"topic_id"`
		ThreadID  string `json:"thread_id"`
		Title     string `json:"title"`
		Kind      string `json:"kind"`
		Role      string `json:"role"`
		Content   string `json:"content"`
		Threshold string `json:"threshold"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("discourse: %w", err)
	}

	if req.Scope == "" {
		req.Scope = "/"
	}

	switch req.Action {
	case "create_topic":
		if req.Title == "" {
			return "", fmt.Errorf("discourse create_topic: title required")
		}
		kind := req.Kind
		if kind == "" {
			kind = "discussion"
		}
		topic := t.Store.CreateTopic(req.Scope, req.Title, kind)
		j, _ := json.Marshal(topic)
		return string(j), nil

	case "get_topic":
		if req.TopicID == "" {
			return "", fmt.Errorf("discourse get_topic: topic_id required")
		}
		topic, ok := t.Store.GetTopic(req.Scope, req.TopicID)
		if !ok {
			return "", fmt.Errorf("discourse get_topic: %q not found in scope %q", req.TopicID, req.Scope)
		}
		j, _ := json.Marshal(topic)
		return string(j), nil

	case "create_thread":
		if req.TopicID == "" {
			return "", fmt.Errorf("discourse create_thread: topic_id required")
		}
		thread, err := t.Store.CreateThread(req.Scope, req.TopicID)
		if err != nil {
			return "", fmt.Errorf("discourse create_thread: %w", err)
		}
		j, _ := json.Marshal(thread)
		return string(j), nil

	case "append":
		if req.TopicID == "" || req.ThreadID == "" || req.Content == "" {
			return "", fmt.Errorf("discourse append: topic_id, thread_id, and content required")
		}
		role := req.Role
		if role == "" {
			role = "assistant"
		}
		if err := t.Store.AppendMessage(req.Scope, req.TopicID, req.ThreadID, role, req.Content); err != nil {
			return "", fmt.Errorf("discourse append: %w", err)
		}
		return "ok", nil

	case "stale":
		stale := t.Store.StaleTopics(req.Scope, 24*60*60*1e9) // 24h default
		j, _ := json.Marshal(stale)
		return string(j), nil

	case "open_count":
		count := t.Store.OpenTopicCount(req.Scope)
		return fmt.Sprintf("%d", count), nil

	default:
		return "", fmt.Errorf("discourse: unknown action %q", req.Action)
	}
}

// ── ReconcileTool ───────────────────────────────────────────────────────

// ReconcileTool wraps tools.ComputeDrift for three-pillar reconciliation.
type ReconcileTool struct {
	PlanStore *tools.TaskStore
	WorkDir   string
}

func (t *ReconcileTool) Name() string        { return "reconcile" }
func (t *ReconcileTool) Description() string { return "Three-pillar drift reconciliation: functionality, structure, performance" }
func (t *ReconcileTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {"type": "string", "enum": ["drift"]}
		},
		"required": ["action"]
	}`)
}

func (t *ReconcileTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("reconcile: %w", err)
	}

	switch req.Action {
	case "drift":
		// Gather data from all three pillars.
		arch, _ := tools.AnalyzeImports(ctx, t.WorkDir)

		// Run go test on the tools package (lightweight) to get test data.
		testResult := &tools.TestResult{} // default empty
		cmd := exec.CommandContext(ctx, "go", "test", "-json", "-count=1", "./...")
		cmd.Dir = t.WorkDir
		if out, err := cmd.CombinedOutput(); err == nil {
			if parsed, parseErr := tools.ParseGoTestJSON(strings.NewReader(string(out))); parseErr == nil {
				testResult = parsed
			}
		}

		report := tools.ComputeDrift(t.PlanStore, arch, testResult)
		j, _ := json.Marshal(report)
		return string(j), nil

	default:
		return "", fmt.Errorf("reconcile: unknown action %q", req.Action)
	}
}

// ── LatencyTool ─────────────────────────────────────────────────────────

// LatencyTool wraps tools.ToolLatencyTracker for telemetry reporting.
type LatencyTool struct {
	Tracker *tools.ToolLatencyTracker
}

func (t *LatencyTool) Name() string        { return "latency" }
func (t *LatencyTool) Description() string { return "Tool latency telemetry: report, p50, p95 per tool" }
func (t *LatencyTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {"type": "string", "enum": ["report", "p50", "p95"]},
			"tool":   {"type": "string"}
		},
		"required": ["action"]
	}`)
}

func (t *LatencyTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var req struct {
		Action string `json:"action"`
		Tool   string `json:"tool"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("latency: %w", err)
	}

	switch req.Action {
	case "report":
		type entry struct {
			Name  string `json:"name"`
			P50   string `json:"p50"`
			P95   string `json:"p95"`
			Count int    `json:"count"`
		}
		var entries []entry
		for _, name := range t.Tracker.AllTools() {
			entries = append(entries, entry{
				Name:  name,
				P50:   t.Tracker.P50(name).String(),
				P95:   t.Tracker.P95(name).String(),
				Count: t.Tracker.Count(name),
			})
		}
		j, _ := json.Marshal(entries)
		return string(j), nil

	case "p50":
		if req.Tool == "" {
			return "", fmt.Errorf("latency p50: tool required")
		}
		return t.Tracker.P50(req.Tool).String(), nil

	case "p95":
		if req.Tool == "" {
			return "", fmt.Errorf("latency p95: tool required")
		}
		return t.Tracker.P95(req.Tool).String(), nil

	default:
		return "", fmt.Errorf("latency: unknown action %q", req.Action)
	}
}
