package watchdog

import "context"

const driftWatchdogName = "drift-watchdog"

// DriftWatchdog is a stub for GOL-3. Real implementation requires
// an LLM agent to evaluate whether the working agent stays on-task.
type DriftWatchdog struct{}

func NewDriftWatchdog() *DriftWatchdog                   { return &DriftWatchdog{} }
func (w *DriftWatchdog) Name() string                    { return driftWatchdogName }
func (w *DriftWatchdog) Category() string                { return CategoryDrift }
func (w *DriftWatchdog) Start(ctx context.Context) error { return nil }
func (w *DriftWatchdog) Stop(ctx context.Context) error  { return nil }
