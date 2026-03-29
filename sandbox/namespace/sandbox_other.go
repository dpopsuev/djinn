//go:build !linux

package namespace

import (
	"context"

	"github.com/dpopsuev/djinn/sandbox"
	"github.com/dpopsuev/mirage"
)

// NamespaceSandbox is not supported on non-Linux platforms.
type NamespaceSandbox struct{}

// New returns a stub on non-Linux.
func New(_ string) *NamespaceSandbox { return &NamespaceSandbox{} }

func (s *NamespaceSandbox) Name() string { return "namespace" }

func (s *NamespaceSandbox) Create(_ context.Context, _ string, _ []string) (sandbox.Handle, error) {
	return "", ErrUnsupported
}

func (s *NamespaceSandbox) Destroy(_ context.Context, _ sandbox.Handle) error {
	return ErrUnsupported
}

func (s *NamespaceSandbox) Exec(_ context.Context, _ sandbox.Handle, _ []string, _ int64) (sandbox.ExecResult, error) {
	return sandbox.ExecResult{}, ErrUnsupported
}

func (s *NamespaceSandbox) GetSpace(_ sandbox.Handle) (mirage.Space, error) {
	return nil, ErrUnsupported
}

func (s *NamespaceSandbox) Diff(_ sandbox.Handle) ([]mirage.Change, error) {
	return nil, ErrUnsupported
}

func (s *NamespaceSandbox) Commit(_ sandbox.Handle, _ []string) error {
	return ErrUnsupported
}
