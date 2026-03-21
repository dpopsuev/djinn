package watchdog

import "testing"

func TestInterfaceSatisfaction(t *testing.T) {
	var _ Watchdog = (*BudgetWatchdog)(nil)
	var _ Watchdog = (*DeadlockWatchdog)(nil)
	var _ Watchdog = (*SecurityWatchdog)(nil)
	var _ Watchdog = (*QualityWatchdog)(nil)
	var _ Watchdog = (*DriftWatchdog)(nil)
}

func TestCategoryConstants(t *testing.T) {
	cats := []string{CategorySecurity, CategoryBudget, CategoryDeadlock, CategoryQuality, CategoryDrift}
	seen := make(map[string]bool)
	for _, c := range cats {
		if c == "" {
			t.Fatal("category constant is empty")
		}
		if seen[c] {
			t.Fatalf("duplicate category: %q", c)
		}
		seen[c] = true
	}
}

func TestStubWatchdogs_NameAndCategory(t *testing.T) {
	tests := []struct {
		w    Watchdog
		name string
		cat  string
	}{
		{NewSecurityWatchdog(), "security-watchdog", CategorySecurity},
		{NewQualityWatchdog(), "quality-watchdog", CategoryQuality},
		{NewDriftWatchdog(), "drift-watchdog", CategoryDrift},
	}
	for _, tt := range tests {
		if tt.w.Name() != tt.name {
			t.Fatalf("%T.Name() = %q, want %q", tt.w, tt.w.Name(), tt.name)
		}
		if tt.w.Category() != tt.cat {
			t.Fatalf("%T.Category() = %q, want %q", tt.w, tt.w.Category(), tt.cat)
		}
	}
}
