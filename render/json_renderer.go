// json_renderer.go — Machine-readable JSON renderer (TSK-433).
package render

import "encoding/json"

type jsonRenderer struct{}

// NewJSONRenderer creates a renderer that outputs machine-readable JSON.
func NewJSONRenderer() Renderer {
	return &jsonRenderer{}
}

func (r *jsonRenderer) Render(g *DataFlowGraph, _ RenderOpts) (string, error) {
	if g == nil {
		return "{}", nil
	}
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
