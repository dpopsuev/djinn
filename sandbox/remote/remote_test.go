package remote

import (
	"context"
	"testing"
)

func TestRemoteSandbox_StubReturnsNotImplemented(t *testing.T) {
	sb := NewRemoteSandbox(&AWSFirecrackerProvider{})
	_, err := sb.Create(context.Background(), "container", []string{"/workspace"})
	if err == nil {
		t.Fatal("stub should return not-implemented error")
	}
}

func TestCloudProvider_InterfaceSatisfaction(t *testing.T) {
	// Verify all stubs satisfy the interface at compile time.
	var _ CloudProvider = (*AWSFirecrackerProvider)(nil)
	var _ CloudProvider = (*GCPGVisorProvider)(nil)
	var _ CloudProvider = (*K8sAgentSandboxProvider)(nil)
}

func TestRemoteSandbox_Name(t *testing.T) {
	sb := NewRemoteSandbox(&K8sAgentSandboxProvider{})
	if sb.Name() != "remote-k8s-agent-sandbox" {
		t.Fatalf("name = %q", sb.Name())
	}
}

// --- Test skeletons: future implementation ---

func TestRemoteSandbox_CreatesPod(t *testing.T)             { t.Skip("not implemented — requires K8s cluster") }
func TestRemoteSandbox_ClutchOverPortForward(t *testing.T)   { t.Skip("not implemented") }
func TestRemoteSandbox_WarmPool(t *testing.T)                { t.Skip("not implemented") }
func TestRemoteSandbox_DeepHibernation(t *testing.T)         { t.Skip("not implemented") }
func TestCloudProvider_AWS(t *testing.T)                     { t.Skip("not implemented — requires AWS credentials") }
func TestCloudProvider_GCP(t *testing.T)                     { t.Skip("not implemented — requires GCP credentials") }
func TestCloudProvider_K8s(t *testing.T)                     { t.Skip("not implemented — requires K8s cluster") }
