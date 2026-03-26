// latency.go — Tool latency tracking for Djinn telemetry.
// Collects per-tool execution time samples and computes percentiles.
package tools

import (
	"sort"
	"sync"
	"time"
)

// ToolLatencyTracker collects per-tool latency samples.
type ToolLatencyTracker struct {
	mu      sync.Mutex
	samples map[string][]time.Duration // tool name -> latency samples
}

// NewToolLatencyTracker creates an empty tracker.
func NewToolLatencyTracker() *ToolLatencyTracker {
	return &ToolLatencyTracker{
		samples: make(map[string][]time.Duration),
	}
}

// Record adds a latency sample for the named tool.
func (t *ToolLatencyTracker) Record(tool string, d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.samples[tool] = append(t.samples[tool], d)
}

// P50 returns the 50th percentile (median) latency for the named tool.
// Returns 0 if no samples exist.
func (t *ToolLatencyTracker) P50(tool string) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return percentile(t.samples[tool], 0.50)
}

// P95 returns the 95th percentile latency for the named tool.
// Returns 0 if no samples exist.
func (t *ToolLatencyTracker) P95(tool string) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return percentile(t.samples[tool], 0.95)
}

// Count returns the number of latency samples for the named tool.
func (t *ToolLatencyTracker) Count(tool string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.samples[tool])
}

// AllTools returns a sorted list of all tool names with recorded samples.
func (t *ToolLatencyTracker) AllTools() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	names := make([]string, 0, len(t.samples))
	for name := range t.samples {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// percentile computes the p-th percentile from a slice of durations.
// p is in [0.0, 1.0]. Returns 0 for empty input.
// Makes a sorted copy to avoid mutating the original.
func percentile(samples []time.Duration, p float64) time.Duration {
	n := len(samples)
	if n == 0 {
		return 0
	}
	// Copy and sort.
	sorted := make([]time.Duration, n)
	copy(sorted, samples)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	idx := int(float64(n-1) * p)
	if idx >= n {
		idx = n - 1
	}
	return sorted[idx]
}
