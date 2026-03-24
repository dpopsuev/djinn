// sandbox_backend_test.go — acceptance tests for sandboxed backend execution.
//
// Tests verify that the sandbox lifecycle integrates with the REPL:
// create sandbox → start backend inside → destroy on exit.
// Misbah tests are skipped if the daemon isn't running.
package acceptance

import (
	"testing"

	"github.com/dpopsuev/djinn/sandbox"
)

func TestSandbox_NoSandbox_WorksWithout(t *testing.T) {
	// When no --sandbox flag is set, Djinn works without isolation.
	// The sandbox package should have no backends registered by default
	// (they register via init() only when imported).
	avail := sandbox.Available()
	// This test just verifies the registry doesn't panic on empty.
	_ = avail
}

func TestSandbox_FailFastIfDeclaredButMissing(t *testing.T) {
	// When --sandbox names a backend that doesn't exist, fail immediately.
	// This is a security invariant: declared isolation MUST be available.
	_, err := sandbox.Get("nonexistent-backend")
	if err == nil {
		t.Fatal("should fail if declared sandbox backend doesn't exist")
	}
}

func TestSandbox_MisbahBackend_Available(t *testing.T) {
	// Check if Misbah is registered. It won't be unless the misbah
	// package is imported with a side-effect init() registration.
	// This test documents that sandbox backends must be explicitly imported.
	avail := sandbox.Available()
	t.Logf("available sandbox backends: %v", avail)
	// Not asserting Misbah is present — it depends on import wiring.
}
