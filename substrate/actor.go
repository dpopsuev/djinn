// actor.go — adapts Djinn's agent.Run() to Troupe's ActorFunc.
//
// The Substrate spawns agents via Troupe's execution.ActorFunc.
// This adapter wraps the existing agent loop so it can be used
// in a Troupe Pool or called directly by the Substrate.
package substrate

import (
	"context"
	"log/slog"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/contextmgr"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// ActorFunc is the work execution function. Takes input, returns output.
// Compatible with Troupe's execution.ActorFunc signature.
type ActorFunc func(ctx context.Context, input string) (string, error)

// AgentActorFunc creates an ActorFunc that runs Djinn's agent loop.
// Each call runs one agent session: prompt in, response out.
func AgentActorFunc(cfg AgentActorConfig) ActorFunc {
	return func(ctx context.Context, prompt string) (string, error) {
		agentCfg := agent.Config{
			Driver:       cfg.Driver,
			Tools:        cfg.Tools,
			Envelope:     cfg.Envelope,
			Session:      cfg.Session,
			SystemPrompt: cfg.SystemPrompt,
			MaxTurns:     cfg.MaxTurns,
			ToolsEnabled: true,
			Mode:         cfg.Mode,
			Approve:      cfg.Approve,
			Enforcer:     cfg.Enforcer,
			Token:        cfg.Token,
			Handler:      cfg.Handler,
			Log:          cfg.Log,
		}
		return agent.Run(ctx, agentCfg, prompt)
	}
}

// AgentActorConfig holds the dependencies needed to create an AgentActorFunc.
type AgentActorConfig struct {
	Driver       driver.ChatDriver
	Tools        builtin.ToolExecutor
	Envelope     *agent.ToolEnvelope
	Session      *contextmgr.Session
	SystemPrompt string
	MaxTurns     int
	Mode         agent.Mode
	Approve      agent.ApprovalFunc
	Enforcer     policy.ToolPolicyEnforcer
	Token        policy.CapabilityToken
	Handler      agent.EventHandler
	Log          *slog.Logger
}
