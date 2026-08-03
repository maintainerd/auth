/**
 * Post-logout landing page.
 *
 * The seeder registers `<identity host>/logout` as the client's only allowed
 * post_logout_redirect_uri, and RP-initiated logout validates against exactly
 * that value — but the route did not exist, so every completed sign-out landed
 * on the 404 page. This is where the browser comes back to rest.
 */
import { useNavigate } from 'react-router-dom'
import { CheckCircle2 } from 'lucide-react'
import LoginLayout from '@/components/layout/LoginLayout'
import { Button } from '@/components/ui/button'
import AuthPageHeading from '@/components/auth/AuthPageHeading'
import { useTenant } from '@/hooks/useTenant'
import { useLoginPageCopy } from '@/hooks/useLoginPageCopy'

export default function LogoutPage() {
  const navigate = useNavigate()
  const { currentTenant } = useTenant()
  const copy = useLoginPageCopy('oauth-end-session')

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <div className="space-y-6" role="status" aria-live="polite">
        <div className="space-y-3">
          <div className="flex justify-center">
            <div className="flex size-14 items-center justify-center rounded-full bg-emerald-500/10">
              <CheckCircle2 className="size-7 text-emerald-600" />
            </div>
          </div>
          <AuthPageHeading title={copy.title} subtitle={copy.subtitle} />
        </div>

        <div className="space-y-4">
          <Button className="w-full" onClick={() => navigate('/login', { replace: true })}>
            Sign in again
          </Button>
        </div>
      </div>
    </LoginLayout>
  )
}
