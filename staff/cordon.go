// cordon.go — Cordon system for the Djinn staffing model.
// A Cordon is an automatic safety brake that halts agent execution
// when budget, time, or behavioural thresholds are breached.
package staff

import "time"

// CordonReason classifies why an agent was cordoned off.
type CordonReason string

const (
	CordonTokenBudget    CordonReason = "token_budget_exceeded"
	CordonCostBudget     CordonReason = "cost_budget_exceeded"
	CordonTimeBudget     CordonReason = "time_budget_exceeded"
	CordonContextRedline CordonReason = "context_redline"
	CordonGateLoop       CordonReason = "gate_loop_exceeded"
	CordonSilence        CordonReason = "agent_silence"
	CordonBlastRadius    CordonReason = "blast_radius_exceeded"
)

// Cordon captures the fact that an agent has been fenced off.
type Cordon struct {
	Reason    CordonReason
	AgentID   string
	Detail    string
	Timestamp time.Time
}

// CordonConfig holds the thresholds that trigger a cordon.
type CordonConfig struct {
	MaxTokens       int           // total token ceiling (in + out); 0 = unlimited
	MaxCost         float64       // cost ceiling in dollars; 0 = unlimited
	MaxDuration     time.Duration // wall-clock ceiling; 0 = unlimited
	ContextRedline  float64       // context usage fraction that triggers cordon (0.0-1.0); 0 = disabled
	MaxGateRetries  int           // consecutive gate failures before cordon; default 3
	SilenceTimeout  time.Duration // if agent produces no output for this long; default 30s
	MaxFilesChanged int           // blast radius: max files an agent may touch; default 50
}

// DefaultCordonConfig returns production-safe defaults.
func DefaultCordonConfig() CordonConfig {
	return CordonConfig{
		MaxGateRetries:  3,
		SilenceTimeout:  30 * time.Second,
		MaxFilesChanged: 50,
	}
}

// CheckCordon inspects agent metrics against the cordon config.
// Returns nil when the agent is within all thresholds.
func CheckCordon(metrics *AgentMetrics, config CordonConfig) *Cordon {
	if metrics == nil {
		return nil
	}

	metrics.mu.Lock()
	agentID := metrics.AgentID
	totalTokens := metrics.TotalIn + metrics.TotalOut
	cost := metrics.Cost
	gateFails := metrics.GateFails
	metrics.mu.Unlock()

	now := time.Now()

	// Token budget.
	if config.MaxTokens > 0 && totalTokens > config.MaxTokens {
		return &Cordon{
			Reason:    CordonTokenBudget,
			AgentID:   agentID,
			Detail:    "total tokens exceeded budget",
			Timestamp: now,
		}
	}

	// Cost budget.
	if config.MaxCost > 0 && cost > config.MaxCost {
		return &Cordon{
			Reason:    CordonCostBudget,
			AgentID:   agentID,
			Detail:    "cost exceeded budget",
			Timestamp: now,
		}
	}

	// Gate loop.
	if config.MaxGateRetries > 0 && gateFails > config.MaxGateRetries {
		return &Cordon{
			Reason:    CordonGateLoop,
			AgentID:   agentID,
			Detail:    "gate failures exceeded max retries",
			Timestamp: now,
		}
	}

	return nil
}
