// vessel_integration_test.go — RED: proves Substrate.Vessel() spawns agent
// in workspace namespace. This test MUST FAIL until GREEN tasks complete.
//
// DJN-TSK-1029
package substrate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dpopsuev/troupe/testkit"
)

// TestVessel_WorkDirIsWorkspace proves that a Vessel created by Substrate
// roots its WorkDir in a real workspace — not the process CWD.
func TestVessel_WorkDirIsWorkspace(t *testing.T) {
	workspaceDir := t.TempDir()
	log := testkit.NewStubEventLog()
	sub := NewStubSubstrate(stubExecutor{}, log)
	sub.SetWorkDir(workspaceDir)

	v := sub.Vessel(SpawnConfig{Role: "developer"})
	defer v.Close(context.Background()) //nolint:errcheck // test cleanup

	// Vessel.WorkDir() must return the actual workspace, not hardcoded /tmp/workspace.
	if v.WorkDir() != workspaceDir {
		t.Fatalf("Vessel.WorkDir() = %q, want %q", v.WorkDir(), workspaceDir)
	}
}

// TestVessel_WriteToolResolvesRelativePath proves that a Write tool call
// through vessel.Tools() resolves a relative path against WorkDir,
// not the process CWD.
func TestVessel_WriteToolResolvesRelativePath(t *testing.T) {
	workspaceDir := t.TempDir()
	log := testkit.NewStubEventLog()
	sub := NewStubSubstrate(stubExecutor{}, log)
	sub.SetWorkDir(workspaceDir)

	v := sub.Vessel(SpawnConfig{Role: "developer"})
	defer v.Close(context.Background()) //nolint:errcheck // test cleanup

	ctx := context.Background()
	input, _ := json.Marshal(map[string]string{
		"path":    "hello.go",
		"content": "package main\n\nfunc main() {}\n",
	})

	// Execute Write via the vessel's tool executor.
	_, err := v.Tools().Execute(ctx, "Write", input)
	if err != nil {
		t.Fatalf("Write tool failed: %v", err)
	}

	// File must exist in workspace, not CWD.
	wsPath := filepath.Join(workspaceDir, "hello.go")
	if _, err := os.Stat(wsPath); os.IsNotExist(err) {
		t.Fatalf("hello.go not found in workspace %s — tool wrote to wrong location", workspaceDir)
	}

	// File must NOT exist in CWD (test dir).
	cwd, _ := os.Getwd()
	cwdPath := filepath.Join(cwd, "hello.go")
	if _, err := os.Stat(cwdPath); err == nil {
		os.Remove(cwdPath) // cleanup leaked file
		t.Fatalf("hello.go found in CWD %s — tool should write to workspace, not CWD", cwd)
	}
}

// TestVessel_EventLogWired proves the Vessel's EventLog is the same
// instance provided by Substrate — events flow to the unified log.
func TestVessel_EventLogWired(t *testing.T) {
	log := testkit.NewStubEventLog()
	sub := NewStubSubstrate(stubExecutor{}, log)

	v := sub.Vessel(SpawnConfig{Role: "developer"})
	defer v.Close(context.Background()) //nolint:errcheck // test cleanup

	if v.EventLog() != log {
		t.Fatal("Vessel.EventLog() should return the Substrate's EventLog")
	}
}
