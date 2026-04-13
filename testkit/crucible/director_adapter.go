// director_adapter.go — bridges UnifiedDirector into crucible ActorFactory.
//
// Strangler Fig: existing Runner stays unchanged. This adapter lets
// tests use UnifiedDirector instead of manually calling agent.Run().
//
// GOL-163, TSK-1085
package crucible

import (
	"context"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/cortex"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/substrate"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// DirectorActorFactory creates a crucible ActorFactory backed by UnifiedDirector.
// The factory creates a Director configured for the workspace, and the
// returned ActorFunc calls Director.Run() with ModeAuto.
func DirectorActorFactory(
	drv driver.ChatDriver,
	systemPromptFn func(workspace string) string,
	opts ...substrate.DirectorOption,
) ActorFactory {
	return func(workspace string) (ActorFunc, error) {
		reg := builtin.NewRegistryWithWorkDir(workspace)
		builtin.RegisterBuiltinTools(reg, workspace, workspace)

		sess := cortex.New("crucible", "test", workspace)

		prompt := ""
		if systemPromptFn != nil {
			prompt = systemPromptFn(workspace)
		}

		dirOpts := append([]substrate.DirectorOption{
			substrate.WithSession(sess),
			substrate.WithSystemPrompt(prompt),
			substrate.WithMaxTurns(10), //nolint:mnd // test default
		}, opts...)

		dir := substrate.NewUnifiedDirector(drv, reg, dirOpts...)

		return func(ctx context.Context, p string) (string, error) {
			return dir.Run(ctx, p, agent.ModeAuto, nil, &nopEventHandler{}, "executor")
		}, nil
	}
}

// nopEventHandler discards all events (test context doesn't need streaming).
type nopEventHandler struct{}

func (*nopEventHandler) OnText(_ string)                     {}
func (*nopEventHandler) OnThinking(_ string)                 {}
func (*nopEventHandler) OnToolCall(_ driver.ToolCall)        {}
func (*nopEventHandler) OnToolResult(_, _, _ string, _ bool) {}
func (*nopEventHandler) OnDone(_ *driver.Usage)              {}
func (*nopEventHandler) OnError(_ error)                     {}
