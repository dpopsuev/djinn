// sla.go — SLA/SLO definitions for Aeon Shell tools (GOL-37).
//
// Each tool has P50/P95 latency targets and an error rate ceiling.
// CheckSLA evaluates actual performance against targets.
package daemon

import "time"

// ToolSLA defines latency and error rate targets for a single tool.
type ToolSLA struct {
	Name        string
	P50Target   time.Duration
	P95Target   time.Duration
	MaxErrorPct float64 // 0.0-1.0
}

// SLAResult captures whether a tool met its SLA.
type SLAResult struct {
	Tool     string
	P50      time.Duration
	P95      time.Duration
	ErrorPct float64
	P50Met   bool
	P95Met   bool
	ErrorMet bool
	Overall  bool
}

// CheckSLA evaluates a tool's actual performance against its SLA targets.
func CheckSLA(sla ToolSLA, p50, p95 time.Duration, errorPct float64) SLAResult {
	r := SLAResult{
		Tool:     sla.Name,
		P50:      p50,
		P95:      p95,
		ErrorPct: errorPct,
		P50Met:   p50 <= sla.P50Target,
		P95Met:   p95 <= sla.P95Target,
		ErrorMet: errorPct <= sla.MaxErrorPct,
	}
	r.Overall = r.P50Met && r.P95Met && r.ErrorMet
	return r
}

// DefaultSLAs returns the SLA targets for all 8 Aeon Shell tools.
func DefaultSLAs() map[string]ToolSLA {
	return map[string]ToolSLA{
		"plan":      {Name: "plan", P50Target: 10 * time.Millisecond, P95Target: 50 * time.Millisecond, MaxErrorPct: 0.01},
		"test":      {Name: "test", P50Target: 5 * time.Second, P95Target: 30 * time.Second, MaxErrorPct: 0.05},
		"git":       {Name: "git", P50Target: 100 * time.Millisecond, P95Target: 1 * time.Second, MaxErrorPct: 0.01},
		"arch":      {Name: "arch", P50Target: 500 * time.Millisecond, P95Target: 5 * time.Second, MaxErrorPct: 0.02},
		"discourse": {Name: "discourse", P50Target: 10 * time.Millisecond, P95Target: 50 * time.Millisecond, MaxErrorPct: 0.01},
		"reconcile": {Name: "reconcile", P50Target: 2 * time.Second, P95Target: 10 * time.Second, MaxErrorPct: 0.05},
		"latency":   {Name: "latency", P50Target: 5 * time.Millisecond, P95Target: 20 * time.Millisecond, MaxErrorPct: 0.01},
		"render":    {Name: "render", P50Target: 50 * time.Millisecond, P95Target: 200 * time.Millisecond, MaxErrorPct: 0.01},
	}
}
