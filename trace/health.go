// health.go — TraceHealthAnalyzer detects error patterns in the ring buffer (TSK-486).
//
// Pure function over ring snapshot. Detects: consecutive errors per server/tool,
// latency spikes (p95 exceeding baseline), error rate above threshold.
package trace

import "time"

// HealthAlert describes a detected anomaly in the trace ring.
type HealthAlert struct {
	Pattern   string    `json:"pattern"` // "consecutive_errors", "latency_spike", "error_rate"
	Component Component `json:"component"`
	Server    string    `json:"server,omitempty"`
	Tool      string    `json:"tool,omitempty"`
	Severity  Severity  `json:"severity"`
	Detail    string    `json:"detail"`
	Evidence  []string  `json:"evidence,omitempty"` // event IDs
}

// Severity levels for health alerts.
type Severity int

const (
	SeverityWarning  Severity = iota // Yellow signal
	SeverityError                    // Red signal
	SeverityCritical                 // Black signal
)

// Default detection thresholds.
const (
	DefaultConsecutiveErrors = 3
	DefaultLatencyMultiplier = 2.0 // spike = 2x baseline
	DefaultErrorRatePercent  = 20  // > 20% error rate
)

// HealthConfig configures detection thresholds.
type HealthConfig struct {
	ConsecutiveErrors int           // trigger after N consecutive errors (default: 3)
	LatencyMultiplier float64       // spike when p95 > multiplier * baseline (default: 2.0)
	ErrorRatePercent  int           // trigger when error rate > N% (default: 20)
	Window            time.Duration // analysis window (default: 5 minutes)
}

// DefaultHealthConfig returns conservative defaults.
func DefaultHealthConfig() HealthConfig {
	return HealthConfig{
		ConsecutiveErrors: DefaultConsecutiveErrors,
		LatencyMultiplier: DefaultLatencyMultiplier,
		ErrorRatePercent:  DefaultErrorRatePercent,
		Window:            5 * time.Minute, //nolint:mnd // 5 minute window
	}
}

// Analyze inspects the ring buffer for health anomalies.
func Analyze(r *Ring, cfg HealthConfig) []HealthAlert {
	if r == nil {
		return nil
	}

	var since time.Time
	if cfg.Window > 0 {
		since = time.Now().Add(-cfg.Window)
	}
	events := r.Since(since)
	if len(events) == 0 {
		return nil
	}

	var alerts []HealthAlert
	alerts = append(alerts, detectConsecutiveErrors(events, cfg.ConsecutiveErrors)...)
	alerts = append(alerts, detectErrorRate(events, cfg.ErrorRatePercent)...)
	alerts = append(alerts, detectLatencySpikes(events, cfg.LatencyMultiplier)...)
	return alerts
}

// detectConsecutiveErrors finds N+ consecutive errors on the same server+tool.
func detectConsecutiveErrors(events []TraceEvent, threshold int) []HealthAlert {
	if threshold <= 0 {
		return nil
	}

	// Track consecutive error count per server+tool.
	type key struct{ server, tool string }
	streaks := make(map[key]struct {
		count    int
		eventIDs []string
	})

	var alerts []HealthAlert
	for i := range events {
		e := &events[i]
		if e.Server == "" || e.Action != "call"+ActionDoneSuffix {
			continue
		}
		k := key{e.Server, e.Tool}
		s := streaks[k]
		if e.Error {
			s.count++
			s.eventIDs = append(s.eventIDs, e.ID)
		} else {
			s.count = 0
			s.eventIDs = nil
		}
		streaks[k] = s

		if s.count >= threshold {
			alerts = append(alerts, HealthAlert{
				Pattern:   "consecutive_errors",
				Component: e.Component,
				Server:    e.Server,
				Tool:      e.Tool,
				Severity:  SeverityError,
				Detail:    e.Tool + " on " + e.Server + " failed " + itoa(s.count) + " times consecutively",
				Evidence:  copyStrings(s.eventIDs),
			})
		}
	}
	return alerts
}

// detectErrorRate flags server+tool pairs with high error rates.
func detectErrorRate(events []TraceEvent, thresholdPct int) []HealthAlert {
	if thresholdPct <= 0 {
		return nil
	}

	type stats struct {
		total  int
		errors int
	}
	byKey := make(map[string]*stats)

	for i := range events {
		e := &events[i]
		if e.Server == "" || e.Action != "call"+ActionDoneSuffix {
			continue
		}
		k := e.Server + "|" + e.Tool
		s, ok := byKey[k]
		if !ok {
			s = &stats{}
			byKey[k] = s
		}
		s.total++
		if e.Error {
			s.errors++
		}
	}

	var alerts []HealthAlert
	for k, s := range byKey {
		if s.total < 5 { //nolint:mnd // need at least 5 calls for meaningful rate
			continue
		}
		rate := (s.errors * 100) / s.total
		if rate >= thresholdPct {
			server, tool := splitKey(k)
			alerts = append(alerts, HealthAlert{
				Pattern:   "error_rate",
				Component: ComponentMCP,
				Server:    server,
				Tool:      tool,
				Severity:  SeverityWarning,
				Detail:    tool + " on " + server + " error rate " + itoa(rate) + "% (" + itoa(s.errors) + "/" + itoa(s.total) + ")",
			})
		}
	}
	return alerts
}

// detectLatencySpikes finds server+tool pairs where recent p95 exceeds baseline.
func detectLatencySpikes(events []TraceEvent, multiplier float64) []HealthAlert {
	if multiplier <= 0 {
		return nil
	}

	// Split events into first half (baseline) and second half (recent).
	if len(events) < 10 { //nolint:mnd // need at least 10 events
		return nil
	}
	mid := len(events) / 2
	baselineLatencies := groupEventLatencies(events[:mid])
	recentLatencies := groupEventLatencies(events[mid:])

	var alerts []HealthAlert
	for k, baseline := range baselineLatencies {
		recent, ok := recentLatencies[k]
		if !ok || len(baseline) < 3 || len(recent) < 3 { //nolint:mnd // minimum samples
			continue
		}
		bp95 := percentileDuration(baseline, 95) //nolint:mnd // p95
		rp95 := percentileDuration(recent, 95)   //nolint:mnd // p95
		if bp95 > 0 && float64(rp95) > float64(bp95)*multiplier {
			server, tool := splitKey(k)
			alerts = append(alerts, HealthAlert{
				Pattern:   "latency_spike",
				Component: ComponentMCP,
				Server:    server,
				Tool:      tool,
				Severity:  SeverityWarning,
				Detail:    tool + " on " + server + " p95 spiked from " + bp95.String() + " to " + rp95.String(),
			})
		}
	}
	return alerts
}

func groupEventLatencies(events []TraceEvent) map[string][]time.Duration {
	m := make(map[string][]time.Duration)
	for i := range events {
		if events[i].Latency > 0 && events[i].Server != "" {
			k := events[i].Server + "|" + events[i].Tool
			m[k] = append(m[k], events[i].Latency)
		}
	}
	return m
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 { //nolint:mnd // single digit
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10)) //nolint:mnd // base 10
}

func copyStrings(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}
