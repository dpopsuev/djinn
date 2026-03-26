// fileref.go — @file reference preprocessing.
//
// Extracts @path/to/file references from prompts and injects file
// content as XML blocks. Relative paths resolved against workDir.
package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// preprocessFileRefs finds @path/to/file references in the prompt and
// injects file content as context blocks.
func preprocessFileRefs(prompt, workDir string) string {
	words := strings.Fields(prompt)
	var refs []string
	for _, w := range words {
		clean := strings.TrimRight(w, ".,;:!?")
		if strings.HasPrefix(clean, "@") && len(clean) > 1 {
			ref := strings.TrimPrefix(clean, "@")
			// Skip email-like patterns (contains @ mid-word).
			if strings.Contains(ref, "@") {
				continue
			}
			refs = append(refs, ref)
		}
	}
	if len(refs) == 0 {
		return prompt
	}

	var sb strings.Builder
	for _, ref := range refs {
		path := ref
		if !filepath.IsAbs(path) {
			path = filepath.Join(workDir, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			sb.WriteString(fmt.Sprintf("\n<file path=%q error=%q />\n", ref, err.Error()))
			continue
		}
		sb.WriteString(fmt.Sprintf("\n<file path=%q>\n%s\n</file>\n", ref, string(data)))
	}
	return prompt + sb.String()
}
