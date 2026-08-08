import { useEffect } from 'react'
import { Loader2 } from 'lucide-react'
import LoginLayout from '@/components/layout/LoginLayout'
import { oauthEndSessionURL } from '@/services/api/oauth'
import { useTenant } from '@/hooks/useTenant'

export default function OAuthEndSessionPage() {
  const { currentTenant } = useTenant()

  useEffect(() => {
    window.location.assign(oauthEndSessionURL(window.location.search))
  }, [])

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <div className="flex flex-col gap-8 text-center" role="status" aria-live="polite">
        <div className="flex flex-col items-center gap-3">
          <div className="flex size-14 items-center justify-center rounded-full bg-primary/10">
            <Loader2 className="size-7 animate-spin text-primary" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Signing out</h1>
          <p className="max-w-xs text-sm text-muted-foreground">Completing the logout request.</p>
        </div>
      </div>
    </LoginLayout>
  )
}
