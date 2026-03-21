package watchdog

import "context"

const qualityWatchdogName = "quality-watchdog"

// QualityWatchdog is a stub for GOL-3. Real implementation requires
// an LLM agent running in a separate container to evaluate code quality.
type QualityWatchdog struct{}

func NewQualityWatchdog() *QualityWatchdog              { return &QualityWatchdog{} }
func (w *QualityWatchdog) Name() string                 { return qualityWatchdogName }
func (w *QualityWatchdog) Category() string             { return CategoryQuality }
func (w *QualityWatchdog) Start(ctx context.Context) error { return nil }
func (w *QualityWatchdog) Stop(ctx context.Context) error  { return nil }
