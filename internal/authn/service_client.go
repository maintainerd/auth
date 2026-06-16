package authn

// resolveClient resolves the auth client for register/login/invite operations.
//
// Resolution priority:
//  1. clientID provided → global lookup by clients.identifier (now unique).
//     The tenant is derived from the client's IdentityProvider.TenantID,
//     ensuring the user is always placed in the correct tenant.
//  2. tenantID provided → find the is_system client under that tenant.
//  3. Neither provided → default to the system tenant's system client.
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
