package signal

// WorkstreamHealth represents the computed health of a single workstream.
type WorkstreamHealth struct {
	Workstream  string
	Level       FlagLevel
	AgentHealth map[string]FlagLevel // per-agent current flag (keyed by Source)
	Latest      Signal
}

// ComputeHealth computes the health for each workstream from a set of signals.
// Uses worst-flag-wins: the highest (worst) FlagLevel seen for each workstream.
// Tracks per-agent health via Signal.Source (empty Source is ignored for agent tracking).
func ComputeHealth(signals []Signal) map[string]WorkstreamHealth {
	health := make(map[string]WorkstreamHealth)
	for i := range signals {
		h, ok := health[signals[i].Workstream]
		if !ok {
			h = WorkstreamHealth{
				Workstream:  signals[i].Workstream,
				AgentHealth: make(map[string]FlagLevel),
			}
		}

		if signals[i].Level > h.Level {
			h.Level = signals[i].Level
		}

		if signals[i].Source != "" {
			if existing, exists := h.AgentHealth[signals[i].Source]; !exists || signals[i].Level > existing {
				h.AgentHealth[signals[i].Source] = signals[i].Level
			}
		}

		if !ok || signals[i].Timestamp.After(h.Latest.Timestamp) {
			h.Latest = signals[i]
		}

		health[signals[i].Workstream] = h
	}
	return health
}
