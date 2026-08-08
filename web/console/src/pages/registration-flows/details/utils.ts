/**
 * Composes the public registration link for a flow.
 *
 * Verified against the backend: the public register endpoint reads `client_id`
 * and `registration_flow` from the query string
 * (internal/authn/handler_register.go) and resolves the client with
 * `clientRepo.FindByIdentifier(client_id)` (service_client.go resolveClient /
 * resolvePublicClient). So `client_id` MUST be the client's OAuth **identifier**,
 * not its UUID — passing the UUID resolves no client and the registration is
 * rejected. The flow itself is resolved by its NAME (registrationFlowByName),
 * scoped to that client AND tenant.
 *
 * @param identityUrl       the tenant's identity origin (trailing slash tolerated)
 * @param clientIdentifier  the client's OAuth identifier (NOT the client UUID)
 * @param flowName          the registration flow's slug-shaped name
 */
export function buildRegistrationUrl(
  identityUrl: string,
  clientIdentifier: string,
  flowName: string,
): string {
  const origin = identityUrl.replace(/\/$/, '')
  const params = new URLSearchParams({
    client_id: clientIdentifier,
    registration_flow: flowName,
  })
  return `${origin}/register?${params.toString()}`
}
