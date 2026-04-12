package testkit

import (
	"encoding/json"

	"github.com/dpopsuev/djinn/tools/builtin"
	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

// registryToAnyllmTools converts builtin.Registry tools to anyllm.Tool
// definitions for the LLM provider. The provider needs these schemas
// to offer tool calling.
func registryToAnyllmTools(reg *builtin.Registry) []anyllm.Tool {
	all := reg.All()
	tools := make([]anyllm.Tool, 0, len(all))
	for _, t := range all {
		var params map[string]any
		_ = json.Unmarshal(t.InputSchema(), &params)
		tools = append(tools, anyllm.Tool{
			Type: "function",
			Function: anyllm.Function{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  params,
			},
		})
	}
	return tools
}
