import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Check, ShieldCheck, User, Shield, Monitor, Link2, Settings } from 'lucide-react'
import { Button } from '@/components/ui/button'
import LoginLayout from '@/components/layout/LoginLayout'
import { useAuth } from '@/hooks/useAuth'
import { useTenant } from '@/hooks/useTenant'
import { useToast } from '@/hooks/useToast'
import { getRequestId } from '@/utils/oauthRedirect'
import { ACCOUNT_ROUTE, resumeOAuthContinuation } from '@/utils/oauthContinuation'
import { fetchMFAStatus } from '@/services/api/mfa'

export default function LoginSuccessPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { logout, isAuthenticated } = useAuth()
  const { currentTenant } = useTenant()
  const { showError } = useToast()
  const redirectedRef = useRef(false)
  const [mfaNudgeDismissed, setMfaNudgeDismissed] = useState(false)
  const requestId = getRequestId(searchParams)

  // MFA enrollment nudge: after a completed sign-in, if the user has no second
  // factor enrolled we gently prompt them to set one up. Non-blocking — the
  // session is fully usable and the prompt can be dismissed.
  const { data: mfaStatus } = useQuery({
    queryKey: ['mfa', 'status'],
    queryFn: fetchMFAStatus,
    enabled: isAuthenticated,
    staleTime: 60_000,
  })

  const hasNoFactors =
    !!mfaStatus &&
    !mfaStatus.is_totp_enabled &&
    !mfaStatus.is_webauthn_enabled &&
    !mfaStatus.is_sms_available &&
    !mfaStatus.is_email_otp_available
  const showMfaNudge = hasNoFactors && !mfaNudgeDismissed

  // The single OAuth2 continuation point. Resumes the pending authorize request
  // with the request_id handle (primary) or the legacy sessionStorage return-to /
  // invite callback (defensive fallback); with nothing to resume it is a direct
  // login → the account profile.
  useEffect(() => {
    if (redirectedRef.current) return
    redirectedRef.current = true
    void (async () => {
      if (await resumeOAuthContinuation(requestId, navigate)) return
      navigate(ACCOUNT_ROUTE, { replace: true })
    })()
  }, [navigate, requestId])

  const handleLogout = async () => {
    try {
      await logout()
      navigate('/login')
    } catch (error) {
      showError(error, 'Logout failed')
      navigate('/login')
    }
  }

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <div className="flex flex-col items-center gap-6 text-center">
        <div className="flex size-16 items-center justify-center rounded-full bg-emerald-500/10">
          <Check className="size-8 text-emerald-600" />
        </div>
        <div className="space-y-2">
          <h1 className="text-2xl font-bold">You&apos;re signed in</h1>
          <p className="text-muted-foreground">
            You can continue from this session or sign out when you are done.
          </p>
        </div>
        {showMfaNudge && (
          <div className="w-full max-w-sm rounded-lg border border-primary/30 bg-primary/5 p-4 text-left">
            <div className="flex items-start gap-3">
              <ShieldCheck className="mt-0.5 size-5 shrink-0 text-primary" />
              <div className="space-y-1">
                <p className="text-sm font-medium">Add an extra layer of security</p>
                <p className="text-sm text-muted-foreground">
                  Your account has no two-factor authentication set up. Enable it to
                  better protect your account.
                </p>
              </div>
            </div>
            <div className="mt-3 flex gap-2">
              <Button size="sm" onClick={() => navigate('/account/mfa')}>
                Set up MFA
              </Button>
              <Button size="sm" variant="ghost" onClick={() => setMfaNudgeDismissed(true)}>
                Not now
              </Button>
            </div>
          </div>
        )}
        {/* Account sections */}
        <div className="mt-6 grid grid-cols-2 gap-3">
          {[
            { href: '/account/profile', label: 'Profile', description: 'Names & avatar', icon: User },
            { href: '/account/security', label: 'Security', description: 'Password & email', icon: Shield },
            { href: '/account/sessions', label: 'Sessions', description: 'Active sign-ins', icon: Monitor },
            { href: '/account/mfa', label: 'Two-Factor', description: 'MFA settings', icon: ShieldCheck },
            { href: '/account/identities', label: 'Linked Accounts', description: 'Social logins', icon: Link2 },
            { href: '/account/settings', label: 'Preferences', description: 'App settings', icon: Settings },
          ].map(({ href, label, description, icon: Icon }) => (
            <Link
              key={href}
              to={href}
              className="flex items-center gap-3 rounded-lg border bg-card p-3 text-sm hover:bg-muted/50 transition-colors"
            >
              <Icon className="size-4 text-muted-foreground shrink-0" />
              <div>
                <p className="font-medium">{label}</p>
                <p className="text-xs text-muted-foreground">{description}</p>
              </div>
            </Link>
          ))}
        </div>
        <Button variant="outline" onClick={handleLogout}>
          Sign out
        </Button>
        <Link
          to="/account/erasure"
          className="text-xs text-muted-foreground/60 hover:text-destructive transition-colors"
        >
          Delete account
        </Link>
      </div>
    </LoginLayout>
  )
}
