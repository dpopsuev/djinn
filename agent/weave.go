// weave.go — auto-weave planning context from available slots.
//
// When the agent is in plan mode, AutoWeaveContext enriches the user
// prompt with context from whatever slots are available. It queries
// slots by NAME (WorkTracker, RuleResolver), never by backend MCP
// tool names. The slot router resolves which backend to call.
//
// The weave function doesn't know about Scribe, Lex, or any specific
// MCP server. It knows about slot capabilities.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dpopsuev/djinn/tools/builtin"
)

// SlotQuery defines a planning context query against a slot.
// Each query specifies which tool to call and how to format the result.
type SlotQuery struct {
	// ToolName is the raw tool name the slot exposes (resolved by router).
	// The weave function tries each tool — if unavailable, skips.
	ToolName string
	// BuildInput creates the tool input from the prompt keywords.
	BuildInput func(keywords string) json.RawMessage
	// WrapResult formats the tool output as a context section.
	WrapResult func(result string) string
}

// DefaultSlotQueries returns the planning context queries.
// These use raw tool names that the slot router maps to backends.
// If the tool isn't available for the current role, it's silently skipped.
var DefaultSlotQueries = []SlotQuery{
	{
		// WorkTracker slot — search for related work items.
		ToolName: "artifact",
		BuildInput: func(keywords string) json.RawMessage {
			input, _ := json.Marshal(map[string]any{
				"action": "list",
				"query":  keywords,
				"fields": []string{"id", "title", "status", "kind"},
				"limit":  10,
				"top":    10,
			})
			return input
		},
		WrapResult: func(result string) string {
			if result == "" || result == "(0 artifacts)" {
				return ""
			}
			return fmt.Sprintf("<work-context>\n%s\n</work-context>", result)
		},
	},
	{
		// RuleResolver slot — resolve applicable rules.
		ToolName: "lexicon",
		BuildInput: func(keywords string) json.RawMessage {
			input, _ := json.Marshal(map[string]any{
				"action":   "resolve",
				"keywords": strings.Fields(keywords),
				"budget":   2000,
			})
			return input
		},
		WrapResult: func(result string) string {
			if result == "" {
				return ""
			}
			return fmt.Sprintf("<rules-context>\n%s\n</rules-context>", result)
		},
	},
}

// AutoWeaveContext enriches a user prompt with context from available
// slots. Queries each slot tool — if available, appends the context.
// If not available (wrong role, backend offline), silently skips.
func AutoWeaveContext(ctx context.Context, tools builtin.ToolExecutor, prompt string) string {
	keywords := extractKeywords(prompt)
	if keywords == "" {
		return prompt
	}

	var sections []string
	for _, q := range DefaultSlotQueries {
		// Try to execute — the router will deny if the tool isn't
		// available for the current role. That's fine, skip it.
		input := q.BuildInput(keywords)
		result, err := tools.Execute(ctx, q.ToolName, input)
		if err != nil {
			continue
		}
		if section := q.WrapResult(result); section != "" {
			sections = append(sections, section)
		}
	}

	if len(sections) == 0 {
		return prompt
	}

	return fmt.Sprintf("<planning-context>\n%s\n</planning-context>\n\n%s",
		strings.Join(sections, "\n\n"), prompt)
}

// extractKeywords pulls meaningful words from a prompt for search queries.
func extractKeywords(prompt string) string {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "shall": true, "can": true, "need": true,
		"to": true, "of": true, "in": true, "for": true, "on": true,
		"with": true, "at": true, "by": true, "from": true, "as": true,
		"into": true, "through": true, "about": true, "this": true, "that": true,
		"it": true, "its": true, "i": true, "we": true, "me": true, "my": true,
		"and": true, "or": true, "but": true, "not": true, "no": true,
		"all": true, "how": true, "what": true, "which": true, "let": true,
		"please": true, "implement": true, "create": true, "make": true,
	}

	words := strings.Fields(strings.ToLower(prompt))
	var keywords []string
	for _, w := range words {
		w = strings.Trim(w, ".,!?()[]{}\"'")
		if len(w) < 3 || stopWords[w] {
			continue
		}
		keywords = append(keywords, w)
		if len(keywords) >= 5 {
			break
		}
	}
	return strings.Join(keywords, " ")
}
