//go:build lab

// lab_test.go — Aeon Shell Laboratory: baseline benchmarks.
//
// Measures latency and output size for each built-in tool on Djinn's own
// codebase. This is the "before" snapshot — optimize tools, then prove
// improvement by re-running.
//
// Run: go test ./tools/ -tags=lab -bench=. -benchtime=10x -v
package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// djinnRoot is Djinn's own source tree — the benchmark target.
const djinnRoot = ".."

func BenchmarkPlanTool(b *testing.B) {
	store := NewTaskStore(b.TempDir() + "/tasks.json")
	ctx := context.Background()

	b.Run("create", func(b *testing.B) {
		for i := range b.N {
			store.Create(fmt.Sprintf("task-%d", i))
		}
	})

	// Seed some tasks.
	for i := range 20 {
		t := store.Create(fmt.Sprintf("seed-%d", i))
		if i > 0 {
			t.DependsOn = []string{fmt.Sprintf("T-%03d", i)}
		}
	}

	b.Run("list", func(b *testing.B) {
		for range b.N {
			tasks := store.List()
			_ = len(tasks)
		}
	})

	b.Run("topo_sort", func(b *testing.B) {
		for range b.N {
			tasks := store.TopoSort()
			_ = len(tasks)
		}
	})
}

func BenchmarkTestTool_Parse(b *testing.B) {
	// Canned go test -json output.
	var lines []string
	for i := range 50 {
		name := fmt.Sprintf("Test%d", i)
		lines = append(lines,
			fmt.Sprintf(`{"Action":"run","Package":"example.com/foo","Test":"%s"}`, name),
			fmt.Sprintf(`{"Action":"output","Package":"example.com/foo","Test":"%s","Output":"--- PASS: %s\n"}`, name, name),
			fmt.Sprintf(`{"Action":"pass","Package":"example.com/foo","Test":"%s","Elapsed":0.01}`, name),
		)
	}
	lines = append(lines, `{"Action":"output","Package":"example.com/foo","Output":"coverage: 85.2% of statements\n"}`)
	lines = append(lines, `{"Action":"pass","Package":"example.com/foo","Elapsed":1.5}`)
	data := strings.Join(lines, "\n")

	b.ResetTimer()
	for range b.N {
		result, err := ParseGoTestJSON(strings.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
		if result.Passed != 50 {
			b.Fatalf("passed = %d", result.Passed)
		}
	}
	b.ReportMetric(float64(len(data))/4, "est_tokens/op")
}

func BenchmarkGitTool_Status(b *testing.B) {
	if _, err := exec.LookPath("git"); err != nil {
		b.Skip("git not on PATH")
	}
	repo := NewGitRepo(djinnRoot)
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		status, err := repo.Status(ctx)
		if err != nil {
			b.Fatal(err)
		}
		_ = status.Branch
	}
}

func BenchmarkGitTool_Log(b *testing.B) {
	if _, err := exec.LookPath("git"); err != nil {
		b.Skip("git not on PATH")
	}
	repo := NewGitRepo(djinnRoot)
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		commits, err := repo.Log(ctx, 10)
		if err != nil {
			b.Fatal(err)
		}
		_ = len(commits)
	}
}

func BenchmarkGitTool_Diff(b *testing.B) {
	if _, err := exec.LookPath("git"); err != nil {
		b.Skip("git not on PATH")
	}
	repo := NewGitRepo(djinnRoot)
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		diff, err := repo.Diff(ctx)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(diff))/4, "est_tokens/op")
	}
}

func BenchmarkArchTool_Analyze(b *testing.B) {
	if _, err := exec.LookPath("go"); err != nil {
		b.Skip("go not on PATH")
	}
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		report, err := AnalyzeImports(ctx, djinnRoot)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(report.Packages)), "packages/op")
	}
}

func BenchmarkGlobBare(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		cmd := exec.CommandContext(ctx, "find", djinnRoot, "-name", "*.go", "-type", "f")
		out, err := cmd.Output()
		if err != nil {
			b.Fatal(err)
		}
		lines := strings.Count(string(out), "\n")
		b.ReportMetric(float64(lines), "files/op")
		b.ReportMetric(float64(len(out))/4, "est_tokens/op")
	}
}

func BenchmarkGrepBare(b *testing.B) {
	rg, err := exec.LookPath("rg")
	if err != nil {
		b.Skip("rg not on PATH")
	}
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		cmd := exec.CommandContext(ctx, rg, "func.*Error", djinnRoot, "--type", "go")
		out, _ := cmd.Output()
		lines := strings.Count(string(out), "\n")
		b.ReportMetric(float64(lines), "matches/op")
		b.ReportMetric(float64(len(out))/4, "est_tokens/op")
	}
}

func BenchmarkReadFull(b *testing.B) {
	// Read a ~200-line file (tools/arch.go).
	target := djinnRoot + "/tools/arch.go"

	b.ResetTimer()
	for range b.N {
		cmd := exec.Command("cat", target)
		out, err := cmd.Output()
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(out))/4, "est_tokens/op")
	}
}

func BenchmarkDiscourseTool(b *testing.B) {
	store := NewDiscourseStore(b.TempDir() + "/discourse.json")

	b.Run("create_topic", func(b *testing.B) {
		for i := range b.N {
			store.CreateTopic("/bench", fmt.Sprintf("topic-%d", i), "feature")
		}
	})

	// Seed a topic + thread for message benchmarks.
	topic := store.CreateTopic("/bench", "seeded", "feature")
	thread, _ := store.CreateThread("/bench", topic.ID)

	b.Run("append_message", func(b *testing.B) {
		for i := range b.N {
			store.AppendMessage("/bench", topic.ID, thread.ID, "user", fmt.Sprintf("msg-%d", i))
		}
	})
}

func BenchmarkReconcileTool(b *testing.B) {
	store := NewTaskStore(b.TempDir() + "/tasks.json")
	for i := range 10 {
		t := store.Create(fmt.Sprintf("task-%d", i))
		if i%3 == 0 {
			t.Status = StatusDone
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		start := time.Now()
		arch, err := AnalyzeImports(ctx, djinnRoot)
		if err != nil {
			b.Fatal(err)
		}
		report := ComputeDrift(store, arch, &TestResult{Passed: 10, Failed: 1})
		_ = report.TasksToConvergence
		b.ReportMetric(float64(time.Since(start).Milliseconds()), "ms/op")
	}
}

func BenchmarkLatencyTool(b *testing.B) {
	tracker := NewToolLatencyTracker()
	for i := range 100 {
		tracker.Record("Read", time.Duration(i)*time.Millisecond)
		tracker.Record("Bash", time.Duration(i*2)*time.Millisecond)
	}

	b.ResetTimer()
	for range b.N {
		for _, name := range tracker.AllTools() {
			_ = tracker.P50(name)
			_ = tracker.P95(name)
		}
	}
}
