// capacity.go — Agent capacity tracking for the Djinn staffing model.
// AgentCapacity uses Go slice-like len/cap semantics.
// Cap is the max concurrent agents (operator-only).
// Running (len) is currently active agents (managed by GenSec).
package staff

import (
	"errors"
	"fmt"
	"sync"
)

// ErrCapacityZero is returned when trying to decrease capacity below zero.
var ErrCapacityZero = errors.New("capacity: cannot decrease below zero")

// ErrCapacityNegative is returned when trying to set negative capacity.
var ErrCapacityNegative = errors.New("capacity: cannot set negative value")

// AgentCapacity tracks concurrent agent slots using len/cap semantics.
// Cap is the maximum concurrent agents — only the operator can change it.
// Running (len) is the number of currently active agents.
type AgentCapacity struct {
	mu      sync.RWMutex
	cap     int
	running int
}

// NewAgentCapacity creates a capacity tracker with the given initial cap.
// Default cap is 1 (single agent, safest starting point).
func NewAgentCapacity(initialCap int) *AgentCapacity {
	if initialCap < 0 {
		initialCap = 0
	}
	return &AgentCapacity{cap: initialCap}
}

// Cap returns the maximum concurrent agent count.
func (ac *AgentCapacity) Cap() int {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.cap
}

// Len returns the number of currently running agents.
func (ac *AgentCapacity) Len() int {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.running
}

// SetCap sets the maximum capacity. Returns error if n < 0.
func (ac *AgentCapacity) SetCap(n int) error {
	if n < 0 {
		return ErrCapacityNegative
	}
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.cap = n
	return nil
}

// Inc increases capacity by 1.
func (ac *AgentCapacity) Inc() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.cap++
}

// Dec decreases capacity by 1. Returns error if already at zero.
func (ac *AgentCapacity) Dec() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	if ac.cap <= 0 {
		return ErrCapacityZero
	}
	ac.cap--
	return nil
}

// CanSpawn returns true if there is room for another agent.
func (ac *AgentCapacity) CanSpawn() bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.running < ac.cap
}

// Acquire atomically increments running if there is room.
// Returns true if the slot was acquired, false if at capacity.
func (ac *AgentCapacity) Acquire() bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	if ac.running >= ac.cap {
		return false
	}
	ac.running++
	return true
}

// Release decrements running. Safe to call even if running is 0.
func (ac *AgentCapacity) Release() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	if ac.running > 0 {
		ac.running--
	}
}

// ExcessRunning returns how many agents exceed current capacity.
// This happens when cap is decreased below running count.
func (ac *AgentCapacity) ExcessRunning() int {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	if ac.running > ac.cap {
		return ac.running - ac.cap
	}
	return 0
}

// String returns "len/cap" format (e.g., "2/4").
func (ac *AgentCapacity) String() string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return fmt.Sprintf("%d/%d", ac.running, ac.cap)
}
