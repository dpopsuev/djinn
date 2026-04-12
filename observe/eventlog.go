// eventlog.go — real Observer implementation backed by Troupe's EventLog.
// Queries EventLog.Since() and filters/formats into TraceLine and HealthReport.
// Replaces TraceProjection as the introspection query layer.
package observe

import (
	"fmt"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/troupe/signal"
)

var _ Observer = (*EventLogObserver)(nil)

// EventLogObserver implements Observer by querying Troupe's EventLog.
type EventLogObserver struct {
	log          signal.EventLog
	stuckTimeout time.Duration // agent inactive for this long = stuck
}

// NewEventLogObserver creates an Observer backed by an EventLog.
func NewEventLogObserver(log signal.EventLog) *EventLogObserver {
	return &EventLogObserver{
		log:          log,
		stuckTimeout: 60 * time.Second,
	}
}

// WithStuckTimeout sets the inactivity threshold for stuck agent detection.
func (o *EventLogObserver) WithStuckTimeout(d time.Duration) *EventLogObserver {
	o.stuckTimeout = d
	return o
}

func (o *EventLogObserver) Trace(opts TraceOpts) ([]TraceLine, error) {
	last := opts.Last
	if last <= 0 {
		last = 50
	}

	// Read all events, then filter + limit.
	// EventLog.Since(-1) returns all.
	events := o.log.Since(-1)

	lines := make([]TraceLine, 0, len(events))
	for i := range events {
		e := &events[i]

		// Filter by kind.
		if opts.Kind != "" && e.Kind != opts.Kind {
			continue
		}

		// Filter by source.
		if opts.Source != "" && e.Source != opts.Source {
			continue
		}

		line := TraceLine{
			Timestamp: e.Timestamp,
			Source:    e.Source,
			Kind:      e.Kind,
			Summary:   formatSummary(e),
		}

		// Extract duration from TraceEvent data if present.
		if te, ok := e.Data.(telemetry.TraceEvent); ok {
			line.Duration = te.Latency.Milliseconds()
			if line.Summary == "" {
				line.Summary = te.Detail
			}
		}

		lines = append(lines, line)
	}

	// Limit to last N.
	if len(lines) > last {
		lines = lines[len(lines)-last:]
	}

	return lines, nil
}

func (o *EventLogObserver) Health() (HealthReport, error) {
	events := o.log.Since(-1)
	now := time.Now()

	// Track per-source activity.
	lastSeen := make(map[string]time.Time)
	errorCount := 0

	for i := range events {
		e := &events[i]
		if e.Source != "" {
			if e.Timestamp.After(lastSeen[e.Source]) {
				lastSeen[e.Source] = e.Timestamp
			}
		}

		// Count errors.
		if te, ok := e.Data.(telemetry.TraceEvent); ok && te.Error {
			errorCount++
		}
	}

	// Build report.
	report := HealthReport{
		AgentsAlive: len(lastSeen),
		Errors:      errorCount,
	}

	// Find last activity and stuck agents.
	for source, ts := range lastSeen {
		if ts.After(report.LastActivity) {
			report.LastActivity = ts
		}
		if now.Sub(ts) > o.stuckTimeout {
			report.StuckAgents = append(report.StuckAgents, source)
		}
	}

	return report, nil
}

func formatSummary(e *signal.Event) string {
	if e.Kind != "" {
		return fmt.Sprintf("[%s] %s", e.Source, e.Kind)
	}
	return e.Source
}
