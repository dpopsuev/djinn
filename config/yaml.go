package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DumpYAML serializes all registered components to YAML.
func (r *Registry) DumpYAML() ([]byte, error) {
	return yaml.Marshal(r.Dump())
}

// LoadYAML parses YAML and applies it to registered components.
func (r *Registry) LoadYAML(data []byte) error {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}
	return r.Load(raw)
}

// LoadFile reads a YAML file and applies it to registered components.
func (r *Registry) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	return r.LoadYAML(data)
}
