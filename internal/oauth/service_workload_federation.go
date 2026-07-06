package oauth

// subjectTokenTypeJWT is the RFC 8693 subject_token_type for a generic JWT.
// A workload identity federation exchange presents an external OIDC token with
// this subject_token_type.
const subjectTokenTypeJWT = "urn:ietf:params:oauth:token-type:jwt"

// workloadIdentityExchanger performs workload identity federation token
// exchanges (section 3.21). It is set once at startup via
// SetWorkloadIdentityExchanger so the oauth package never imports the
// federation domain. When nil, the token endpoint only serves the standard
// RFC 8693 token-exchange path.
var workloadIdentityExchanger WorkloadIdentityExchanger

// SetWorkloadIdentityExchanger installs the workload identity federation
// exchanger used by the token endpoint. Call once during app startup before
// serving requests.
func SetWorkloadIdentityExchanger(e WorkloadIdentityExchanger) {
	workloadIdentityExchanger = e
}
