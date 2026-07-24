import { useQuery } from '@tanstack/react-query'
import { fetchOAuthConnections } from '@/services/api/oauth'
import { currentPublicAuthContext } from '@/utils/clientContext'

// Login options are a property of the CLIENT, not of any registration flow: the
// backend deliberately takes no registration_flow here so that adding or
// disabling a flow can never change the login page. Signup field requirements
// come from useRegistrationContext instead.
export function useOAuthConnections(enabled = true) {
  const clientId = currentPublicAuthContext().clientId
  return useQuery({
    queryKey: ['oauth-connections', clientId],
    queryFn: () => fetchOAuthConnections(clientId!),
    enabled: enabled && Boolean(clientId),
    staleTime: 30_000,
  })
}
