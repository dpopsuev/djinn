// bugle_bridge.go — bridges Djinn's SignalBus to Bugle's Bus.
// Djinn signals are forwarded to the Bugle bus for persistence and
// cross-agent coordination. The reverse direction is also supported.
package telemetry

import (
	"time"

	jSignal "github.com/dpopsuev/jericho/signal"
)

// BugleBridge forwards signals between Djinn's SignalBus and Bugle's Bus.
type BugleBridge struct {
	djinn *SignalBus
	bugle jSignal.Bus
}

// NewBugleBridge creates a bridge that forwards in both directions.
func NewBugleBridge(djinn *SignalBus, bugle jSignal.Bus) *BugleBridge {
	b := &BugleBridge{djinn: djinn, bugle: bugle}

	// Forward Djinn → Bugle.
	djinn.OnSignal(func(s Signal) {
		b.bugle.Emit(&jSignal.Signal{
			Timestamp: s.Timestamp.Format(time.RFC3339),
			Event:     s.Category,
			Agent:     s.Source,
			Meta: map[string]string{
				"workstream": s.Workstream,
				"message":    s.Message,
				"level":      s.Level.String(),
			},
		})
	})

	// Forward Bugle → Djinn.
	bugle.OnEmit(func(bs jSignal.Signal) {
		ts, _ := time.Parse(time.RFC3339, bs.Timestamp)
		djinn.Emit(Signal{
			Category:   bs.Event,
			Source:     bs.Agent,
			Timestamp:  ts,
			Workstream: bs.Meta["workstream"],
			Message:    bs.Meta["message"],
		})
	})

	return b
}
