package gate

import "testing"

func TestGateConfig_Construction(t *testing.T) {
	cfg := GateConfig{
		Name:     "lint",
		Severity: "blocking",
		Thresholds: map[string]float64{
			"coverage": 80.0,
		},
	}
	if cfg.Name != "lint" {
		t.Fatalf("Name = %q, want %q", cfg.Name, "lint")
	}
	if cfg.Severity != "blocking" {
		t.Fatalf("Severity = %q, want %q", cfg.Severity, "blocking")
	}
	if cfg.Thresholds["coverage"] != 80.0 {
		t.Fatalf("Thresholds[coverage] = %f, want 80.0", cfg.Thresholds["coverage"])
	}
}
