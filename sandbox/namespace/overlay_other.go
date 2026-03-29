//go:build !linux

package namespace

import "errors"

// ErrUnsupported is returned on non-Linux platforms.
var ErrUnsupported = errors.New("namespace: overlayfs requires Linux")
