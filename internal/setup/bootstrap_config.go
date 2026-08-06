package setup

import "github.com/maintainerd/maintainerd-auth/internal/platform/config"

// Whether this instance is orchestrator-provisioned is read straight from
// configuration, because that is the whole of the fact.
//
// There is deliberately no table tracking whether the credential has been
// "spent". That state is already recorded, durably and shared across replicas,
// by the existence of the system tenant — which is exactly what ensureSetupOpen
// checks, and what the single-system-tenant constraint settles when two replicas
// race. A second copy of that boolean could only ever drift from it.

// bootstrapControlPlaneEnabled reports whether the gRPC control plane is on,
// which is what distinguishes an orchestrated instance from a standalone one.
//
// A variable rather than a plain function so a test can drive both modes without
// mutating process-wide config.
var bootstrapControlPlaneEnabled = func() bool { return config.ControlPlaneEnabled }

// bootstrapCredentialConfigured reports whether this instance was provisioned
// with a bootstrap credential (injected by the orchestrator, e.g. through
// maintainerd-docker).
func bootstrapCredentialConfigured() bool { return config.SetupBootstrapToken != "" }
