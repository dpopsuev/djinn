// Package workspace manages named workspace manifests and provides
// an event bus for workspace lifecycle notifications.
//
// Deprecated: workspace is the declaration layer — it loads YAML manifests
// describing repos, drivers, and MCP servers. For runtime path translation
// and mount management, use the vfs/ package instead. The vfs.MountTable
// is the runtime equivalent of workspace.Workspace.Repos.
//
// Do NOT delete this package — it is still used for loading workspace config.
// VFS consumes workspace declarations to build the live mount table.
package workspace
