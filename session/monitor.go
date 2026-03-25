package session

import "sync"

// MonitorState tracks the relay lifecycle.
type MonitorState int

const (
	MonitorIdle     MonitorState = iota // normal operation
	MonitorSpawning                     // background session being prepared
	MonitorReady                        // background session ready for swap
	MonitorSwapping                     // swap in progress
)

// Default thresholds.
const (
	DefaultMaxTokens = 200_000
	DefaultSpawnAt   = 0.80
	DefaultSwapAt    = 0.95
)

// ContextMonitor watches cumulative token usage and triggers relay callbacks.
type ContextMonitor struct {
	mu        sync.Mutex
	maxTokens int
	spawnAt   float64
	swapAt    float64
	totalIn   int
	totalOut  int
	state     MonitorState
	onSpawn   func() // called when spawnAt reached
	onSwap    func() // called when swapAt reached
}

// MonitorOption configures a ContextMonitor.
type MonitorOption func(*ContextMonitor)

// WithMaxTokens sets the context window size.
func WithMaxTokens(n int) MonitorOption {
	return func(m *ContextMonitor) { m.maxTokens = n }
}

// WithSpawnAt sets the threshold (0.0-1.0) to trigger background spawn.
func WithSpawnAt(f float64) MonitorOption {
	return func(m *ContextMonitor) { m.spawnAt = f }
}

// WithSwapAt sets the threshold (0.0-1.0) to trigger session swap.
func WithSwapAt(f float64) MonitorOption {
	return func(m *ContextMonitor) { m.swapAt = f }
}

// WithOnSpawn sets the callback fired when spawn threshold is reached.
func WithOnSpawn(fn func()) MonitorOption {
	return func(m *ContextMonitor) { m.onSpawn = fn }
}

// WithOnSwap sets the callback fired when swap threshold is reached.
func WithOnSwap(fn func()) MonitorOption {
	return func(m *ContextMonitor) { m.onSwap = fn }
}

// NewContextMonitor creates a monitor with the given options.
func NewContextMonitor(opts ...MonitorOption) *ContextMonitor {
	m := &ContextMonitor{
		maxTokens: DefaultMaxTokens,
		spawnAt:   DefaultSpawnAt,
		swapAt:    DefaultSwapAt,
	}
	for _, o := range opts {
		o(m)
	}
	return m
}

// Record updates cumulative token counts and checks thresholds.
// Returns true if a callback was fired.
func (m *ContextMonitor) Record(inputTokens, outputTokens int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalIn += inputTokens
	m.totalOut += outputTokens

	usage := m.usageLocked()

	// Check swap threshold first (higher priority).
	if usage >= m.swapAt && m.state == MonitorReady {
		m.state = MonitorSwapping
		if m.onSwap != nil {
			m.onSwap()
		}
		return true
	}

	// Check spawn threshold.
	if usage >= m.spawnAt && m.state == MonitorIdle {
		m.state = MonitorSpawning
		if m.onSpawn != nil {
			m.onSpawn()
		}
		return true
	}

	return false
}

// Usage returns the current context utilization as a fraction (0.0-1.0).
func (m *ContextMonitor) Usage() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.usageLocked()
}

// TotalTokens returns the cumulative input + output tokens.
func (m *ContextMonitor) TotalTokens() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.totalIn + m.totalOut
}

// State returns the current monitor state.
func (m *ContextMonitor) State() MonitorState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// SetState transitions the monitor to a new state.
// Used by RelayManager to advance the lifecycle externally.
func (m *ContextMonitor) SetState(s MonitorState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = s
}

// ShouldSpawn returns true if usage exceeds spawn threshold and state is idle.
func (m *ContextMonitor) ShouldSpawn() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.usageLocked() >= m.spawnAt && m.state == MonitorIdle
}

// ShouldSwap returns true if usage exceeds swap threshold and state is ready.
func (m *ContextMonitor) ShouldSwap() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.usageLocked() >= m.swapAt && m.state == MonitorReady
}

// Reset clears token counters and returns to idle state.
func (m *ContextMonitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalIn = 0
	m.totalOut = 0
	m.state = MonitorIdle
}

func (m *ContextMonitor) usageLocked() float64 {
	if m.maxTokens <= 0 {
		return 0
	}
	total := float64(m.totalIn + m.totalOut)
	return total / float64(m.maxTokens)
}
