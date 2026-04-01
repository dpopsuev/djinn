package builders

import "github.com/dpopsuev/djinn/terminal"

// DjinnBuilder provides fluent construction of terminal.Djinn for tests.
type DjinnBuilder struct {
	djinn *terminal.Djinn
}

// NewDjinnBuilder starts building a Djinn instance.
func NewDjinnBuilder() *DjinnBuilder {
	return &DjinnBuilder{djinn: terminal.NewDjinn()}
}

// WithOperation sets the active operation (ask/plan/agent).
func (b *DjinnBuilder) WithOperation(op string) *DjinnBuilder {
	b.djinn.SetOperation(op)
	return b
}

// WithCapacity sets the maximum concurrent agent count.
func (b *DjinnBuilder) WithCapacity(n int) *DjinnBuilder {
	b.djinn.SetCapacity(n)
	return b
}

// Build returns the constructed Djinn instance.
func (b *DjinnBuilder) Build() *terminal.Djinn {
	return b.djinn
}
