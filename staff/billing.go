// billing.go — per-agent token billing via Bugle Tracker.
package staff

import (
	"fmt"
	"time"

	"github.com/dpopsuev/djinn/bugleport"
)

// RecordUsage records token usage for an agent entity.
func RecordUsage(tracker *bugleport.InMemoryTracker, entityID bugleport.EntityID, role, step string, promptTokens, artifactTokens int) {
	tracker.Record(&bugleport.TokenRecord{
		CaseID:         fmt.Sprintf("agent-%d", entityID),
		Step:           step,
		Node:           role,
		PromptTokens:   promptTokens,
		ArtifactTokens: artifactTokens,
		Timestamp:      time.Now(),
	})
}
