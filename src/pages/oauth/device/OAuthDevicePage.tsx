import { useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { AlertCircle, CheckCircle2 } from 'lucide-react'
import LoginLayout from '@/components/layout/LoginLayout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { approveOAuthDevice, denyOAuthDevice } from '@/services/api/oauth'
import { useTenant } from '@/hooks/useTenant'
import { useToast } from '@/hooks/useToast'

export default function OAuthDevicePage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { currentTenant } = useTenant()
  const { showError } = useToast()
  const initialCode = useMemo(() => searchParams.get('user_code') || '', [searchParams])
  const [userCode, setUserCode] = useState(initialCode)
  const [status, setStatus] = useState<'idle' | 'approved' | 'denied'>('idle')
  const [pending, setPending] = useState<'approve' | 'deny' | null>(null)
  const [error, setError] = useState<string | null>(null)

  const submit = async (approved: boolean) => {
    const code = userCode.trim()
    if (!code) {
      setError('User code is required.')
      return
    }
    setError(null)
    setPending(approved ? 'approve' : 'deny')
    try {
      if (approved) {
        await approveOAuthDevice(code)
        setStatus('approved')
      } else {
        await denyOAuthDevice(code)
        setStatus('denied')
      }
    } catch (err) {
      showError(err)
      setPending(null)
    }
  }

  if (status !== 'idle') {
    return (
      <LoginLayout branding={currentTenant?.branding}>
        <div className="flex flex-col gap-8 text-center" role="status">
          <div className="flex flex-col items-center gap-3">
            <div className="flex size-14 items-center justify-center rounded-full bg-emerald-500/10">
              <CheckCircle2 className="size-7 text-emerald-600" />
            </div>
            <h1 className="text-2xl font-semibold tracking-tight">
              {status === 'approved' ? 'Device approved' : 'Device denied'}
            </h1>
          </div>
          <Button className="w-full" onClick={() => navigate('/login-success', { replace: true })}>
            Continue
          </Button>
        </div>
      </LoginLayout>
    )
  }

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <div className="flex flex-col gap-8">
        <div className="flex flex-col items-center gap-2 text-center">
          <h1 className="text-2xl font-semibold tracking-tight">Authorize device</h1>
          <p className="text-sm text-muted-foreground">Enter the code shown on your device.</p>
        </div>

        {error && (
          <div role="alert" className="flex items-start gap-2.5 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
            <AlertCircle className="mt-0.5 size-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <div className="space-y-2">
          <Label htmlFor="oauth-device-code">User code</Label>
          <Input
            id="oauth-device-code"
            value={userCode}
            onChange={(event) => setUserCode(event.target.value.toUpperCase())}
            autoComplete="one-time-code"
            className="font-mono uppercase tracking-[0.3em] text-center"
          />
        </div>

        <div className="flex items-center gap-2">
          <Button className="flex-1" onClick={() => submit(true)} disabled={pending !== null}>
            {pending === 'approve' ? 'Approving...' : 'Approve'}
          </Button>
          <Button variant="outline" className="flex-1" onClick={() => submit(false)} disabled={pending !== null}>
            {pending === 'deny' ? 'Denying...' : 'Deny'}
          </Button>
        </div>
      </div>
    </LoginLayout>
  )
}
