// scratch_paper.go — ScratchPaper tool for agent workstation state (TSK-657).
//
// Agents use this tool to write structured notes to their workstation's
// scratch paper. Actions: write_understanding, add_step, add_risk, append_notes.
// Each action maps to an artifact section on the scratch paper artifact.
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

const (
	scratchPaperToolName = "ScratchPaper"
	scratchPaperToolDesc = "Write structured notes to the workstation scratch paper. " +
		"Actions: write_understanding, add_step, add_risk, append_notes."
)

// ScratchPaperStore is the interface for persisting scratch paper content.
// The workstation wires this to the artifact graph.
type ScratchPaperStore interface {
	WriteSection(scratchID, section, content string) error
	ReadSection(scratchID, section string) (string, error)
}

type scratchPaperInput struct {
	Action    string `json:"action"`     // write_understanding, add_step, add_risk, append_notes
	ScratchID string `json:"scratch_id"` // artifact ID of the scratch paper
	Content   string `json:"content"`    // text to write
}

// ScratchPaperTool writes to the scratch paper artifact sections.
type ScratchPaperTool struct {
	store ScratchPaperStore
	mu    sync.RWMutex

	// In-memory fallback when no store is wired.
	sections map[string]map[string]string // scratchID → section → content
}

// NewScratchPaperTool creates a scratch paper tool.
// If store is nil, uses in-memory storage.
func NewScratchPaperTool(store ScratchPaperStore) *ScratchPaperTool {
	return &ScratchPaperTool{
		store:    store,
		sections: make(map[string]map[string]string),
	}
}

func (t *ScratchPaperTool) Name() string        { return scratchPaperToolName }
func (t *ScratchPaperTool) Description() string { return scratchPaperToolDesc }

func (t *ScratchPaperTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["write_understanding", "add_step", "add_risk", "append_notes"],
				"description": "The scratch paper action to perform"
			},
			"scratch_id": {
				"type": "string",
				"description": "Artifact ID of the scratch paper"
			},
			"content": {
				"type": "string",
				"description": "Content to write"
			}
		},
		"required": ["action", "scratch_id", "content"]
	}`)
}

func (t *ScratchPaperTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var in scratchPaperInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("scratch_paper: %w", err)
	}
	if in.Action == "" || in.ScratchID == "" || in.Content == "" {
		return "", fmt.Errorf("scratch_paper: %w: action, scratch_id, and content are required", ErrEmptyInput)
	}

	section, err := t.actionToSection(in.Action)
	if err != nil {
		return "", err
	}

	// Delegate to store if wired.
	if t.store != nil {
		existing, _ := t.store.ReadSection(in.ScratchID, section)
		newContent := t.mergeContent(in.Action, existing, in.Content)
		if err := t.store.WriteSection(in.ScratchID, section, newContent); err != nil {
			return "", fmt.Errorf("scratch_paper: write: %w", err)
		}
		return fmt.Sprintf("wrote %s to %s", section, in.ScratchID), nil
	}

	// In-memory fallback.
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.sections[in.ScratchID] == nil {
		t.sections[in.ScratchID] = make(map[string]string)
	}

	existing := t.sections[in.ScratchID][section]
	t.sections[in.ScratchID][section] = t.mergeContent(in.Action, existing, in.Content)

	return fmt.Sprintf("wrote %s to %s", section, in.ScratchID), nil
}

// ReadSections returns all in-memory sections for a scratch paper (testing/debug).
func (t *ScratchPaperTool) ReadSections(scratchID string) map[string]string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	sections := t.sections[scratchID]
	if sections == nil {
		return nil
	}

	out := make(map[string]string, len(sections))
	for k, v := range sections {
		out[k] = v
	}
	return out
}

func (t *ScratchPaperTool) actionToSection(action string) (string, error) {
	switch action {
	case "write_understanding":
		return "understanding", nil
	case "add_step":
		return "plan", nil
	case "add_risk":
		return "risks", nil
	case "append_notes":
		return "notes", nil
	default:
		return "", fmt.Errorf("scratch_paper: unknown action %q", action)
	}
}

func (t *ScratchPaperTool) mergeContent(action, existing, newContent string) string {
	switch action {
	case "write_understanding":
		// Replace entirely.
		return newContent
	case "add_step", "add_risk":
		// Append as newline-delimited list.
		if existing == "" {
			return newContent
		}
		return existing + "\n" + newContent
	case "append_notes":
		// Append with separator.
		if existing == "" {
			return newContent
		}
		return existing + "\n" + newContent
	default:
		return newContent
	}
}

// Sections returns a formatted summary of all sections for a scratch paper ID.
// Used by agents to read the scratch paper state after relay.
func (t *ScratchPaperTool) Sections(scratchID string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	sections := t.sections[scratchID]
	if sections == nil {
		return "empty scratch paper"
	}

	var b strings.Builder
	for _, name := range []string{"understanding", "plan", "risks", "notes"} {
		if content, ok := sections[name]; ok && content != "" {
			fmt.Fprintf(&b, "## %s\n%s\n\n", name, content)
		}
	}
	if b.Len() == 0 {
		return "empty scratch paper"
	}
	return b.String()
}
