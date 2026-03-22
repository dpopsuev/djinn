package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Claude Code JSONL entry types we care about.
const (
	claudeTypeUser      = "user"
	claudeTypeAssistant = "assistant"
)

// claudeJSONLEntry is a single line from Claude Code's session JSONL.
type claudeJSONLEntry struct {
	Type      string          `json:"type"`
	Message   *claudeMessage  `json:"message,omitempty"`
	Timestamp string          `json:"timestamp,omitempty"`
}

type claudeMessage struct {
	Role    string              `json:"role"`
	Content json.RawMessage     `json:"content"`
}

type claudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ImportClaudeSession parses a Claude Code JSONL session file and
// creates a Djinn session. Extracts user/assistant text messages,
// skips tool results and internal types.
//
// If tokenBudget > 0, compacts old messages to fit within budget
// (keeps recent messages at full fidelity, summarizes old ones).
// If tokenBudget == 0, imports all messages.
func ImportClaudeSession(jsonlPath string, tokenBudget int) (*Session, error) {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("open session: %w", err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB line buffer

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var raw claudeJSONLEntry
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue // skip malformed lines
		}

		// Only process user and assistant messages
		if raw.Type != claudeTypeUser && raw.Type != claudeTypeAssistant {
			continue
		}
		if raw.Message == nil {
			continue
		}

		// Extract text content from content blocks
		text := extractText(raw.Message.Content)
		if text == "" {
			continue // skip messages with no text (e.g., tool_result only)
		}

		ts, _ := time.Parse(time.RFC3339, raw.Timestamp)

		entries = append(entries, Entry{
			Role:      raw.Message.Role,
			Content:   text,
			Timestamp: ts,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan session: %w", err)
	}

	// Apply compaction if budget is set
	if tokenBudget > 0 && len(entries) > 0 {
		entries = compact(entries, tokenBudget)
	}

	// Create session
	id := fmt.Sprintf("imported-%d", time.Now().Unix())
	sess := New(id, "", "")

	for _, e := range entries {
		sess.Append(e)
	}

	return sess, nil
}

// extractText pulls text content from Claude's content block array.
// Claude stores content as either a string or an array of typed blocks.
func extractText(raw json.RawMessage) string {
	// Try as string first
	var str string
	if json.Unmarshal(raw, &str) == nil {
		return strings.TrimSpace(str)
	}

	// Try as array of content blocks
	var blocks []claudeContentBlock
	if json.Unmarshal(raw, &blocks) == nil {
		var sb strings.Builder
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				if sb.Len() > 0 {
					sb.WriteString("\n")
				}
				sb.WriteString(b.Text)
			}
		}
		return strings.TrimSpace(sb.String())
	}

	return ""
}

// compact keeps recent messages at full fidelity and summarizes
// older messages to fit within the token budget.
func compact(entries []Entry, tokenBudget int) []Entry {
	// Keep at least the last 20 turns at full fidelity
	const keepRecent = 20

	if len(entries) <= keepRecent {
		return entries // nothing to compact
	}

	// Split: old (to compact) + recent (to keep)
	splitIdx := len(entries) - keepRecent
	old := entries[:splitIdx]
	recent := entries[splitIdx:]

	// Summarize old entries: first line of each
	var summaryParts []string
	for _, e := range old {
		first := firstLine(e.Content)
		if first != "" {
			prefix := "user"
			if e.Role == "assistant" {
				prefix = "assistant"
			}
			summaryParts = append(summaryParts, prefix+": "+first)
		}
	}

	summary := strings.Join(summaryParts, "\n")

	// Create compacted entry list
	var result []Entry
	if summary != "" {
		result = append(result, Entry{
			Role:    "user",
			Content: "[Conversation history summary]\n" + summary,
		})
	}
	result = append(result, recent...)

	// If still over budget, trim from the front of recent
	if tokenBudget > 0 {
		for totalTokens(result) > tokenBudget && len(result) > 2 {
			result = result[1:]
		}
	}

	return result
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	const maxLen = 80
	if len(s) > maxLen {
		s = s[:maxLen] + "..."
	}
	return s
}

func totalTokens(entries []Entry) int {
	total := 0
	for _, e := range entries {
		total += estimateTokens(e.Content)
	}
	return total
}
