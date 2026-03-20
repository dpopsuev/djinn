package tier

// TierLevel represents a level in the recursive decomposition hierarchy.
type TierLevel int

const (
	Eco TierLevel = iota // Ecosystem — top-level orchestration
	Sys                  // System — cross-cutting concerns
	Com                  // Component — bounded context
	Mod                  // Module — leaf unit of work
)

func (l TierLevel) String() string {
	switch l {
	case Eco:
		return "eco"
	case Sys:
		return "sys"
	case Com:
		return "com"
	case Mod:
		return "mod"
	default:
		return "unknown"
	}
}

// Scope identifies a specific tier instance within the hierarchy.
type Scope struct {
	Level TierLevel
	Name  string
}
