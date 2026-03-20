package tier

import "testing"

func TestTierLevel_String(t *testing.T) {
	tests := []struct {
		level TierLevel
		want  string
	}{
		{Eco, "eco"},
		{Sys, "sys"},
		{Com, "com"},
		{Mod, "mod"},
		{TierLevel(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Fatalf("TierLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestScope_Construction(t *testing.T) {
	s := Scope{Level: Com, Name: "auth-service"}
	if s.Level != Com {
		t.Fatalf("Level = %v, want Com", s.Level)
	}
	if s.Name != "auth-service" {
		t.Fatalf("Name = %q, want %q", s.Name, "auth-service")
	}
}
