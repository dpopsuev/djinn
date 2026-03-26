package tools

import (
	"sync"
	"testing"
	"time"
)

func TestToolLatencyTracker_RecordAndCount(t *testing.T) {
	tr := NewToolLatencyTracker()
	tr.Record("Read", 100*time.Millisecond)
	tr.Record("Read", 200*time.Millisecond)
	tr.Record("Write", 50*time.Millisecond)

	if c := tr.Count("Read"); c != 2 {
		t.Fatalf("Read count = %d, want 2", c)
	}
	if c := tr.Count("Write"); c != 1 {
		t.Fatalf("Write count = %d, want 1", c)
	}
	if c := tr.Count("NonExistent"); c != 0 {
		t.Fatalf("NonExistent count = %d, want 0", c)
	}
}

func TestToolLatencyTracker_P50(t *testing.T) {
	tr := NewToolLatencyTracker()

	// P50 of empty — should be 0.
	if p := tr.P50("Read"); p != 0 {
		t.Fatalf("empty P50 = %v, want 0", p)
	}

	// Add samples: 100, 200, 300, 400, 500
	for i := 1; i <= 5; i++ {
		tr.Record("Read", time.Duration(i*100)*time.Millisecond)
	}

	p50 := tr.P50("Read")
	// Index = int(4 * 0.5) = 2 → sorted[2] = 300ms
	if p50 != 300*time.Millisecond {
		t.Fatalf("P50 = %v, want 300ms", p50)
	}
}

func TestToolLatencyTracker_P95(t *testing.T) {
	tr := NewToolLatencyTracker()

	// 20 samples: 50ms, 100ms, 150ms, ..., 1000ms
	for i := 1; i <= 20; i++ {
		tr.Record("Bash", time.Duration(i*50)*time.Millisecond)
	}

	p95 := tr.P95("Bash")
	// Index = int(19 * 0.95) = 18 → sorted[18] = 950ms
	if p95 != 950*time.Millisecond {
		t.Fatalf("P95 = %v, want 950ms", p95)
	}
}

func TestToolLatencyTracker_P50_SingleSample(t *testing.T) {
	tr := NewToolLatencyTracker()
	tr.Record("Glob", 42*time.Millisecond)

	p50 := tr.P50("Glob")
	if p50 != 42*time.Millisecond {
		t.Fatalf("P50 single = %v, want 42ms", p50)
	}
}

func TestToolLatencyTracker_AllTools(t *testing.T) {
	tr := NewToolLatencyTracker()
	tr.Record("Write", time.Millisecond)
	tr.Record("Read", time.Millisecond)
	tr.Record("Bash", time.Millisecond)

	tools := tr.AllTools()
	if len(tools) != 3 {
		t.Fatalf("AllTools = %v, want 3 tools", tools)
	}
	// Should be sorted.
	if tools[0] != "Bash" || tools[1] != "Read" || tools[2] != "Write" {
		t.Fatalf("AllTools = %v, want sorted [Bash Read Write]", tools)
	}
}

func TestToolLatencyTracker_ConcurrentAccess(t *testing.T) {
	tr := NewToolLatencyTracker()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tr.Record("Read", time.Duration(i)*time.Millisecond)
			_ = tr.P50("Read")
			_ = tr.P95("Read")
			_ = tr.Count("Read")
			_ = tr.AllTools()
		}(i)
	}
	wg.Wait()

	if c := tr.Count("Read"); c != 50 {
		t.Fatalf("concurrent count = %d, want 50", c)
	}
}

func TestToolLatencyTracker_P50P95_UnsortedInput(t *testing.T) {
	tr := NewToolLatencyTracker()
	// Add out of order: 500, 100, 300, 200, 400
	tr.Record("Edit", 500*time.Millisecond)
	tr.Record("Edit", 100*time.Millisecond)
	tr.Record("Edit", 300*time.Millisecond)
	tr.Record("Edit", 200*time.Millisecond)
	tr.Record("Edit", 400*time.Millisecond)

	p50 := tr.P50("Edit")
	if p50 != 300*time.Millisecond {
		t.Fatalf("P50 unsorted = %v, want 300ms", p50)
	}

	p95 := tr.P95("Edit")
	// Index = int(4 * 0.95) = 3 → sorted[3] = 400ms
	if p95 != 400*time.Millisecond {
		t.Fatalf("P95 unsorted = %v, want 400ms", p95)
	}
}
