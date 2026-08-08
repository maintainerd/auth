import { useMemo } from 'react'
import { useTenant } from '@/hooks/useTenant'
import { loginPageCopy, type LoginPageCopy, type LoginPageId } from '@/lib/branding/loginPageContent'

/**
 * Title/subtitle for an auth page, resolved against the tenant's branding.
 *
 * Pages call this instead of hardcoding their heading so the copy an operator
 * writes (and previews) in the console branding editor is the copy that
 * renders. Falls back to the shared defaults when the tenant has no override.
 */
export function useLoginPageCopy(id: LoginPageId): LoginPageCopy {
  const { currentTenant } = useTenant()
  const metadata = currentTenant?.branding?.metadata
  return useMemo(() => loginPageCopy(metadata, id), [metadata, id])
}
