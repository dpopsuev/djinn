// billing.go — per-agent token billing via Bugle Tracker.
package persona

import (
	"fmt"
	"time"

	jBilling "github.com/dpopsuev/troupe/billing"
	"github.com/dpopsuev/troupe/world"
)

// RecordUsage records token usage for an agent entity.
func RecordUsage(tracker *jBilling.InMemoryTracker, entityID world.EntityID, role, step string, promptTokens, artifactTokens int) {
	tracker.Record(&jBilling.TokenRecord{
		CaseID:         fmt.Sprintf("agent-%d", entityID),
		Step:           step,
		Node:           role,
		PromptTokens:   promptTokens,
		ArtifactTokens: artifactTokens,
		Timestamp:      time.Now(),
	})
}
