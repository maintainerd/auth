import { useEffect, useRef, useState } from 'react'
import { LogIn } from 'lucide-react'
import { Button } from '@/components/ui/button'
import LoginLayout from '@/components/layout/LoginLayout'
import { useTenant } from '@/hooks/useTenant'
import { useToast } from '@/hooks/useToast'
import { startConsoleOAuthLogin } from '@/services/api/oauth'

/**
 * Public login page for the console.
 *
 * This is an unprotected route: it never auto-redirects. It simply offers a
 * Sign in button that starts the hosted-identity OAuth2 flow. Logging out lands
 * the user here, so logout actually sticks instead of immediately bouncing back
 * through SSO. Automatic OAuth redirects only happen on protected routes.
 *
 * The target tenant is identified by the host subdomain and resolved into
 * `currentTenant` on app bootstrap. After logout the tenant context is cleared,
 * so we re-resolve the tenant here for the Sign in button.
 */
const LoginPage = () => {
  const { currentTenant, consoleClient, identityUrl, initializeFromHost } = useTenant()
  const { showError } = useToast()
  const [redirecting, setRedirecting] = useState(false)
  const tenantFetchedRef = useRef(false)

  // After logout the tenant context is cleared. Re-resolve the tenant (the host
  // subdomain drives which one) so the Sign in button can start OAuth without a
  // full page reload.
  useEffect(() => {
    if (currentTenant?.name || tenantFetchedRef.current) return
    tenantFetchedRef.current = true
    initializeFromHost().catch(() => {
      /* surfaced via service-unavailable handling elsewhere */
    })
  }, [currentTenant?.name, initializeFromHost])

  const tenantIdentifier = currentTenant?.name
  const companyName = currentTenant?.branding?.company_name || 'Maintainerd IAM'

  const handleLogin = async () => {
    if (!tenantIdentifier) return
    setRedirecting(true)
    try {
      await startConsoleOAuthLogin({
        tenantId: tenantIdentifier,
        clientId: consoleClient?.client_id,
        identityUrl: identityUrl ?? undefined,
      })
    } catch (error) {
      console.error('[console-oauth] sign-in failed to start', error)
      showError('Unable to start sign in. Please try again.')
      setRedirecting(false)
    }
  }

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <div className="flex flex-col gap-8 text-center">
        <div className="flex flex-col items-center gap-3">
          <h1 className="text-2xl font-semibold tracking-tight">{companyName} Console</h1>
          <p className="max-w-xs text-sm text-muted-foreground">
            Sign in to administer tenants, users, clients, and security settings.
          </p>
        </div>
        <Button
          className="w-full"
          onClick={handleLogin}
          disabled={redirecting || !tenantIdentifier}
        >
          <LogIn className="mr-2 size-4" />
          {redirecting ? 'Redirecting…' : 'Sign in'}
        </Button>
      </div>
    </LoginLayout>
  )
}

export default LoginPage
