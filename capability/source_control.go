package capability

import "context"

// SourceControlPort provides repository operations.
type SourceControlPort interface {
	Clone(ctx context.Context, repo string) error
	Branch(ctx context.Context, name string) error
	Commit(ctx context.Context, msg string) error
	CreatePR(ctx context.Context, title, body string) (string, error)
}
