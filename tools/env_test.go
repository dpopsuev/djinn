package tools

import (
	"runtime"
	"testing"
)

func TestSnapshot_GoVersion(t *testing.T) {
	snap := Snapshot()
	if snap.GoVersion != runtime.Version() {
		t.Fatalf("GoVersion = %q, want %q", snap.GoVersion, runtime.Version())
	}
}

func TestSnapshot_WorkingDir(t *testing.T) {
	snap := Snapshot()
	if snap.WorkingDir == "" {
		t.Fatal("WorkingDir is empty")
	}
}

func TestSnapshot_Hostname(t *testing.T) {
	snap := Snapshot()
	if snap.Hostname == "" {
		t.Fatal("Hostname is empty")
	}
}

func TestSnapshot_GitBranch(t *testing.T) {
	snap := Snapshot()
	// May be empty if not in a git repo, but should not panic
	_ = snap.GitBranch
}

func TestSnapshot_MisbahDaemon(t *testing.T) {
	snap := Snapshot()
	// Just verify it doesn't panic — result depends on environment
	_ = snap.MisbahDaemon
}
