// convert.go — converts legacy agent.HookConfig to unified Hook format.
//
// Backward compatibility adapter. ConvertLegacy maps old-style shell hooks
// (pre_tool_use / post_tool_use with command + tools list) to the new
// unified Hook type. Deleted when agent/hook.go is removed (TSK-1076).
//
// GOL-161, TSK-1072
package hook

import "fmt"

// LegacyHookConfig matches agent.HookConfig for conversion without importing agent/.
type LegacyHookConfig struct {
	Command string
	Tools   []string
}

// ConvertLegacy maps old-style shell hooks to unified Hook format.
func ConvertLegacy(pre, post []LegacyHookConfig) []Hook {
	hooks := make([]Hook, 0, len(pre)+len(post))
	for i, h := range pre {
		toolMatch := "*"
		if len(h.Tools) == 1 {
			toolMatch = h.Tools[0]
		}
		hooks = append(hooks, Hook{
			Name:   fmt.Sprintf("legacy-pre-%d", i),
			On:     PhasePreToolUse,
			Match:  Matcher{Tool: toolMatch},
			Action: Action{Shell: h.Command},
		})
	}
	for i, h := range post {
		toolMatch := "*"
		if len(h.Tools) == 1 {
			toolMatch = h.Tools[0]
		}
		hooks = append(hooks, Hook{
			Name:   fmt.Sprintf("legacy-post-%d", i),
			On:     PhasePostToolUse,
			Match:  Matcher{Tool: toolMatch},
			Action: Action{Shell: h.Command},
		})
	}
	return hooks
}
