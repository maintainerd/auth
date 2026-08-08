import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { AlertCircle, Check, Loader2, X } from 'lucide-react'
import LoginLayout from '@/components/layout/LoginLayout'
import { Button } from '@/components/ui/button'
import { fetchOAuthConsentChallenge, submitOAuthConsent } from '@/services/api/oauth'
import type { OAuthConsentChallenge } from '@/services/api/oauth/types'
import { useTenant } from '@/hooks/useTenant'
import { useToast } from '@/hooks/useToast'

export default function OAuthConsentPage() {
  const { challengeId = '' } = useParams()
  const navigate = useNavigate()
  const { currentTenant } = useTenant()
  const { showError } = useToast()
  const [challenge, setChallenge] = useState<OAuthConsentChallenge | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState<'approve' | 'deny' | null>(null)

  useEffect(() => {
    let alive = true
    async function load() {
      try {
        const result = await fetchOAuthConsentChallenge(challengeId)
        if (alive) setChallenge(result)
      } catch (err) {
        if (alive) setError(err instanceof Error ? err.message : 'Consent request unavailable.')
      }
    }
    load()
    return () => { alive = false }
  }, [challengeId])

  const decide = async (approved: boolean) => {
    setPending(approved ? 'approve' : 'deny')
    try {
      const result = await submitOAuthConsent(challengeId, approved)
      window.location.assign(result.redirect_uri)
    } catch (err) {
      showError(err)
      setPending(null)
    }
  }

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <div className="flex flex-col gap-8">
        {!challenge && !error && (
          <div className="flex flex-col items-center gap-3 text-center" role="status">
            <div className="flex size-14 items-center justify-center rounded-full bg-primary/10">
              <Loader2 className="size-7 animate-spin text-primary" />
            </div>
            <h1 className="text-2xl font-semibold tracking-tight">Loading request</h1>
          </div>
        )}

        {error && (
          <div className="flex flex-col gap-6 text-center" role="alert">
            <div className="flex flex-col items-center gap-3">
              <div className="flex size-14 items-center justify-center rounded-full bg-destructive/10">
                <AlertCircle className="size-7 text-destructive" />
              </div>
              <h1 className="text-2xl font-semibold tracking-tight">Consent unavailable</h1>
              <p className="text-sm text-muted-foreground">{error}</p>
            </div>
            <Button variant="outline" onClick={() => navigate('/login', { replace: true })}>Back</Button>
          </div>
        )}

        {challenge && (
          <>
            <div className="flex flex-col items-center gap-2 text-center">
              <h1 className="text-2xl font-semibold tracking-tight">{challenge.client_name}</h1>
              <p className="text-sm text-muted-foreground">wants access to your account.</p>
            </div>

            <div className="grid gap-2">
              {challenge.scopes.map((scope) => (
                <div key={scope} className="flex items-center gap-3 rounded-md border p-3 text-sm">
                  <Check className="size-4 text-green-600" />
                  <span className="font-mono">{scope}</span>
                </div>
              ))}
            </div>

            <div className="flex items-center gap-2">
              <Button
                className="flex-1"
                onClick={() => decide(true)}
                disabled={pending !== null}
              >
                {pending === 'approve' ? 'Approving...' : 'Allow'}
              </Button>
              <Button
                variant="outline"
                className="flex-1"
                onClick={() => decide(false)}
                disabled={pending !== null}
              >
                <X className="mr-2 size-4" />
                {pending === 'deny' ? 'Denying...' : 'Deny'}
              </Button>
            </div>
          </>
        )}
      </div>
    </LoginLayout>
  )
}
