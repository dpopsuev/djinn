//go:build linux

// Package namespace implements an invisible filesystem proxy using
// mirage overlay spaces. The agent's built-in tools hit the filesystem —
// mirage controls what the filesystem returns via copy-on-write isolation.
package namespace

import "errors"

// Sentinel errors.
var (
	ErrUnsupported = errors.New("namespace: fuse-overlayfs not available")
)
