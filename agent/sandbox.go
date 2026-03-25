// sandbox.go — path translation for jail-mounted workspaces.
// When an agent runs inside a sandbox, file paths in tool inputs
// need translation from host paths to jail mount paths.
package agent

import (
	"encoding/json"
	"strings"
)

// TranslatePath rewrites file_path/path fields in tool input from
// host workspace path to jail mount point.
// Example: /home/user/project/main.go → /workspace/main.go
func TranslatePath(input json.RawMessage, hostWorkDir, jailMount string) json.RawMessage {
	if len(input) == 0 || hostWorkDir == "" || jailMount == "" {
		return input
	}

	var params map[string]any
	if err := json.Unmarshal(input, &params); err != nil {
		return input
	}

	changed := false
	for _, key := range []string{"file_path", "path"} {
		if v, ok := params[key].(string); ok && strings.HasPrefix(v, hostWorkDir) {
			params[key] = jailMount + strings.TrimPrefix(v, hostWorkDir)
			changed = true
		}
	}

	if !changed {
		return input
	}

	out, err := json.Marshal(params)
	if err != nil {
		return input
	}
	return out
}
