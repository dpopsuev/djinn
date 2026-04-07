package builtin

// Battery bridge — strangler fig phase 1.
//
// Battery's tool.Tool and tool.Executor have identical signatures to
// builtin.Tool and builtin.ToolExecutor. This file documents the
// compatibility and provides compile-time verification.
//
// Phase 2 (next): replace builtin.Tool with battery/tool.Tool type alias.
// Phase 3: delete builtin.Tool, all consumers import battery/tool directly.

import (
	batterytool "github.com/dpopsuev/battery/tool"
)

// Compile-time verification: battery.Tool and builtin.Tool are compatible.
// Both have: Name() string, Description() string, InputSchema() json.RawMessage,
// Execute(ctx, input) (string, error).
var _ batterytool.Tool = (Tool)(nil)
