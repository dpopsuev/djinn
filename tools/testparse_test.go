package tools

import (
	"strings"
	"testing"
)

func jsonLines(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func TestParseGoTestJSON_PassingTests(t *testing.T) {
	input := jsonLines(
		`{"Time":"2026-01-01T00:00:00Z","Action":"run","Package":"example.com/p","Test":"TestA"}`,
		`{"Time":"2026-01-01T00:00:01Z","Action":"output","Package":"example.com/p","Test":"TestA","Output":"--- PASS: TestA\n"}`,
		`{"Time":"2026-01-01T00:00:01Z","Action":"pass","Package":"example.com/p","Test":"TestA","Elapsed":0.1}`,
		`{"Time":"2026-01-01T00:00:01Z","Action":"run","Package":"example.com/p","Test":"TestB"}`,
		`{"Time":"2026-01-01T00:00:01Z","Action":"pass","Package":"example.com/p","Test":"TestB","Elapsed":0.05}`,
		`{"Time":"2026-01-01T00:00:01Z","Action":"output","Package":"example.com/p","Output":"ok  \texample.com/p\t0.15s\tcoverage: 82.5% of statements\n"}`,
		`{"Time":"2026-01-01T00:00:01Z","Action":"pass","Package":"example.com/p","Elapsed":0.15}`,
	)

	result, err := ParseGoTestJSON(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseGoTestJSON: %v", err)
	}
	if result.Passed != 2 {
		t.Fatalf("Passed = %d, want 2", result.Passed)
	}
	if result.Failed != 0 {
		t.Fatalf("Failed = %d, want 0", result.Failed)
	}
	if result.Suite != "example.com/p" {
		t.Fatalf("Suite = %q, want example.com/p", result.Suite)
	}
}

func TestParseGoTestJSON_FailingTest(t *testing.T) {
	input := jsonLines(
		`{"Time":"2026-01-01T00:00:00Z","Action":"run","Package":"example.com/p","Test":"TestFail"}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"output","Package":"example.com/p","Test":"TestFail","Output":"    fail_test.go:10: assertion failed\n"}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"fail","Package":"example.com/p","Test":"TestFail","Elapsed":0.01}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"fail","Package":"example.com/p","Elapsed":0.01}`,
	)

	result, err := ParseGoTestJSON(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseGoTestJSON: %v", err)
	}
	if result.Failed != 1 {
		t.Fatalf("Failed = %d, want 1", result.Failed)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("Failures = %d, want 1", len(result.Failures))
	}
	if result.Failures[0].Name != "TestFail" {
		t.Fatalf("failure name = %q, want TestFail", result.Failures[0].Name)
	}
	if !strings.Contains(result.Failures[0].Output, "assertion failed") {
		t.Fatalf("failure output = %q, expected 'assertion failed'", result.Failures[0].Output)
	}
}

func TestParseGoTestJSON_SkippedTest(t *testing.T) {
	input := jsonLines(
		`{"Time":"2026-01-01T00:00:00Z","Action":"run","Package":"example.com/p","Test":"TestSkip"}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"skip","Package":"example.com/p","Test":"TestSkip","Elapsed":0.0}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"pass","Package":"example.com/p","Elapsed":0.0}`,
	)

	result, err := ParseGoTestJSON(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseGoTestJSON: %v", err)
	}
	if result.Skipped != 1 {
		t.Fatalf("Skipped = %d, want 1", result.Skipped)
	}
}

func TestParseGoTestJSON_Coverage(t *testing.T) {
	input := jsonLines(
		`{"Time":"2026-01-01T00:00:00Z","Action":"run","Package":"example.com/p","Test":"TestA"}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"pass","Package":"example.com/p","Test":"TestA","Elapsed":0.1}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"output","Package":"example.com/p","Output":"coverage: 91.7% of statements\n"}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"pass","Package":"example.com/p","Elapsed":0.1}`,
	)

	result, err := ParseGoTestJSON(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseGoTestJSON: %v", err)
	}
	if result.Coverage != 91.7 {
		t.Fatalf("Coverage = %f, want 91.7", result.Coverage)
	}
}

func TestParseGoTestJSON_EmptyInput(t *testing.T) {
	result, err := ParseGoTestJSON(strings.NewReader(""))
	if err != nil {
		t.Fatalf("ParseGoTestJSON: %v", err)
	}
	if result.Total() != 0 {
		t.Fatalf("Total = %d, want 0", result.Total())
	}
}

func TestParseGoTestJSON_MixedPassFail(t *testing.T) {
	input := jsonLines(
		`{"Time":"2026-01-01T00:00:00Z","Action":"run","Package":"example.com/p","Test":"TestOK"}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"pass","Package":"example.com/p","Test":"TestOK","Elapsed":0.01}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"run","Package":"example.com/p","Test":"TestBad"}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"output","Package":"example.com/p","Test":"TestBad","Output":"error!\n"}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"fail","Package":"example.com/p","Test":"TestBad","Elapsed":0.02}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"run","Package":"example.com/p","Test":"TestSkipped"}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"skip","Package":"example.com/p","Test":"TestSkipped","Elapsed":0.0}`,
		`{"Time":"2026-01-01T00:00:00Z","Action":"fail","Package":"example.com/p","Elapsed":0.05}`,
	)

	result, err := ParseGoTestJSON(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseGoTestJSON: %v", err)
	}
	if result.Passed != 1 {
		t.Fatalf("Passed = %d, want 1", result.Passed)
	}
	if result.Failed != 1 {
		t.Fatalf("Failed = %d, want 1", result.Failed)
	}
	if result.Skipped != 1 {
		t.Fatalf("Skipped = %d, want 1", result.Skipped)
	}
	if result.Total() != 3 {
		t.Fatalf("Total = %d, want 3", result.Total())
	}
}

func TestParseCoverage_ValidFormats(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"coverage: 100.0% of statements", 100.0},
		{"coverage: 0.0% of statements", 0.0},
		{"coverage: 42.5% of statements\n", 42.5},
		{"ok  example.com/p 0.1s coverage: 73.2% of statements", 73.2},
		{"no coverage here", 0},
	}
	for _, tc := range tests {
		got := parseCoverage(tc.input)
		if got != tc.want {
			t.Errorf("parseCoverage(%q) = %f, want %f", tc.input, got, tc.want)
		}
	}
}
