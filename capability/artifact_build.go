package capability

import "context"

// ArtifactRef identifies a built artifact.
type ArtifactRef struct {
	ID     string
	Digest string
	URI    string
}

// ArtifactBuildPort compiles code into deployable artifacts.
type ArtifactBuildPort interface {
	Build(ctx context.Context, target string) (ArtifactRef, error)
}
