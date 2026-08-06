package seeder

// DefaultControlActions is the control surface an orchestrator gets when it does
// not ask for a narrower one.
//
// It is exactly the set of permissions guarding the core-provisioning gRPC
// services, and no more. TestControlPolicyCoversEveryCoreProvisioningMethod
// derives that requirement from the interceptor's own maps and fails if this
// list stops covering a method core must call, so the list cannot silently drift
// out of date the way a hand-maintained one does.
//
// Deliberately absent, and why:
//
//   - user:* and account:*:self — between them these read and mutate every end
//     user on the instance: change their email, strip their MFA, revoke their
//     sessions. An orchestrator provisions tenants and their configuration and
//     never needs them, and granting them to the most network-reachable principal
//     on the system turns one compromise into a takeover of every tenant.
//
//   - security-setting:*, ip-restriction-rule:* — the defences themselves.
//
//   - audit:read, auth_event:* — the evidence trail. Withheld together with the
//     defences so an attacker who reaches the orchestrator cannot lower the
//     controls and then erase the record of having done so.
//
//   - idp:*, registration-flow:*, branding:*, email-template:*, sms-template:*,
//     email-config:*, sms-config:* — these were in the seeded policy, but their
//     gRPC services do not exist: they are REST-only control-plane surfaces, so
//     an orchestrator driving gRPC cannot call them. Granting a surface the
//     holder cannot reach buys nothing today and silently becomes live the moment
//     someone adds the RPC. An orchestrator that genuinely needs one asks for it
//     explicitly through AllowedActions, where it is visible in the provisioning
//     request that grants it.
var DefaultControlActions = []string{
	"tenant:*",
	"tenant-setting:*",
	"service:*",
	"api:*",
	"permission:*",
	"policy:*",
	"role:*",
	"client:*",
	// Registering a federation is how an orchestrator gives a workload it
	// provisioned a platform-attested identity instead of an injected secret.
	// It was briefly dropped from this list on the reasoning that it had no gRPC
	// service — true at the time, and the wrong conclusion: the REST twin is
	// unreachable to machine callers, so removing it left an orchestrator unable
	// to configure keyless workloads at all. The gRPC service now exists.
	"workload-identity-federation:*",
}
