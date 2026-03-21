// Package capability defines hexagonal driven port interfaces for DevOps
// capabilities. Each capability is a port; tools are swappable adapters
// connected via MCP. This package contains only interfaces and value
// types — zero imports from Djinn.
package capability

import "errors"

// DevOps lifecycle phases.
const (
	PhasePlan    = "plan"
	PhaseCode    = "code"
	PhaseVerify  = "verify"
	PhasePackage = "package"
	PhaseDeploy  = "deploy"
	PhaseObserve = "observe"
	PhaseReact   = "react"
)

// Port names for capability registry lookup.
const (
	PortSourceControl      = "source_control"
	PortTestRunner         = "test_runner"
	PortStaticAnalysis     = "static_analysis"
	PortMetricsSource      = "metrics_source"
	PortArtifactBuild      = "artifact_build"
	PortContinuousDelivery = "continuous_delivery"
	PortSecurityScanning   = "security_scanning"
)

// ErrPortNotRegistered is returned when a requested port is not in the registry.
var ErrPortNotRegistered = errors.New("capability port not registered")
