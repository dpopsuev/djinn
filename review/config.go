// config.go — Review backpressure configuration (TSK-441).
package review

// Config holds review backpressure settings from djinn.yaml.
type Config struct {
	MaxFiles        int  `yaml:"max_files"`         // default: 10
	MaxLOCDelta     int  `yaml:"max_loc_delta"`     // default: 500
	MaxPackages     int  `yaml:"max_packages"`      // default: 4
	MaxNewFiles     int  `yaml:"max_new_files"`     // default: 5
	MaxDeletedFiles int  `yaml:"max_deleted_files"` // default: 3
	OnNewDependency bool `yaml:"on_new_dependency"` // default: true
	ScopeAnchor     bool `yaml:"scope_anchor"`      // default: true
	AgentNarration  bool `yaml:"agent_narration"`   // default: false
}

// DefaultConfig returns conservative defaults.
func DefaultConfig() Config {
	return Config{
		MaxFiles:        10,
		MaxLOCDelta:     500,
		MaxPackages:     4,
		MaxNewFiles:     5,
		MaxDeletedFiles: 3,
		OnNewDependency: true,
		ScopeAnchor:     true,
		AgentNarration:  false,
	}
}
