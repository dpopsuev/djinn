package litmus

import (
	"testing"
	"time"
)

// RunLitmusContract runs the Liskov contract test suite against any Litmus.
func RunLitmusContract(t *testing.T, factory func(t *testing.T) Litmus) {
	t.Helper()

	t.Run("RecordTest_and_TestResult", func(t *testing.T) {
		l := factory(t)
		l.RecordTest("auth/", TestResultEntry{
			Pass:       true,
			Total:      10,
			SourceHash: "abc123",
			Timestamp:  time.Now(),
		})

		r, ok := l.TestResult("auth/")
		if !ok {
			t.Fatal("TestResult should return cached entry")
		}
		if !r.Pass {
			t.Error("should be pass")
		}
		if r.Total != 10 {
			t.Errorf("total = %d, want 10", r.Total)
		}
		if r.SourceHash != "abc123" {
			t.Errorf("hash = %q", r.SourceHash)
		}
	})

	t.Run("RecordBuild_and_BuildResult", func(t *testing.T) {
		l := factory(t)
		l.RecordBuild("main/", BuildResultEntry{
			Pass:       false,
			Errors:     []string{"undefined: Foo"},
			SourceHash: "def456",
			Timestamp:  time.Now(),
		})

		r, ok := l.BuildResult("main/")
		if !ok {
			t.Fatal("BuildResult should return cached entry")
		}
		if r.Pass {
			t.Error("should fail")
		}
		if len(r.Errors) != 1 {
			t.Errorf("errors = %d, want 1", len(r.Errors))
		}
	})

	t.Run("TestResult_miss", func(t *testing.T) {
		l := factory(t)
		_, ok := l.TestResult("nonexistent/")
		if ok {
			t.Fatal("should miss on uncached package")
		}
	})

	t.Run("Invalidate_removes_package", func(t *testing.T) {
		l := factory(t)
		l.RecordTest("pkg/", TestResultEntry{Pass: true, SourceHash: "x"})
		l.RecordBuild("pkg/", BuildResultEntry{Pass: true, SourceHash: "x"})

		l.Invalidate("pkg/")

		if _, ok := l.TestResult("pkg/"); ok {
			t.Error("test result should be gone after invalidate")
		}
		if _, ok := l.BuildResult("pkg/"); ok {
			t.Error("build result should be gone after invalidate")
		}
	})

	t.Run("InvalidateAll_clears_everything", func(t *testing.T) {
		l := factory(t)
		l.RecordTest("a/", TestResultEntry{Pass: true, SourceHash: "1"})
		l.RecordTest("b/", TestResultEntry{Pass: true, SourceHash: "2"})

		l.InvalidateAll()

		if _, ok := l.TestResult("a/"); ok {
			t.Error("a/ should be gone")
		}
		if _, ok := l.TestResult("b/"); ok {
			t.Error("b/ should be gone")
		}
	})

	t.Run("RecordTest_overwrites", func(t *testing.T) {
		l := factory(t)
		l.RecordTest("pkg/", TestResultEntry{Pass: true, SourceHash: "old"})
		l.RecordTest("pkg/", TestResultEntry{Pass: false, SourceHash: "new", Failed: 3})

		r, _ := l.TestResult("pkg/")
		if r.Pass {
			t.Error("should be overwritten to fail")
		}
		if r.SourceHash != "new" {
			t.Errorf("hash = %q, want new", r.SourceHash)
		}
		if r.Failed != 3 {
			t.Errorf("failed = %d, want 3", r.Failed)
		}
	})
}

// --- Run contract against StubLitmus ---

func TestStubLitmus_Contract(t *testing.T) {
	RunLitmusContract(t, func(_ *testing.T) Litmus {
		return NewStubLitmus()
	})
}
