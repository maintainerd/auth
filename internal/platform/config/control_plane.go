package config

import (
	"crypto/x509"
	"fmt"
	"os"
	"strings"
)

// Instance roles. A maintainerd-auth instance is either the ONE IAM the
// maintainerd ecosystem is built on, or an ordinary instance a developer
// provisioned for their own application — core can create many of the latter and
// throw them away. Only the former may serve core's provisioning surface; on an
// ordinary instance those RPCs are an API a tenant admin was never meant to reach.
const (
	// InstanceRoleSystem is the ecosystem's system IAM.
	InstanceRoleSystem = "system"
	// InstanceRoleRegular is an ordinary, application-scoped instance. This is the
	// default: an instance is only the system IAM if the operator/core says so.
	InstanceRoleRegular = "regular"
)

var (
	// ControlPlaneEnabled (CONTROL_PLANE_ENABLED) opts this deployment into the
	// machine control plane — the gRPC surface core drives. Default FALSE, because
	// the default deployment is STANDALONE: an organisation dropping this in as
	// their IAM never asked for an ecosystem orchestrator and must not have the
	// listener that lets one create tenants, services and clients.
	ControlPlaneEnabled bool

	// InstanceRole (INSTANCE_ROLE) is InstanceRoleSystem or InstanceRoleRegular,
	// defaulting to InstanceRoleRegular.
	//
	// The role is CONFIGURATION fixed at provision time, not a value in the
	// database and not anything a call can carry. That is what makes it
	// unforgeable: core sets it on the container it provisions as the system IAM,
	// and there is no RPC, request field or metadata key that can change it — a
	// caller can only ever talk to whatever the instance already is. Keeping it out
	// of the database also means a caller who reaches the provisioning surface
	// cannot promote the instance by writing a row.
	InstanceRole string
)

// IsSystemInstance reports whether this instance is the ecosystem's system IAM.
//
// Anything other than an explicit InstanceRoleSystem is false, including the zero
// value seen when Init has not run: an unknown role must never be read as "yes,
// serve the provisioning surface".
func IsSystemInstance() bool {
	return InstanceRole == InstanceRoleSystem
}

// resolveInstanceRole normalizes INSTANCE_ROLE, rejecting anything unrecognised.
//
// A typo'd role is refused at startup rather than quietly treated as regular: the
// operator who wrote "System " meant to run the ecosystem IAM, and an instance
// that silently disagrees with the orchestrator that provisioned it is worse to
// debug than a process that will not boot.
func resolveInstanceRole(raw string) (string, error) {
	role := strings.ToLower(strings.TrimSpace(raw))
	switch role {
	case "":
		return InstanceRoleRegular, nil
	case InstanceRoleSystem, InstanceRoleRegular:
		return role, nil
	default:
		return "", fmt.Errorf("invalid INSTANCE_ROLE %q, must be one of: %v", raw, []string{InstanceRoleSystem, InstanceRoleRegular})
	}
}

// validateControlPlaneTLS refuses to start a control plane that cannot verify who
// its caller is.
//
// With the control plane on, the channel that creates tenants, services and
// clients has to PROVE the peer is core, not take a bearer token's word for it —
// so mTLS is not an operator choice here and there is no switch that turns it
// off. A missing or unreadable client CA is therefore fatal rather than a silent
// downgrade to server-side TLS: the downgrade leaves the same listener open to
// anyone who can reach the port, which is exactly the posture this replaces.
func validateControlPlaneTLS() error {
	if !ControlPlaneEnabled {
		return nil
	}
	if strings.TrimSpace(GRPCTLSCertFile) == "" || strings.TrimSpace(GRPCTLSKeyFile) == "" {
		return fmt.Errorf("GRPC_TLS_CERT_FILE and GRPC_TLS_KEY_FILE are required when CONTROL_PLANE_ENABLED=true")
	}
	if strings.TrimSpace(GRPCClientCAFile) == "" {
		return fmt.Errorf("GRPC_CLIENT_CA_FILE is required when CONTROL_PLANE_ENABLED=true: the control plane only accepts verified client certificates")
	}
	caBytes, err := os.ReadFile(GRPCClientCAFile)
	if err != nil {
		return fmt.Errorf("read gRPC client CA file %q: %w", GRPCClientCAFile, err)
	}
	// Parse here as well as at listener setup: an unparsable CA yields an EMPTY
	// pool, and Go's RequireAndVerifyClientCert with an empty pool rejects every
	// peer — the control plane would boot "secure" and be uniformly broken, which
	// reads like a networking fault rather than a config error.
	if !x509.NewCertPool().AppendCertsFromPEM(caBytes) {
		return fmt.Errorf("gRPC client CA file %q contains no PEM certificate", GRPCClientCAFile)
	}
	return nil
}
