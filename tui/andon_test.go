package tui

import "testing"

func TestAndonState_Levels(t *testing.T) {
	tests := []struct {
		level AndonLevel
		str   string
	}{
		{AndonGreen, "green"},
		{AndonYellow, "yellow"},
		{AndonRed, "red"},
		{AndonLevel(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			if got := tt.level.String(); got != tt.str {
				t.Fatalf("AndonLevel(%d).String() = %q, want %q", tt.level, got, tt.str)
			}
		})
	}

	// Verify ordering: Green < Yellow < Red.
	if AndonGreen >= AndonYellow {
		t.Fatal("Green should be less than Yellow")
	}
	if AndonYellow >= AndonRed {
		t.Fatal("Yellow should be less than Red")
	}
}

func TestShouldCordon_OnlyRed(t *testing.T) {
	tests := []struct {
		level AndonLevel
		want  bool
	}{
		{AndonGreen, false},
		{AndonYellow, false},
		{AndonRed, true},
	}
	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := ShouldCordon(tt.level); got != tt.want {
				t.Fatalf("ShouldCordon(%v) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}
