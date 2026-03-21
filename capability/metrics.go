package capability

import (
	"context"
	"time"
)

// MetricsSourcePort queries numeric time-series metrics.
type MetricsSourcePort interface {
	Query(ctx context.Context, metric string, window time.Duration) (float64, error)
}
