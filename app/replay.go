package app

import (
	"context"
	"fmt"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
)

// ReplayHistory sends session entries into a driver to restore context.
// User entries with Blocks (e.g. tool_result) go through SendRich
// to preserve structured content. Plain text goes through Send.
// Returns an error if any replay step fails — caller decides recovery.
func ReplayHistory(ctx context.Context, d driver.ChatDriver, sess *session.Session) error {
	for _, entry := range sess.Entries() {
		switch entry.Role {
		case driver.RoleUser:
			var err error
			if len(entry.Blocks) > 0 {
				err = d.SendRich(ctx, driver.RichMessage{
					Role:   entry.Role,
					Blocks: entry.Blocks,
				})
			} else {
				err = d.Send(ctx, driver.Message{
					Role:    entry.Role,
					Content: entry.TextContent(),
				})
			}
			if err != nil {
				return fmt.Errorf("replay user entry: %w", err)
			}
		case driver.RoleAssistant:
			d.AppendAssistant(driver.RichMessage{
				Role:    entry.Role,
				Content: entry.TextContent(),
				Blocks:  entry.Blocks,
			})
		}
	}
	return nil
}
