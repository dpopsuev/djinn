package stubs

import (
	"testing"
)

func TestFeedbackStub_QueryReturnsValues(t *testing.T) {
	f := NewFeedbackStub(5.0, map[string]float64{
		"error_rate": 0.3,
		"latency":    42.0,
	})

	if got := f.Query("error_rate"); got != 0.3 {
		t.Fatalf("Query(error_rate) = %f, want 0.3", got)
	}
	if got := f.Query("latency"); got != 42.0 {
		t.Fatalf("Query(latency) = %f, want 42.0", got)
	}
	if got := f.Query("unknown"); got != 0.0 {
		t.Fatalf("Query(unknown) = %f, want 0.0", got)
	}
}

func TestFeedbackStub_ThresholdEmitsAlert(t *testing.T) {
	f := NewFeedbackStub(5.0, map[string]float64{"error_rate": 0.3})

	f.SetMetric("error_rate", 7.2)

	select {
	case alert := <-f.Alerts():
		if alert.Metric != "error_rate" {
			t.Fatalf("alert.Metric = %q, want %q", alert.Metric, "error_rate")
		}
		if alert.Value != 7.2 {
			t.Fatalf("alert.Value = %f, want 7.2", alert.Value)
		}
		if alert.Level != "critical" {
			t.Fatalf("alert.Level = %q, want %q", alert.Level, "critical")
		}
	default:
		t.Fatal("expected alert, got none")
	}

	if !f.Fired() {
		t.Fatal("Fired() = false after threshold breach")
	}
}

func TestFeedbackStub_NoDoubleAlert(t *testing.T) {
	f := NewFeedbackStub(5.0, map[string]float64{"error_rate": 0.3})

	f.SetMetric("error_rate", 7.2)
	<-f.Alerts() // consume first

	f.SetMetric("error_rate", 8.0) // still above, no new alert

	select {
	case <-f.Alerts():
		t.Fatal("unexpected second alert")
	default:
	}
}

func TestFeedbackStub_RecoveryDoesNotAlert(t *testing.T) {
	f := NewFeedbackStub(5.0, map[string]float64{"error_rate": 0.3})

	f.SetMetric("error_rate", 7.2)
	<-f.Alerts()

	f.RecoverMetric("error_rate", 0.1)

	if f.Fired() {
		t.Fatal("Fired() = true after recovery")
	}
	if got := f.Query("error_rate"); got != 0.1 {
		t.Fatalf("Query(error_rate) = %f after recovery, want 0.1", got)
	}
}

func TestFeedbackStub_RecoveryThenRefire(t *testing.T) {
	f := NewFeedbackStub(5.0, map[string]float64{"error_rate": 0.3})

	f.SetMetric("error_rate", 7.2)
	<-f.Alerts()

	f.RecoverMetric("error_rate", 0.1)

	f.SetMetric("error_rate", 6.0)
	select {
	case alert := <-f.Alerts():
		if alert.Value != 6.0 {
			t.Fatalf("refire alert.Value = %f, want 6.0", alert.Value)
		}
	default:
		t.Fatal("expected refire alert")
	}
}
