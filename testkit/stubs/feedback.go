package stubs

import (
	"sync"

	"github.com/dpopsuev/djinn/ari"
)

// FeedbackStub implements both MetricsPort (driven) and EventIngressPort (driving).
// This proves the bidirectional MCP pattern: same backend serves both ports.
type FeedbackStub struct {
	mu        sync.Mutex
	metrics   map[string]float64
	threshold float64
	alertCh   chan ari.Alert
	fired     bool
}

// NewFeedbackStub creates a feedback stub with initial metrics and a threshold.
// When a metric exceeds the threshold, an alert is sent on the Alerts channel.
func NewFeedbackStub(threshold float64, initial map[string]float64) *FeedbackStub {
	m := make(map[string]float64)
	for k, v := range initial {
		m[k] = v
	}
	return &FeedbackStub{
		metrics:   m,
		threshold: threshold,
		alertCh:   make(chan ari.Alert, 10),
	}
}

// Query implements MetricsPort (driven side).
func (f *FeedbackStub) Query(metric string) float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.metrics[metric]
}

// Alerts implements EventIngressPort (driving side).
func (f *FeedbackStub) Alerts() <-chan ari.Alert {
	return f.alertCh
}

// SetMetric updates a metric value. If it exceeds the threshold, fires an alert.
func (f *FeedbackStub) SetMetric(metric string, value float64) {
	f.mu.Lock()
	f.metrics[metric] = value
	shouldFire := value > f.threshold && !f.fired
	if shouldFire {
		f.fired = true
	}
	f.mu.Unlock()

	if shouldFire {
		f.alertCh <- ari.Alert{
			Source: "feedback-stub",
			Metric: metric,
			Value:  value,
			Level:  "critical",
		}
	}
}

// RecoverMetric sets a metric below threshold. Does NOT fire an alert.
func (f *FeedbackStub) RecoverMetric(metric string, value float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.metrics[metric] = value
	f.fired = false
}

// Fired reports whether an alert has been fired.
func (f *FeedbackStub) Fired() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.fired
}
