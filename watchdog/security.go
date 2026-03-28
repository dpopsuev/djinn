package watchdog

import "context"

const securityWatchdogName = "security-watchdog"

// SecurityWatchdog is a stub for GOL-3. Real implementation requires
// Misbah permission event stream integration.
type SecurityWatchdog struct{}

func NewSecurityWatchdog() *SecurityWatchdog                { return &SecurityWatchdog{} }
func (w *SecurityWatchdog) Name() string                    { return securityWatchdogName }
func (w *SecurityWatchdog) Category() string                { return CategorySecurity }
func (w *SecurityWatchdog) Start(ctx context.Context) error { return nil }
func (w *SecurityWatchdog) Stop(ctx context.Context) error  { return nil }
