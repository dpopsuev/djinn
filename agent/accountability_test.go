package agent

import "testing"

func TestAccountabilityMetric_ComplianceRate(t *testing.T) {
	m := &AccountabilityMetric{Kind: MetricArchive, Attempted: 10, Approved: 8, Denied: 2}
	if rate := m.ComplianceRate(); rate != 0.8 {
		t.Fatalf("ComplianceRate = %f, want 0.8", rate)
	}
}

func TestAccountabilityMetric_ZeroAttempts(t *testing.T) {
	m := &AccountabilityMetric{Kind: MetricArchive}
	if rate := m.ComplianceRate(); rate != 1.0 {
		t.Fatalf("ComplianceRate with 0 attempts = %f, want 1.0", rate)
	}
}

func TestAgentAccountability_RecordApproveDeny(t *testing.T) {
	a := NewAgentAccountability("executor-1")

	a.Record(MetricArchive)
	a.Record(MetricArchive)
	a.Record(MetricArchive)
	a.Approve(MetricArchive)
	a.Deny(MetricArchive)

	m := a.Metric(MetricArchive)
	if m == nil {
		t.Fatal("expected metric for archive")
	}
	if m.Attempted != 3 {
		t.Fatalf("Attempted = %d, want 3", m.Attempted)
	}
	if m.Approved != 1 {
		t.Fatalf("Approved = %d, want 1", m.Approved)
	}
	if m.Denied != 1 {
		t.Fatalf("Denied = %d, want 1", m.Denied)
	}
}

func TestAgentAccountability_Rate(t *testing.T) {
	a := NewAgentAccountability("executor-1")

	// No attempts = 1.0
	if rate := a.Rate(MetricArchive); rate != 1.0 {
		t.Fatalf("Rate with no attempts = %f, want 1.0", rate)
	}

	a.Record(MetricArchive)
	a.Record(MetricArchive)
	a.Approve(MetricArchive)

	if rate := a.Rate(MetricArchive); rate != 0.5 {
		t.Fatalf("Rate = %f, want 0.5", rate)
	}
}

func TestAgentAccountability_OverallRate(t *testing.T) {
	a := NewAgentAccountability("executor-1")

	// No metrics = 1.0
	if rate := a.OverallRate(); rate != 1.0 {
		t.Fatalf("OverallRate with no metrics = %f, want 1.0", rate)
	}

	// Archive: 2 attempted, 2 approved = 1.0
	a.Record(MetricArchive)
	a.Record(MetricArchive)
	a.Approve(MetricArchive)
	a.Approve(MetricArchive)

	// Deferral: 2 attempted, 1 approved = 0.5
	a.Record(MetricDeferral)
	a.Record(MetricDeferral)
	a.Approve(MetricDeferral)

	if rate := a.OverallRate(); rate != 0.75 {
		t.Fatalf("OverallRate = %f, want 0.75", rate)
	}
}

func TestAgentAccountability_MetricReturnsNilForUntracked(t *testing.T) {
	a := NewAgentAccountability("executor-1")
	if m := a.Metric(MetricBlocker); m != nil {
		t.Fatal("expected nil for untracked metric")
	}
}

func TestAgentAccountability_MetricReturnsCopy(t *testing.T) {
	a := NewAgentAccountability("executor-1")
	a.Record(MetricArchive)
	a.Approve(MetricArchive)

	m := a.Metric(MetricArchive)
	m.Attempted = 999 // mutate the copy

	original := a.Metric(MetricArchive)
	if original.Attempted == 999 {
		t.Fatal("Metric should return a copy, not a reference")
	}
}

func TestAgentAccountability_Kinds(t *testing.T) {
	a := NewAgentAccountability("executor-1")
	a.Record(MetricArchive)
	a.Record(MetricBlocker)

	kinds := a.Kinds()
	if len(kinds) != 2 {
		t.Fatalf("Kinds = %d, want 2", len(kinds))
	}
}

func TestAgentAccountability_AgentID(t *testing.T) {
	a := NewAgentAccountability("gensec")
	if a.AgentID() != "gensec" {
		t.Fatalf("AgentID = %q, want gensec", a.AgentID())
	}
}

func TestMetricKind_Constants(t *testing.T) {
	kinds := []MetricKind{MetricArchive, MetricDeferral, MetricBlocker, MetricOverride, MetricEscalation}
	seen := make(map[MetricKind]bool)
	for _, k := range kinds {
		if k == "" {
			t.Fatal("MetricKind constant is empty")
		}
		if seen[k] {
			t.Fatalf("duplicate MetricKind: %q", k)
		}
		seen[k] = true
	}
}
