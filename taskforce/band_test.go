package taskforce

import "testing"

func TestComplexityBand_String(t *testing.T) {
	tests := []struct {
		band ComplexityBand
		want string
	}{
		{Clear, "clear"},
		{Complicated, "complicated"},
		{Complex, "complex"},
		{Chaotic, "chaotic"},
		{ComplexityBand(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.band.String(); got != tt.want {
			t.Fatalf("ComplexityBand(%d).String() = %q, want %q", tt.band, got, tt.want)
		}
	}
}
