// Package remote defines interfaces for cloud-hosted sandbox backends.
//
// CloudProvider abstracts the lifecycle of remote VMs across cloud
// platforms (AWS Firecracker, GCP gVisor, Azure Hyper-V, K8s CRD).
// RemoteSandbox implements sandbox.Sandbox backed by a CloudProvider.
//
// All implementations are stubs — real cloud code is out of scope.
// The interfaces define the port contract for future adapters.
//
// Prior art: agentcomputer.ai uses Firecracker microVMs with sub-second
// provisioning, REST control plane, and SSE agent session streaming.
// Djinn's equivalent: Misbah (Kata/Firecracker) + Clutch + MCP Shell.
package remote

import (
	"context"
	"errors"
	"net"

	"github.com/dpopsuev/djinn/sandbox"
)

// Sentinel errors for remote sandbox providers.
var (
	ErrRemoteExecNotImpl = errors.New("remote exec not implemented")
	ErrAWSNotImpl        = errors.New("AWS Firecracker provider not implemented")
	ErrGCPNotImpl        = errors.New("GCP gVisor provider not implemented")
	ErrK8sNotImpl        = errors.New("K8s agent-sandbox provider not implemented")
)

// VMID identifies a remote VM instance.
type VMID string

// VMStatus represents the lifecycle state of a remote VM.
type VMStatus string

const (
	VMPending      VMStatus = "pending"
	VMProvisioning VMStatus = "provisioning"
	VMRunning      VMStatus = "running"
	VMStopping     VMStatus = "stopping"
	VMStopped      VMStatus = "stopped"
	VMError        VMStatus = "error"
)

// VMSpec describes the desired VM configuration.
type VMSpec struct {
	Image    string   // OCI image, AMI, or Nix derivation
	CPUs     int      // vCPU count
	MemoryMB int      // RAM in megabytes
	DiskGB   int      // persistent disk in gigabytes
	Repos    []string // workspace repos to mount into the VM
	Level    string   // isolation level (namespace, container, kata)
}

// CloudProvider abstracts cloud infrastructure for remote sandboxes.
// Each provider manages VM lifecycle on a specific cloud platform.
// The clutch protocol runs over the net.Conn returned by ConnectVM.
type CloudProvider interface {
	CreateVM(ctx context.Context, spec VMSpec) (VMID, error)
	DestroyVM(ctx context.Context, id VMID) error
	StatusVM(ctx context.Context, id VMID) (VMStatus, error)
	ConnectVM(ctx context.Context, id VMID) (net.Conn, error)
	Name() string
}

// RemoteSandbox implements sandbox.Sandbox backed by a CloudProvider.
// The shell runs locally; the backend runs inside the remote VM.
// Communication flows over clutch socket tunneled through ConnectVM.
type RemoteSandbox struct {
	provider CloudProvider
}

// NewRemoteSandbox creates a sandbox backed by the given cloud provider.
func NewRemoteSandbox(provider CloudProvider) *RemoteSandbox {
	return &RemoteSandbox{provider: provider}
}

func (r *RemoteSandbox) Create(ctx context.Context, level string, repos []string) (sandbox.Handle, error) {
	id, err := r.provider.CreateVM(ctx, VMSpec{
		Level: level,
		Repos: repos,
	})
	if err != nil {
		return "", err
	}
	return sandbox.Handle(id), nil
}

func (r *RemoteSandbox) Destroy(ctx context.Context, handle sandbox.Handle) error {
	return r.provider.DestroyVM(ctx, VMID(handle))
}

func (r *RemoteSandbox) Exec(_ context.Context, _ sandbox.Handle, _ []string, _ int64) (sandbox.ExecResult, error) {
	return sandbox.ExecResult{}, ErrRemoteExecNotImpl
}

func (r *RemoteSandbox) Name() string {
	return "remote-" + r.provider.Name()
}

// --- Stub providers: compile but return "not implemented" ---

// AWSFirecrackerProvider would manage Firecracker microVMs on EC2 bare-metal.
type AWSFirecrackerProvider struct{}

func (p *AWSFirecrackerProvider) CreateVM(_ context.Context, _ VMSpec) (VMID, error) {
	return "", ErrAWSNotImpl
}
func (p *AWSFirecrackerProvider) DestroyVM(_ context.Context, _ VMID) error {
	return ErrAWSNotImpl
}
func (p *AWSFirecrackerProvider) StatusVM(_ context.Context, _ VMID) (VMStatus, error) {
	return VMError, ErrAWSNotImpl
}
func (p *AWSFirecrackerProvider) ConnectVM(_ context.Context, _ VMID) (net.Conn, error) {
	return nil, ErrAWSNotImpl
}
func (p *AWSFirecrackerProvider) Name() string { return "aws-firecracker" }

// GCPGVisorProvider would manage gVisor sandboxes on GCE.
type GCPGVisorProvider struct{}

func (p *GCPGVisorProvider) CreateVM(_ context.Context, _ VMSpec) (VMID, error) {
	return "", ErrGCPNotImpl
}
func (p *GCPGVisorProvider) DestroyVM(_ context.Context, _ VMID) error {
	return ErrGCPNotImpl
}
func (p *GCPGVisorProvider) StatusVM(_ context.Context, _ VMID) (VMStatus, error) {
	return VMError, ErrGCPNotImpl
}
func (p *GCPGVisorProvider) ConnectVM(_ context.Context, _ VMID) (net.Conn, error) {
	return nil, ErrGCPNotImpl
}
func (p *GCPGVisorProvider) Name() string { return "gcp-gvisor" }

// K8sAgentSandboxProvider would manage VMs via kubernetes-sigs/agent-sandbox CRD.
type K8sAgentSandboxProvider struct{}

func (p *K8sAgentSandboxProvider) CreateVM(_ context.Context, _ VMSpec) (VMID, error) {
	return "", ErrK8sNotImpl
}
func (p *K8sAgentSandboxProvider) DestroyVM(_ context.Context, _ VMID) error {
	return ErrK8sNotImpl
}
func (p *K8sAgentSandboxProvider) StatusVM(_ context.Context, _ VMID) (VMStatus, error) {
	return VMError, ErrK8sNotImpl
}
func (p *K8sAgentSandboxProvider) ConnectVM(_ context.Context, _ VMID) (net.Conn, error) {
	return nil, ErrK8sNotImpl
}
func (p *K8sAgentSandboxProvider) Name() string { return "k8s-agent-sandbox" }

// Interface compliance.
var (
	_ CloudProvider   = (*AWSFirecrackerProvider)(nil)
	_ CloudProvider   = (*GCPGVisorProvider)(nil)
	_ CloudProvider   = (*K8sAgentSandboxProvider)(nil)
	_ sandbox.Sandbox = (*RemoteSandbox)(nil)
)
