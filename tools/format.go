package tools

import (
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Supported format names.
const (
	FormatJSON = "json"
	FormatYAML = "yaml"
)

// Sentinel errors for format conversion.
var (
	ErrUnsupportedFormat = errors.New("unsupported format")
)

// Convert transforms data from one format to another.
// Supported: json↔yaml.
func Convert(data []byte, from, to string) ([]byte, error) {
	if from == to {
		return data, nil
	}

	// Parse input
	var intermediate any
	switch from {
	case FormatJSON:
		if err := json.Unmarshal(data, &intermediate); err != nil {
			return nil, fmt.Errorf("parse %s: %w", from, err)
		}
	case FormatYAML:
		if err := yaml.Unmarshal(data, &intermediate); err != nil {
			return nil, fmt.Errorf("parse %s: %w", from, err)
		}
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedFormat, from)
	}

	// Serialize output
	switch to {
	case FormatJSON:
		return json.MarshalIndent(intermediate, "", "  ")
	case FormatYAML:
		return yaml.Marshal(intermediate)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedFormat, to)
	}
}
