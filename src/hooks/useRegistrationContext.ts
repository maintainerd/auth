import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { fetchRegistrationContext } from '@/services/api/auth'
import { currentPublicAuthContext } from '@/utils/clientContext'
import { ApiError } from '@/services/api/client'
import type { RegistrationContext, RegistrationRequiredField } from '@/services/api/auth/types'

const KNOWN_FIELDS: RegistrationRequiredField[] = ['fullname', 'email', 'phone']

/**
 * The resolved signup requirements, as a state the form can branch on.
 *
 * `invalid` is an AUTHORITATIVE refusal — the link names a flow this client
 * cannot use (unknown, renamed, deactivated, or system/invite-only). The server
 * collapses all of those into one 404 on purpose, so the UI must not try to
 * distinguish them either.
 *
 * `unavailable` is a transport failure. The distinction matters: an authoritative
 * refusal must block signup, while a network blip must not — the server stays the
 * enforcement point, so degrading to the plain form is safe.
 */
export type RegistrationContextState =
  | { status: 'none' }
  | { status: 'loading' }
  | { status: 'ready'; context: RegistrationContext }
  | { status: 'invalid' }
  | { status: 'unavailable'; retry: () => void }

export function useRegistrationContext(): RegistrationContextState {
  const [searchParams] = useSearchParams()
  const registrationFlow = searchParams.get('registration_flow') || undefined
  const clientId = currentPublicAuthContext().clientId

  const query = useQuery({
    queryKey: ['registration-context', clientId, registrationFlow],
    queryFn: () => fetchRegistrationContext(clientId!, registrationFlow),
    enabled: Boolean(clientId) && Boolean(registrationFlow),
    staleTime: 0,
    // A 404 is the server's verdict on the link; retrying cannot change it.
    retry: (failureCount, error) =>
      !(error instanceof ApiError && error.status === 404) && failureCount < 2,
  })

  // No flow in the link is the ordinary case: plain self-service signup, with the
  // tenant's own policy applied server-side.
  if (!registrationFlow || !clientId) return { status: 'none' }

  if (query.isPending) return { status: 'loading' }

  if (query.isError) {
    if (query.error instanceof ApiError && query.error.status === 404) {
      return { status: 'invalid' }
    }
    return { status: 'unavailable', retry: () => void query.refetch() }
  }


  const retry = () => void query.refetch()

  const context = query.data
  if (!context) return { status: 'unavailable', retry }

  // A required field this build does not know how to collect would produce a 400
  // the user cannot resolve, so treat it as an invalid link rather than rendering
  // a form that cannot succeed.
  const unsupported = context.required_fields.some((field) => !KNOWN_FIELDS.includes(field))
  if (unsupported) return { status: 'invalid' }

  return { status: 'ready', context }
}
