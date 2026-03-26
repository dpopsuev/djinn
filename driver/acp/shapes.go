// shapes.go — ACP event shape classifier (GOL-23).
//
// ShapeKind classifies ACP update events by their structure, not their name.
// Forward-compatible: unknown event types get classified by structure so that
// the TUI can render them without hard-coding every possible event name.
package acp

// ShapeKind classifies ACP update events by their structure, not their name.
type ShapeKind int

const (
	ShapeTextStream      ShapeKind = iota // has "content" or "text" field
	ShapeStructuredList                   // has "items" array
	ShapeActionLifecycle                  // has "status" field (pending/running/done/error)
	ShapeStateChange                      // has "key" + "value" fields
	ShapeCapabilityList                   // has "capabilities" or "tools" array
	ShapePlanUpdate                       // has "steps" or "plan" array
	ShapeDiffUpdate                       // has "diff" or "changes" with file paths
	ShapeUnknown                          // none of the above
)

// ClassifyShape inspects a raw ACP event payload and returns its structural kind.
// Check order is priority-based: more specific shapes (plan, diff) come first.
func ClassifyShape(data map[string]any) ShapeKind {
	if data == nil {
		return ShapeUnknown
	}

	// 1. Plan update: has "steps" or "plan" array.
	if isSlice(data, "steps") || isSlice(data, "plan") {
		return ShapePlanUpdate
	}

	// 2. Diff update: has "diff" or "changes" with file-like entries.
	if isSlice(data, "diff") || hasDiffChanges(data) {
		return ShapeDiffUpdate
	}

	// 3. Text stream: has "content" or "text" string.
	if isString(data, "content") || isString(data, "text") {
		return ShapeTextStream
	}

	// 4. Structured list: has "items" array.
	if isSlice(data, "items") {
		return ShapeStructuredList
	}

	// 5. Action lifecycle: has "status" string.
	if isString(data, "status") {
		return ShapeActionLifecycle
	}

	// 6. State change: has "key" + "value".
	if _, hasKey := data["key"]; hasKey {
		if _, hasVal := data["value"]; hasVal {
			return ShapeStateChange
		}
	}

	// 7. Capability list: has "capabilities" or "tools" array.
	if isSlice(data, "capabilities") || isSlice(data, "tools") {
		return ShapeCapabilityList
	}

	return ShapeUnknown
}

// String returns a human-readable name for the shape kind.
func (k ShapeKind) String() string {
	switch k {
	case ShapeTextStream:
		return "text_stream"
	case ShapeStructuredList:
		return "structured_list"
	case ShapeActionLifecycle:
		return "action_lifecycle"
	case ShapeStateChange:
		return "state_change"
	case ShapeCapabilityList:
		return "capability_list"
	case ShapePlanUpdate:
		return "plan_update"
	case ShapeDiffUpdate:
		return "diff_update"
	case ShapeUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// --- helpers ---

// isString checks whether data[key] exists and is a string.
func isString(data map[string]any, key string) bool {
	v, ok := data[key]
	if !ok {
		return false
	}
	_, isStr := v.(string)
	return isStr
}

// isSlice checks whether data[key] exists and is a slice.
func isSlice(data map[string]any, key string) bool {
	v, ok := data[key]
	if !ok {
		return false
	}
	_, isArr := v.([]any)
	return isArr
}

// hasDiffChanges checks for a "changes" array whose elements look like
// file-level diffs (contain a "file" key).
func hasDiffChanges(data map[string]any) bool {
	v, ok := data["changes"]
	if !ok {
		return false
	}
	arr, isArr := v.([]any)
	if !isArr || len(arr) == 0 {
		return false
	}
	// At least the first element should have a "file" key.
	first, isMap := arr[0].(map[string]any)
	if !isMap {
		return false
	}
	_, hasFile := first["file"]
	return hasFile
}
