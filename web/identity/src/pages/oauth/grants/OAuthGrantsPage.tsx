import { useEffect, useState } from 'react'
import { AlertCircle, Loader2, Trash2 } from 'lucide-react'
import LoginLayout from '@/components/layout/LoginLayout'
import { Button } from '@/components/ui/button'
import { listOAuthConsentGrants, revokeOAuthConsentGrant } from '@/services/api/oauth'
import type { OAuthConsentGrant } from '@/services/api/oauth/types'
import { useTenant } from '@/hooks/useTenant'
import { useToast } from '@/hooks/useToast'

export default function OAuthGrantsPage() {
  const { currentTenant } = useTenant()
  const { showError, showSuccess } = useToast()
  const [grants, setGrants] = useState<OAuthConsentGrant[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [revoking, setRevoking] = useState<string | null>(null)

  const load = async () => {
    setLoading(true)
    setError(null)
    try {
      setGrants(await listOAuthConsentGrants())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load connected applications.')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const revoke = async (grantId: string) => {
    setRevoking(grantId)
    try {
      await revokeOAuthConsentGrant(grantId)
      setGrants((items) => items.filter((item) => item.consent_grant_id !== grantId))
      showSuccess('Access revoked')
    } catch (err) {
      showError(err)
    } finally {
      setRevoking(null)
    }
  }

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <div className="flex flex-col gap-8">
        <div className="flex flex-col items-center gap-2 text-center">
          <h1 className="text-2xl font-semibold tracking-tight">Connected apps</h1>
          <p className="text-sm text-muted-foreground">Review applications you have authorized.</p>
        </div>

        {loading && (
          <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground" role="status">
            <Loader2 className="size-4 animate-spin" />
            Loading
          </div>
        )}

        {error && (
          <div role="alert" className="flex items-start gap-2.5 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
            <AlertCircle className="mt-0.5 size-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {!loading && !error && grants.length === 0 && (
          <p className="rounded-md border p-4 text-center text-sm text-muted-foreground">
            No connected applications.
          </p>
        )}

        <div className="grid gap-3">
          {grants.map((grant) => (
            <div key={grant.consent_grant_id} className="rounded-md border p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0 space-y-2">
                  <p className="font-medium">{grant.client_name}</p>
                  <div className="flex flex-wrap gap-1.5">
                    {grant.scopes.map((scope) => (
                      <span key={scope} className="rounded border px-2 py-0.5 font-mono text-xs text-muted-foreground">
                        {scope}
                      </span>
                    ))}
                  </div>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => revoke(grant.consent_grant_id)}
                  disabled={revoking === grant.consent_grant_id}
                  aria-label={`Revoke access for ${grant.client_name}`}
                >
                  <Trash2 className="size-4" />
                </Button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </LoginLayout>
  )
}
