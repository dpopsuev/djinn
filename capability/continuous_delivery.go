package capability

import "context"

// DeployStatus describes the state of a deployment.
type DeployStatus struct {
	State   string
	Message string
}

// ContinuousDeliveryPort syncs desired state to target environments.
type ContinuousDeliveryPort interface {
	Deploy(ctx context.Context, artifact ArtifactRef, env string) error
	Status(ctx context.Context, deployID string) (DeployStatus, error)
}
