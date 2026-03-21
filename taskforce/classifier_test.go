package taskforce

import (
	"testing"

	"github.com/dpopsuev/djinn/tier"
)

func TestHeuristicClassifier(t *testing.T) {
	c := NewHeuristicClassifier()

	tests := []struct {
		name      string
		scopes    []tier.Scope
		fileCount int
		repoCount int
		want      ComplexityBand
	}{
		{
			name:      "single file, single scope = Clear",
			scopes:    []tier.Scope{{Level: tier.Mod, Name: "auth"}},
			fileCount: 1,
			repoCount: 1,
			want:      Clear,
		},
		{
			name:      "few files, single scope = Complicated",
			scopes:    []tier.Scope{{Level: tier.Com, Name: "auth"}},
			fileCount: 5,
			repoCount: 1,
			want:      Complicated,
		},
		{
			name:      "many files = Complex",
			scopes:    []tier.Scope{{Level: tier.Sys, Name: "api"}, {Level: tier.Com, Name: "auth"}},
			fileCount: 20,
			repoCount: 1,
			want:      Complex,
		},
		{
			name:      "multi-repo = Complex",
			scopes:    []tier.Scope{{Level: tier.Eco, Name: "workspace"}},
			fileCount: 3,
			repoCount: 2,
			want:      Complex,
		},
		{
			name:      "zero files, single scope = Clear",
			scopes:    []tier.Scope{{Level: tier.Mod, Name: "fix"}},
			fileCount: 0,
			repoCount: 1,
			want:      Clear,
		},
	}

	for _, tt := range tests {
		got := c.Classify(tt.scopes, tt.fileCount, tt.repoCount)
		if got != tt.want {
			t.Fatalf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}
