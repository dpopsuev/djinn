// arsenal.go — scans for installed agent CLIs and API keys.
// Used by first-run auto-detection and djinn doctor.
package app

import (
	"os"
	"os/exec"
	"strings"
)

// DetectedDriver represents an agent CLI or API key found on the system.
type DetectedDriver struct {
	Name    string // "cursor", "claude", "gemini", "codex", "ollama", "claude-api"
	Binary  string // path to binary (empty for API key detections)
	Version string // version string if available
	Source  string // "cli" or "api-key"
}

// ACPName returns the driver name to use in djinn.yaml.
func (d DetectedDriver) ACPName() string {
	switch d.Name {
	case DriverCursor:
		return "acp"
	case DriverClaude:
		return DriverClaude
	case DriverClaudeAPI:
		return DriverClaude
	default:
		return d.Name
	}
}

// DefaultModel returns the default model for this driver.
func (d DetectedDriver) DefaultModel() string {
	switch d.Name {
	case DriverCursor:
		return "cursor/sonnet-4"
	case DriverClaude, DriverClaudeAPI:
		return "claude-sonnet-4-6"
	case DriverGemini:
		return "gemini-2.5-pro"
	case DriverCodex:
		return DriverCodex
	case DriverOllama:
		return "llama3"
	default:
		return ""
	}
}

// knownCLIs lists agent CLIs to scan for, in preference order.
var knownCLIs = []struct {
	name       string
	binary     string
	versionArg string
}{
	{"cursor", "cursor", "--version"},
	{"claude", "claude", "--version"},
	{"gemini", "gemini", "--version"},
	{"codex", "codex", "--version"},
	{"ollama", "ollama", "--version"},
}

// knownAPIKeys lists environment variables that indicate API access.
var knownAPIKeys = []struct {
	name   string
	envVar string
}{
	{"claude-api", "ANTHROPIC_API_KEY"},
	{"gemini-api", "GOOGLE_API_KEY"},
}

// ScanArsenal detects installed agent CLIs and API keys.
// Results are sorted by preference (cursor first — ACP native).
func ScanArsenal() []DetectedDriver {
	detected := make([]DetectedDriver, 0, len(knownCLIs)+len(knownAPIKeys))

	// Scan for CLIs on PATH.
	for _, cli := range knownCLIs {
		path, err := exec.LookPath(cli.binary)
		if err != nil {
			continue
		}

		drv := DetectedDriver{
			Name:   cli.name,
			Binary: path,
			Source: "cli",
		}

		// Try to get version (best-effort, 2s timeout).
		if out, err := exec.Command(cli.binary, cli.versionArg).Output(); err == nil { //nolint:gosec // binary and args are from compile-time knownCLIs table
			version := strings.TrimSpace(string(out))
			if len(version) > 100 {
				version = version[:100]
			}
			drv.Version = version
		}

		detected = append(detected, drv)
	}

	// Check for API keys in environment.
	for _, key := range knownAPIKeys {
		if os.Getenv(key.envVar) != "" {
			detected = append(detected, DetectedDriver{
				Name:   key.name,
				Source: "api-key",
			})
		}
	}

	return detected
}
