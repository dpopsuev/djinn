package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/tools/builtin"
)

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		prompt string
		want   int // expected keyword count (up to 5)
	}{
		{"implement the VCS port with quality gates", 4},
		{"plan the architecture", 2},
		{"", 0},
		{"the a an is are", 0},
		{"one two three four five six seven", 5},
	}
	for _, tt := range tests {
		result := extractKeywords(tt.prompt)
		words := strings.Fields(result)
		if len(words) != tt.want {
			t.Errorf("extractKeywords(%q) = %d words %q, want %d", tt.prompt, len(words), result, tt.want)
		}
	}
}

func TestAutoWeaveContext_NoMCPTools(t *testing.T) {
	tools := builtin.NewRegistry()
	prompt := "plan the VCS implementation"

	result := AutoWeaveContext(context.Background(), tools, prompt)
	if result != prompt {
		t.Fatalf("without MCP tools, prompt should be unchanged, got %q", result)
	}
}

func TestAutoWeaveContext_PreservesPrompt(t *testing.T) {
	tools := builtin.NewRegistry()
	prompt := "design the logging architecture"

	result := AutoWeaveContext(context.Background(), tools, prompt)
	if !strings.Contains(result, prompt) {
		t.Fatal("original prompt should be preserved")
	}
}
