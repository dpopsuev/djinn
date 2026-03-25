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

func TestRemoteSandbox_Destroy(t *testing.T) {
	sb := NewRemoteSandbox(&AWSFirecrackerProvider{})
	err := sb.Destroy(context.Background(), "some-handle")
	if err == nil {
		t.Fatal("stub should return not-implemented error")
	}
}

func TestRemoteSandbox_AllProviders(t *testing.T) {
	providers := []CloudProvider{
		&AWSFirecrackerProvider{},
		&GCPGVisorProvider{},
		&K8sAgentSandboxProvider{},
	}
	for _, p := range providers {
		sb := NewRemoteSandbox(p)
		if sb.Name() == "" {
			t.Fatalf("provider %T has empty name", p)
		}

		_, err := sb.Create(context.Background(), "container", []string{"/workspace"})
		if err == nil {
			t.Fatalf("provider %T: stub should return error", p)
		}

		err = sb.Destroy(context.Background(), "id")
		if err == nil {
			t.Fatalf("provider %T: destroy stub should return error", p)
		}
	}
}

func TestCloudProvider_StatusVM(t *testing.T) {
	providers := []CloudProvider{
		&AWSFirecrackerProvider{},
		&GCPGVisorProvider{},
		&K8sAgentSandboxProvider{},
	}
	for _, p := range providers {
		status, err := p.StatusVM(context.Background(), "vm-1")
		if err == nil {
			t.Fatalf("provider %T: stub StatusVM should return error", p)
		}
		if status != VMError {
			t.Fatalf("provider %T: status = %q, want VMError", p, status)
		}
	}
}

func TestCloudProvider_ConnectVM(t *testing.T) {
	providers := []CloudProvider{
		&AWSFirecrackerProvider{},
		&GCPGVisorProvider{},
		&K8sAgentSandboxProvider{},
	}
	for _, p := range providers {
		conn, err := p.ConnectVM(context.Background(), "vm-1")
		if err == nil {
			t.Fatalf("provider %T: stub ConnectVM should return error", p)
		}
		if conn != nil {
			t.Fatalf("provider %T: conn should be nil", p)
		}
	}
}

func TestVMStatus_Constants(t *testing.T) {
	statuses := []VMStatus{VMPending, VMProvisioning, VMRunning, VMStopping, VMStopped, VMError}
	for _, s := range statuses {
		if s == "" {
			t.Fatal("VMStatus constant should not be empty")
		}
	}
}

func TestVMSpec_Fields(t *testing.T) {
	spec := VMSpec{
		Image:    "djinn:latest",
		CPUs:     2,
		MemoryMB: 1024,
		DiskGB:   10,
		Repos:    []string{"/workspace"},
		Level:    "container",
	}
	if spec.Image != "djinn:latest" {
		t.Fatal("spec fields")
	}
}
