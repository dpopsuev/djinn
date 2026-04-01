package config

import (
	"errors"
	"testing"
)

// ErrPathTraversal and ErrInvalidValue are expected sentinel errors.
// These tests will FAIL until validation is added to Apply() methods (GREEN phase).

func TestConfig_TapFile_RejectsPathTraversal(t *testing.T) {
	c := &DebugConfigurable{}
	err := c.Apply(map[string]any{"tap_file": "../../../etc/passwd"})
	if err == nil {
		t.Fatal("SECURITY: path traversal in tap_file accepted")
	}
	if !errors.Is(err, ErrPathTraversal) {
		t.Fatalf("wrong error type: %v", err)
	}
}

func TestConfig_TapFile_RejectsAbsoluteOutsideWorkspace(t *testing.T) {
	c := &DebugConfigurable{}
	err := c.Apply(map[string]any{"tap_file": "/etc/hosts"})
	if err == nil {
		t.Fatal("SECURITY: absolute path outside workspace accepted")
	}
	if !errors.Is(err, ErrPathTraversal) {
		t.Fatalf("wrong error type: %v", err)
	}
}

func TestConfig_TapFile_AcceptsRelativeWithinWorkspace(t *testing.T) {
	c := &DebugConfigurable{}
	err := c.Apply(map[string]any{"tap_file": "debug/trace.jsonl"})
	if err != nil {
		t.Fatalf("valid relative path rejected: %v", err)
	}
	if c.TapFile != "debug/trace.jsonl" {
		t.Fatalf("TapFile = %q, want debug/trace.jsonl", c.TapFile)
	}
}

func TestConfig_SandboxBackend_RejectsTraversal(t *testing.T) {
	c := &SandboxConfigurable{}
	err := c.Apply(map[string]any{"backend": "../../bin/evil"})
	if err == nil {
		t.Fatal("SECURITY: path traversal in sandbox backend accepted")
	}
	if !errors.Is(err, ErrPathTraversal) {
		t.Fatalf("wrong error type: %v", err)
	}
}

func TestConfig_SandboxBackend_AcceptsKnownValues(t *testing.T) {
	for _, backend := range []string{"misbah", "bubblewrap", "podman", "none"} {
		c := &SandboxConfigurable{}
		err := c.Apply(map[string]any{"backend": backend})
		if err != nil {
			t.Errorf("valid backend %q rejected: %v", backend, err)
		}
	}
}

func TestConfig_MaxTurns_RejectsNegative(t *testing.T) {
	c := &SessionConfigurable{}
	err := c.Apply(map[string]any{"max_turns": -100})
	if err == nil {
		t.Fatal("SECURITY: negative max_turns accepted")
	}
	if !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("wrong error type: %v", err)
	}
}

func TestConfig_MaxTurns_RejectsTooLarge(t *testing.T) {
	c := &SessionConfigurable{}
	err := c.Apply(map[string]any{"max_turns": 999_999_999})
	if err == nil {
		t.Fatal("SECURITY: unreasonably large max_turns accepted")
	}
	if !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("wrong error type: %v", err)
	}
}

func TestConfig_MaxTurns_AcceptsValidRange(t *testing.T) {
	c := &SessionConfigurable{}
	err := c.Apply(map[string]any{"max_turns": 100})
	if err != nil {
		t.Fatalf("valid max_turns rejected: %v", err)
	}
	if c.MaxTurns != 100 {
		t.Fatalf("MaxTurns = %d, want 100", c.MaxTurns)
	}
}

func TestConfig_OutputMode_RejectsUnknown(t *testing.T) {
	c := &SessionConfigurable{}
	err := c.Apply(map[string]any{"output_mode": "arbitrary_injection"})
	if err == nil {
		t.Fatal("SECURITY: unknown output_mode accepted")
	}
	if !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("wrong error type: %v", err)
	}
}

func TestConfig_OutputMode_AcceptsKnownValues(t *testing.T) {
	for _, mode := range []string{"standard", "verbose", "quiet", ""} {
		c := &SessionConfigurable{}
		err := c.Apply(map[string]any{"output_mode": mode})
		if err != nil {
			t.Errorf("valid output_mode %q rejected: %v", mode, err)
		}
	}
}
