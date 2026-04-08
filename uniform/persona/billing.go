// billing.go — per-agent token billing via Bugle Tracker.
package persona

import (
	"fmt"
	"time"

	"github.com/dpopsuev/djinn/jerichoport"
)

// RecordUsage records token usage for an agent entity.
func RecordUsage(tracker *jerichoport.InMemoryTracker, entityID jerichoport.EntityID, role, step string, promptTokens, artifactTokens int) {
	tracker.Record(&jerichoport.TokenRecord{
		CaseID:         fmt.Sprintf("agent-%d", entityID),
		Step:           step,
		Node:           role,
		PromptTokens:   promptTokens,
		ArtifactTokens: artifactTokens,
		Timestamp:      time.Now(),
	})
}
