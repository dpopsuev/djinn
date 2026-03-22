package session

import "testing"

func TestSearch_EmptyQuery(t *testing.T) {
	summaries := []SessionSummary{
		{Name: "alpha", Model: "claude"},
		{Name: "beta", Model: "ollama"},
	}
	results := Search(summaries, "")
	if len(results) != 2 {
		t.Fatalf("empty query should return all, got %d", len(results))
	}
}

func TestSearch_ByName(t *testing.T) {
	summaries := []SessionSummary{
		{Name: "djinn-dev"},
		{Name: "misbah-test"},
		{Name: "djinn-prod"},
	}
	results := Search(summaries, "djinn")
	if len(results) != 2 {
		t.Fatalf("got %d, want 2", len(results))
	}
}

func TestSearch_ByModel(t *testing.T) {
	summaries := []SessionSummary{
		{Name: "a", Model: "claude-opus-4-6"},
		{Name: "b", Model: "qwen2.5"},
	}
	results := Search(summaries, "opus")
	if len(results) != 1 || results[0].Name != "a" {
		t.Fatalf("results = %v", results)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	summaries := []SessionSummary{
		{Name: "DjinnDev"},
	}
	results := Search(summaries, "djinndev")
	if len(results) != 1 {
		t.Fatal("should be case insensitive")
	}
}

func TestSearch_NoMatch(t *testing.T) {
	summaries := []SessionSummary{
		{Name: "alpha"},
	}
	results := Search(summaries, "zzz")
	if len(results) != 0 {
		t.Fatal("should return empty")
	}
}

func TestSearch_ByWorkDir(t *testing.T) {
	summaries := []SessionSummary{
		{Name: "a", WorkDir: "/home/user/Workspace/djinn"},
		{Name: "b", WorkDir: "/home/user/Workspace/misbah"},
	}
	results := Search(summaries, "misbah")
	if len(results) != 1 || results[0].Name != "b" {
		t.Fatalf("results = %v", results)
	}
}
