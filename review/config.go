// config.go — Review backpressure configuration (TSK-441).
package review

// Config holds review backpressure settings from djinn.yaml.
type Config struct {
	// Tier 1: git diff
	MaxFiles        int  `yaml:"max_files"`         // default: 10
	MaxLOCDelta     int  `yaml:"max_loc_delta"`     // default: 500
	MaxPackages     int  `yaml:"max_packages"`      // default: 4
	MaxNewFiles     int  `yaml:"max_new_files"`     // default: 5
	MaxDeletedFiles int  `yaml:"max_deleted_files"` // default: 3
	OnNewDependency bool `yaml:"on_new_dependency"` // default: true

	// Tier 2: LSP symbol diff
	MaxExportedChanges      int  `yaml:"max_exported_changes"`      // default: 5
	MaxInterfacesIntroduced int  `yaml:"max_interfaces_introduced"` // default: 2
	OnInterfaceChange       bool `yaml:"on_interface_change"`       // default: true
	MaxStructFieldChanges   int  `yaml:"max_struct_field_changes"`  // default: 10
	MaxSignatureChanges     int  `yaml:"max_signature_changes"`     // default: 5

	// General
	ScopeAnchor    bool `yaml:"scope_anchor"`    // default: true
	AgentNarration bool `yaml:"agent_narration"` // default: false
}

// DefaultConfig returns conservative defaults.
func DefaultConfig() Config {
	return Config{
		// Tier 1
		MaxFiles:        10,
		MaxLOCDelta:     500,
		MaxPackages:     4,
		MaxNewFiles:     5,
		MaxDeletedFiles: 3,
		OnNewDependency: true,

		// Tier 2
		MaxExportedChanges:      5,
		MaxInterfacesIntroduced: 2,
		OnInterfaceChange:       true,
		MaxStructFieldChanges:   10,
		MaxSignatureChanges:     5,

		// General
		ScopeAnchor:    true,
		AgentNarration: false,
	}
}
