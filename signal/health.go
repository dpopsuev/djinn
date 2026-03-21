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
	for _, s := range signals {
		h, ok := health[s.Workstream]
		if !ok {
			h = WorkstreamHealth{
				Workstream:  s.Workstream,
				AgentHealth: make(map[string]FlagLevel),
			}
		}

		if s.Level > h.Level {
			h.Level = s.Level
		}

		if s.Source != "" {
			if existing, exists := h.AgentHealth[s.Source]; !exists || s.Level > existing {
				h.AgentHealth[s.Source] = s.Level
			}
		}

		if !ok || s.Timestamp.After(h.Latest.Timestamp) {
			h.Latest = s
		}

		health[s.Workstream] = h
	}
	return health
}
