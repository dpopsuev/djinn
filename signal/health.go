package signal

// WorkstreamHealth represents the computed health of a single workstream.
type WorkstreamHealth struct {
	Workstream string
	Level      FlagLevel
	Latest     Signal
}

// ComputeHealth computes the health for each workstream from a set of signals.
// Uses worst-flag-wins: the highest (worst) FlagLevel seen for each workstream.
func ComputeHealth(signals []Signal) map[string]WorkstreamHealth {
	health := make(map[string]WorkstreamHealth)
	for _, s := range signals {
		h, ok := health[s.Workstream]
		if !ok || s.Level > h.Level {
			health[s.Workstream] = WorkstreamHealth{
				Workstream: s.Workstream,
				Level:      s.Level,
				Latest:     s,
			}
		} else if s.Timestamp.After(h.Latest.Timestamp) {
			h.Latest = s
			health[s.Workstream] = h
		}
	}
	return health
}
