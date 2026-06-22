package authn

// resolveClient resolves the auth client for register/login/invite operations.
//
// HTTP handlers enforce the surface contract. This resolver retains the
// default for trusted/internal callers that do not originate at those routes.
func resolveClient(
	clientRepo ClientRepository,
	clientID *string,
	tenantID *string,
) (*Client, error) {
	if clientID != nil && *clientID != "" {
		return clientRepo.FindByIdentifier(*clientID)
	}
	if tenantID != nil && *tenantID != "" {
		return clientRepo.FindSystemByTenantIdentifier(*tenantID)
	}
	return clientRepo.FindSystem()
}
