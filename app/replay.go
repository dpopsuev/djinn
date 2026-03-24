package app

import (
	"context"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
)

// ReplayHistory sends session entries into a driver to restore context.
// User entries with Blocks (e.g. tool_result) go through SendRich
// to preserve structured content. Plain text goes through Send.
func ReplayHistory(ctx context.Context, d driver.ChatDriver, sess *session.Session) {
	for _, entry := range sess.Entries() {
		switch entry.Role {
		case driver.RoleUser:
			if len(entry.Blocks) > 0 {
				d.SendRich(ctx, driver.RichMessage{ //nolint:errcheck
					Role:   entry.Role,
					Blocks: entry.Blocks,
				})
			} else {
				d.Send(ctx, driver.Message{ //nolint:errcheck
					Role:    entry.Role,
					Content: entry.TextContent(),
				})
			}
		case driver.RoleAssistant:
			d.AppendAssistant(driver.RichMessage{
				Role:    entry.Role,
				Content: entry.TextContent(),
				Blocks:  entry.Blocks,
			})
		}
	}
}
