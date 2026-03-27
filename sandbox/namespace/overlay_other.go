//go:build !linux

package namespace

import "errors"

// ErrUnsupported is returned on non-Linux platforms.
var ErrUnsupported = errors.New("namespace: overlayfs requires Linux")

// Overlay is not supported on non-Linux platforms.
type Overlay struct {
	Lower  string
	Upper  string
	Work   string
	Merged string
}

// Mount returns ErrUnsupported on non-Linux platforms.
func Mount(_ string) (*Overlay, error) {
	return nil, ErrUnsupported
}

// Unmount is a no-op on non-Linux platforms.
func (o *Overlay) Unmount() error { return nil }

// Diff returns ErrUnsupported on non-Linux platforms.
func (o *Overlay) Diff() ([]string, error) { return nil, ErrUnsupported }

// Commit returns ErrUnsupported on non-Linux platforms.
func (o *Overlay) Commit(_ []string) error { return ErrUnsupported }
