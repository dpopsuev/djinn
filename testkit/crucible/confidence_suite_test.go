//go:build e2e

// confidence_suite_test.go — Real LLM confidence tests using AgentSuite.
// Workspace isolation is invisible. Tools are auto-wired. RBAC is automatic.
//
// Run: DJINN_PROVIDER=vertex-ai DJINN_MODEL=claude-sonnet-4-6 go test -tags e2e -run TestConfidence -v -timeout 120s ./testkit/crucible/
package crucible

import (
	"testing"

	"github.com/dpopsuev/djinn/testkit"
	"github.com/stretchr/testify/suite"
)

type ConfidenceSuite struct {
	testkit.AgentSuite
}

func TestConfidenceSuite(t *testing.T) {
	suite.Run(t, new(ConfidenceSuite))
}

// TestWritesFile proves: real LLM + RBAC Coder + Write tool → file on disk.
func (s *ConfidenceSuite) TestWritesFile() {
	s.SkipIfNoProvider()

	result := s.RunAgent("coder-1", []string{"developer"}, "Create a file called hello.go with package main and a main function that prints 'hello djinn'")
	s.T().Logf("Agent: %s", result)

	s.True(s.Workspace().HasFile("hello.go"), "hello.go should exist")

	content, err := s.Workspace().ReadFile("hello.go")
	s.Require().NoError(err)
	s.Contains(content, "package main")
	s.Contains(content, "func main")
}

// TestReadsAndEdits proves: real LLM + Read + Edit → file modified.
func (s *ConfidenceSuite) TestReadsAndEdits() {
	s.SkipIfNoProvider()

	s.Workspace().WriteFile(s.T(), "config.go", "package config\n\nvar Version = \"0.0.0\"\n")

	result := s.RunAgent("coder-1", []string{"developer"}, "Read config.go and change the Version from 0.0.0 to 1.0.0")
	s.T().Logf("Agent: %s", result)

	content, err := s.Workspace().ReadFile("config.go")
	s.Require().NoError(err)
	s.Contains(content, "1.0.0", "version should be updated")
	s.NotContains(content, "0.0.0", "old version should be gone")
}
