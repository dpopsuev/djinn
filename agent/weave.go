package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dpopsuev/djinn/tools/builtin"
)

// AutoWeaveContext enriches a user prompt with context from connected
// MCP tools (Scribe, Lex) when in plan mode. This eliminates the manual
// exploration phase that wastes 20K+ tokens per planning cycle.
//
// Only runs when:
// 1. Mode is Plan (tools disabled, thinking only)
// 2. MCP tools are available in the registry
//
// Queries are lightweight: 2 tool calls at most.
func AutoWeaveContext(ctx context.Context, tools *builtin.Registry, prompt string) string {
	var sections []string

	// Query Scribe for related specs/tasks
	if scribeCtx := queryScribe(ctx, tools, prompt); scribeCtx != "" {
		sections = append(sections, scribeCtx)
	}

	// Query Lex for applicable rules
	if lexCtx := queryLex(ctx, tools, prompt); lexCtx != "" {
		sections = append(sections, lexCtx)
	}

	if len(sections) == 0 {
		return prompt
	}

	return fmt.Sprintf("<planning-context>\n%s\n</planning-context>\n\n%s",
		strings.Join(sections, "\n\n"), prompt)
}

func queryScribe(ctx context.Context, tools *builtin.Registry, prompt string) string {
	// Look for mcp__scribe__artifact tool
	_, err := tools.Get("mcp__scribe__artifact")
	if err != nil {
		return ""
	}

	// Extract keywords from prompt for search
	keywords := extractKeywords(prompt)
	input, _ := json.Marshal(map[string]any{
		"action": "list",
		"query":  keywords,
		"fields": []string{"id", "title", "status", "kind"},
		"limit":  10,
		"top":    10,
	})

	result, err := tools.Execute(ctx, "mcp__scribe__artifact", input)
	if err != nil {
		return ""
	}

	if result == "" || result == "(0 artifacts)" {
		return ""
	}

	return fmt.Sprintf("<scribe-context>\n%s\n</scribe-context>", result)
}

func queryLex(ctx context.Context, tools *builtin.Registry, prompt string) string {
	// Look for mcp__lex__lexicon tool
	_, err := tools.Get("mcp__lex__lexicon")
	if err != nil {
		return ""
	}

	keywords := extractKeywords(prompt)
	input, _ := json.Marshal(map[string]any{
		"action":   "resolve",
		"keywords": strings.Fields(keywords),
		"budget":   2000,
	})

	result, err := tools.Execute(ctx, "mcp__lex__lexicon", input)
	if err != nil {
		return ""
	}

	if result == "" {
		return ""
	}

	return fmt.Sprintf("<lex-context>\n%s\n</lex-context>", result)
}

// extractKeywords pulls meaningful words from a prompt for search queries.
func extractKeywords(prompt string) string {
	// Simple: take the first 5 non-stop words
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
