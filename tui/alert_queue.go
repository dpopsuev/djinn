// alert_queue.go — ring buffer for andon alerts.
//
// AlertQueue stores the most recent alerts, dropping the oldest when capacity
// is reached. Thread-safe for concurrent Push/Ack/read operations.
package tui

import (
	"sync"
	"time"
)

// Alert represents a single andon alert event.
type Alert struct {
	Level     AndonLevel
	Source    string
	Message   string
	Timestamp time.Time
	Acked     bool
}

// AlertQueue is a fixed-capacity ring buffer for alerts.
// When full, the oldest alert is dropped on Push.
type AlertQueue struct {
	alerts []Alert
	max    int // ring buffer capacity
	mu     sync.RWMutex
}

// NewAlertQueue creates an AlertQueue with the given capacity.
// Capacity must be at least 1; defaults to 50 if zero or negative.
func NewAlertQueue(capacity int) *AlertQueue {
	if capacity <= 0 {
		capacity = 50
	}
	return &AlertQueue{
		alerts: make([]Alert, 0, capacity),
		max:    capacity,
	}
}

// Push adds an alert, dropping the oldest if at capacity.
func (q *AlertQueue) Push(a Alert) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.alerts) >= q.max {
		// Drop oldest: shift left by one.
		copy(q.alerts, q.alerts[1:])
		q.alerts[len(q.alerts)-1] = a
	} else {
		q.alerts = append(q.alerts, a)
	}
}

// Alerts returns a copy of all alerts (oldest first).
func (q *AlertQueue) Alerts() []Alert {
	q.mu.RLock()
	defer q.mu.RUnlock()
	out := make([]Alert, len(q.alerts))
	copy(out, q.alerts)
	return out
}

// Ack marks the alert at the given index as acknowledged.
// Out-of-range indices are silently ignored.
func (q *AlertQueue) Ack(index int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if index >= 0 && index < len(q.alerts) {
		q.alerts[index].Acked = true
	}
}

// Unacked returns the count of unacknowledged alerts.
func (q *AlertQueue) Unacked() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	count := 0
	for i := range q.alerts {
		if !q.alerts[i].Acked {
			count++
		}
	}
	return count
}

// Worst returns the worst (highest) AndonLevel among unacked alerts.
// Returns AndonGreen if no unacked alerts exist.
func (q *AlertQueue) Worst() AndonLevel {
	q.mu.RLock()
	defer q.mu.RUnlock()
	worst := AndonGreen
	for i := range q.alerts {
		if !q.alerts[i].Acked && q.alerts[i].Level > worst {
			worst = q.alerts[i].Level
		}
	}
	return worst
}
