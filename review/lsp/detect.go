// detect.go — Auto-detect language server from project markers.
package lsp

import (
	"os"
	"path/filepath"
)

// ServerConfig describes how to start a language server.
type ServerConfig struct {
	Language string
	Command  string
	Args     []string
}

// Known language server configurations, checked in priority order.
var knownServers = []struct {
	marker string
	config ServerConfig
}{
	{"go.mod", ServerConfig{Language: "go", Command: "gopls", Args: []string{"serve"}}},
	{"Cargo.toml", ServerConfig{Language: "rust", Command: "rust-analyzer", Args: nil}},
	{"tsconfig.json", ServerConfig{Language: "typescript", Command: "typescript-language-server", Args: []string{"--stdio"}}},
	{"package.json", ServerConfig{Language: "javascript", Command: "typescript-language-server", Args: []string{"--stdio"}}},
	{"pyproject.toml", ServerConfig{Language: "python", Command: "pyright-langserver", Args: []string{"--stdio"}}},
	{"requirements.txt", ServerConfig{Language: "python", Command: "pyright-langserver", Args: []string{"--stdio"}}},
	{"CMakeLists.txt", ServerConfig{Language: "c", Command: "clangd", Args: nil}},
}

// DetectServer looks for project markers in workDir and returns
// the matching language server configuration. Returns zero value if none found.
func DetectServer(workDir string) (ServerConfig, bool) {
	for _, entry := range knownServers {
		path := filepath.Join(workDir, entry.marker)
		if _, err := os.Stat(path); err == nil {
			return entry.config, true
		}
	}
	return ServerConfig{}, false
}
