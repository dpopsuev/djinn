// Package vfs implements the virtual filesystem mount table for Djinn.
//
// VFS is the runtime equivalent of workspace/ — while workspace/ declares
// repos and config as YAML manifests, vfs/ manages the live mount table
// that translates agent-visible virtual paths to host paths.
//
// Three isolation layers (SPC-95):
//   - Sandbox (Misbah jail): physical namespace isolation
//   - VFS (this package): logical path translation via MountTable
//   - Agent Space: invisible overlay per agent (future)
//
// Mount semantics by scope type:
//   - System: mount persists until session ends
//   - Operations: mount is ephemeral — unmounts when operation completes
//   - Global: read-only mounts of all repos
package vfs
