import { useSearchParams } from 'react-router-dom'
import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link2, CheckCircle2, AlertCircle, ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import LoginLayout from '@/components/layout/LoginLayout'
import { fetchAccount, confirmAccountLink, resumeBrokerSession } from '@/services/api/auth'
import { useToast } from '@/hooks/useToast'

export default function AccountLinkPage() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''
  const provider = searchParams.get('provider') ?? ''
  const email = searchParams.get('email') ?? ''
  const brokerSession = searchParams.get('broker_session') ?? ''

  const { showError } = useToast()
  const [done, setDone] = useState(false)
  const [errorMsg, setErrorMsg] = useState('')

  const { data: account, isLoading: accountLoading } = useQuery({
    queryKey: ['account'],
    queryFn: fetchAccount,
    retry: false,
  })

  const linkMutation = useMutation({
    mutationFn: async () => {
      await confirmAccountLink(token)
      const result = await resumeBrokerSession({
        broker_session_id: brokerSession,
        account_link_token: token,
      })
      return result
    },
    onSuccess: (result) => {
      setDone(true)
      // Small delay so user sees success state
      setTimeout(() => {
        window.location.href = result.redirect_url
      }, 1500)
    },
    onError: (error: unknown) => {
      const msg = error instanceof Error ? error.message : 'Failed to link accounts'
      setErrorMsg(msg)
      showError(error)
    },
  })

  if (!token || !brokerSession) {
    return (
      <LoginLayout>
        <div className="max-w-md mx-auto pt-12 px-4 text-center">
          <AlertCircle className="h-12 w-12 mx-auto mb-4 text-destructive" />
          <h1 className="text-xl font-semibold mb-2">Invalid Link</h1>
          <p className="text-muted-foreground mb-6">This account link URL is incomplete or has expired.</p>
          <Button variant="outline" asChild>
            <a href="/login">
              <ArrowLeft className="h-4 w-4 mr-2" />Back to sign in
            </a>
          </Button>
        </div>
      </LoginLayout>
    )
  }

  const providerLabel = provider
    ? provider.charAt(0).toUpperCase() + provider.slice(1)
    : 'Social'

  if (done) {
    return (
      <LoginLayout>
        <div className="max-w-md mx-auto pt-12 px-4 text-center">
          <CheckCircle2 className="h-12 w-12 mx-auto mb-4 text-emerald-600" />
          <h1 className="text-xl font-semibold mb-2">Accounts linked!</h1>
          <p className="text-muted-foreground">Redirecting you to the app…</p>
        </div>
      </LoginLayout>
    )
  }

  return (
    <LoginLayout>
      <div className="max-w-md mx-auto pt-12 px-4">
        <div className="flex justify-center mb-6">
          <div className="flex size-14 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <Link2 className="h-7 w-7" />
          </div>
        </div>
        <h1 className="text-2xl font-bold text-center mb-2">
          Link your {providerLabel} account
        </h1>
        <p className="text-center text-muted-foreground mb-8">
          Your <span className="font-medium text-foreground">{providerLabel}</span> account uses{' '}
          {email && (
            <>
              <span className="font-medium text-foreground">{email}</span>{' '}
            </>
          )}
          which is already registered. Sign in to confirm you own this account and link them.
        </p>

        {accountLoading ? (
          <p className="text-center text-muted-foreground">Checking sign-in status…</p>
        ) : !account ? (
          <div className="space-y-4">
            <p className="text-center text-sm text-muted-foreground">
              You need to be signed in to confirm this link.
            </p>
            <Button className="w-full" asChild>
              <a
                href={`/login?return_to=${encodeURIComponent(
                  window.location.pathname + window.location.search,
                )}`}
              >
                Sign in to continue
              </a>
            </Button>
            <div className="text-center">
              <a href="/login" className="text-sm text-muted-foreground hover:text-foreground">
                <ArrowLeft className="h-3 w-3 inline mr-1" />Back to sign in
              </a>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            {errorMsg && (
              <div className="flex items-start gap-2 rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
                <AlertCircle className="h-4 w-4 mt-0.5 shrink-0" />
                <span>{errorMsg}</span>
              </div>
            )}
            <Button
              className="w-full"
              onClick={() => linkMutation.mutate()}
              disabled={linkMutation.isPending}
            >
              {linkMutation.isPending ? 'Linking accounts…' : `Link ${providerLabel} account`}
            </Button>
            <div className="text-center">
              <a href="/login" className="text-sm text-muted-foreground hover:text-foreground">
                <ArrowLeft className="h-3 w-3 inline mr-1" />Cancel
              </a>
            </div>
          </div>
        )}
      </div>
    </LoginLayout>
  )
}
