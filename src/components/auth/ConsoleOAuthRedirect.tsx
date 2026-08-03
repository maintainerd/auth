import { useEffect, useRef } from 'react'
import { useTenant } from '@/hooks/useTenant'
import AppLoadingScreen from '@/components/layout/AppLoadingScreen'
import { startConsoleOAuthLogin, tryConsoleSilentOAuthLogin } from '@/services/api/oauth'

export function ConsoleOAuthRedirect({ returnTo }: { returnTo: string }) {
  const { currentTenant, consoleClient, identityUrl } = useTenant()
  const startedRef = useRef(false)

  const clientId = consoleClient?.client_id

  useEffect(() => {
    // The tenant hint is the tenant `name`; the system tenant carries its own
    // name here too. The console client + per-tenant identity origin come from
    // the tenant-bootstrap response.
    const tenantId = currentTenant?.name
    if (startedRef.current || !tenantId) return
    startedRef.current = true

    const run = async () => {
      const options = {
        tenantId,
        clientId,
        identityUrl: identityUrl ?? undefined,
        returnTo,
      }
      const silent = await tryConsoleSilentOAuthLogin(options)
      if (silent) {
        // Reload the protected URL so bootstrap reads the freshly stored token.
        window.location.replace(returnTo || window.location.href)
        return
      }
      await startConsoleOAuthLogin(options)
    }
    run().catch((error) => {
        console.error('[console-oauth] failed to start hosted identity OAuth flow', error)
        window.location.replace('/service-unavailable')
      })
  }, [currentTenant?.name, clientId, identityUrl, returnTo])

  return <AppLoadingScreen branding={currentTenant?.branding} />
}
