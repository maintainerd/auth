import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { yupResolver } from '@hookform/resolvers/yup'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { DetailsContainer } from '@/components/container'
import { FormPageHeader } from '@/components/header'
import { FormSwitchField, FormInputField, FormSelectField, FormSubmitButton } from '@/components/form'
import { useSessionSettings, useUpdateSessionSettings } from '@/hooks/useSessionSettings'
import { useToast } from '@/hooks/useToast'
import { ConfirmationDialog } from '@/components/dialog'
import { useUnsavedChangesGuard } from '@/hooks/useUnsavedChangesGuard'
import { sessionSettingsSchema, type SessionSettingsFormData } from '@/lib/validations'

const BACKEND_FIELD_MAP: Record<string, string> = {
  access_token_ttl_minutes: 'access_token_ttl_minutes',
  refresh_token_ttl_days: 'refresh_token_ttl_days',
  max_concurrent_sessions: 'max_concurrent_sessions',
  idle_timeout_minutes: 'idle_timeout_minutes',
  absolute_timeout_hours: 'absolute_timeout_hours',
  rotate_refresh_tokens: 'rotate_refresh_tokens',
  refresh_token_reuse_interval_seconds: 'refresh_token_reuse_interval_seconds',
  cookie_secure: 'cookie_secure',
  cookie_http_only: 'cookie_http_only',
  cookie_same_site: 'cookie_same_site',
  revoke_sessions_on_password_change: 'revoke_sessions_on_password_change',
}

const SAME_SITE_OPTIONS = [
  { value: 'Strict', label: 'Strict' },
  { value: 'Lax', label: 'Lax' },
  { value: 'None', label: 'None (requires Secure)' },
]

export default function SessionManagementPage() {
  const navigate = useNavigate()
  const { showSuccess, showError, parseError } = useToast()
  const backTo = `/security?tab=sessions`

  const { data: saved, isLoading } = useSessionSettings()
  const updateMutation = useUpdateSessionSettings()

  const { handleSubmit, reset, watch, setValue, setError, formState: { errors, isSubmitting, isDirty } } = useForm<SessionSettingsFormData>({
    resolver: yupResolver(sessionSettingsSchema),
    defaultValues: {
      access_token_ttl_minutes: 15,
      refresh_token_ttl_days: 30,
      max_concurrent_sessions: 5,
      idle_timeout_minutes: 30,
      absolute_timeout_hours: 24,
      rotate_refresh_tokens: true,
      refresh_token_reuse_interval_seconds: 10,
      cookie_secure: true,
      cookie_http_only: true,
      cookie_same_site: 'Lax',
      revoke_sessions_on_password_change: true,
    },
    mode: 'onSubmit',
  })

  const formValues = watch()

  useEffect(() => {
    if (saved) {
      reset({
        access_token_ttl_minutes: saved.access_token_ttl_minutes ?? 15,
        refresh_token_ttl_days: saved.refresh_token_ttl_days ?? 30,
        max_concurrent_sessions: saved.max_concurrent_sessions ?? 5,
        idle_timeout_minutes: saved.idle_timeout_minutes ?? 30,
        absolute_timeout_hours: saved.absolute_timeout_hours ?? 24,
        rotate_refresh_tokens: saved.rotate_refresh_tokens ?? true,
        refresh_token_reuse_interval_seconds: saved.refresh_token_reuse_interval_seconds ?? 10,
        cookie_secure: saved.cookie_secure ?? true,
        cookie_http_only: saved.cookie_http_only ?? true,
        cookie_same_site: saved.cookie_same_site ?? 'Lax',
        revoke_sessions_on_password_change: saved.revoke_sessions_on_password_change ?? true,
      })
    }
  }, [saved, reset])

  const handleUpdate = (updates: Partial<SessionSettingsFormData>) => {
    Object.entries(updates).forEach(([key, value]) => {
      setValue(key as keyof SessionSettingsFormData, value, { shouldValidate: false, shouldDirty: true })
    })
  }

  const onSubmit = async (data: SessionSettingsFormData) => {
    try {
      await updateMutation.mutateAsync(data)
      showSuccess('Session settings saved successfully')
      navigate(backTo)
    } catch (error) {
      const parsed = parseError(error)
      let mappedToField = false
      if (parsed.fieldErrors) {
        for (const [field, message] of Object.entries(parsed.fieldErrors)) {
          const formField = BACKEND_FIELD_MAP[field]
          if (formField) {
            setError(formField as never, { type: 'server', message })
            mappedToField = true
          }
        }
      }
      if (!mappedToField) {
        const lower = parsed.message.toLowerCase()
        const keywordOrder: Array<[string, string]> = [
          ['refresh_token_reuse_interval_seconds', 'refresh_token_reuse_interval_seconds'],
          ['revoke_sessions_on_password_change', 'revoke_sessions_on_password_change'],
          ['absolute_timeout_hours', 'absolute_timeout_hours'],
          ['idle_timeout_minutes', 'idle_timeout_minutes'],
          ['refresh_token_ttl_days', 'refresh_token_ttl_days'],
          ['access_token_ttl_minutes', 'access_token_ttl_minutes'],
          ['max_concurrent_sessions', 'max_concurrent_sessions'],
          ['rotate_refresh_tokens', 'rotate_refresh_tokens'],
          ['cookie_same_site', 'cookie_same_site'],
          ['cookie_http_only', 'cookie_http_only'],
          ['cookie_secure', 'cookie_secure'],
        ]
        const hit = keywordOrder.find(([keyword]) => lower.includes(keyword))
        if (hit) {
          setError(hit[1] as never, { type: 'server', message: parsed.message })
        }
      }
      showError(error)
    }
  }

  const isBusy = isSubmitting || updateMutation.isPending

  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  if (isLoading) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader backUrl={backTo} onBack={() => guard(() => navigate(backTo))} backLabel="Back to Sessions" title="Configure Sessions" description="Set token lifetimes, timeouts, and cookie settings." />
          <Card>
            <CardContent className="space-y-4 pt-6">
              <Skeleton className="h-5 w-40" />
              <div className="grid gap-4 md:grid-cols-2">
                {Array.from({ length: 4 }).map((_, i) => (
                  <Skeleton key={i} className="h-10 w-full" />
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      </DetailsContainer>
    )
  }

  return (
    <DetailsContainer>
      <div className="flex flex-col gap-6">
        <FormPageHeader
          backUrl={backTo}
          onBack={() => guard(() => navigate(backTo))}
          backLabel="Back to Sessions"
          title="Configure Sessions"
          description="Configure token lifetimes, idle/absolute timeouts, concurrency, refresh rotation, and cookie flags."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Token Lifetimes</CardTitle>
              <p className="text-sm text-muted-foreground">Set how long access and refresh tokens remain valid.</p>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField label="Access Token TTL (minutes)" description="1–60 minutes" type="number" value={formValues.access_token_ttl_minutes.toString()} onChange={(e) => handleUpdate({ access_token_ttl_minutes: parseInt(e.target.value) || 1 })} error={errors.access_token_ttl_minutes?.message} disabled={isBusy} />
                <FormInputField label="Refresh Token TTL (days)" description="1–365 days" type="number" value={formValues.refresh_token_ttl_days.toString()} onChange={(e) => handleUpdate({ refresh_token_ttl_days: parseInt(e.target.value) || 1 })} error={errors.refresh_token_ttl_days?.message} disabled={isBusy} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Timeouts & Concurrency</CardTitle>
              <p className="text-sm text-muted-foreground">Idle and absolute session limits. Set concurrency to 0 for unlimited.</p>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-3">
                <FormInputField label="Idle Timeout (minutes)" type="number" value={formValues.idle_timeout_minutes.toString()} onChange={(e) => handleUpdate({ idle_timeout_minutes: parseInt(e.target.value) || 1 })} error={errors.idle_timeout_minutes?.message} disabled={isBusy} />
                <FormInputField label="Absolute Timeout (hours)" type="number" value={formValues.absolute_timeout_hours.toString()} onChange={(e) => handleUpdate({ absolute_timeout_hours: parseInt(e.target.value) || 1 })} error={errors.absolute_timeout_hours?.message} disabled={isBusy} />
                <FormInputField label="Max Concurrent Sessions" description="0 = unlimited" type="number" value={formValues.max_concurrent_sessions.toString()} onChange={(e) => handleUpdate({ max_concurrent_sessions: parseInt(e.target.value) || 0 })} error={errors.max_concurrent_sessions?.message} disabled={isBusy} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Refresh Rotation</CardTitle>
              <p className="text-sm text-muted-foreground">Single-use refresh tokens with reuse detection and family revocation.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormSwitchField label="Rotate Refresh Tokens" description="Issue a new refresh token on every refresh; replaying a consumed token revokes the entire token family" checked={formValues.rotate_refresh_tokens} onCheckedChange={(v) => handleUpdate({ rotate_refresh_tokens: v })} disabled={isBusy} />
              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField label="Reuse Grace (seconds)" description="Replay window before family revocation (0 = instant)" type="number" value={formValues.refresh_token_reuse_interval_seconds.toString()} onChange={(e) => handleUpdate({ refresh_token_reuse_interval_seconds: parseInt(e.target.value) || 0 })} error={errors.refresh_token_reuse_interval_seconds?.message} disabled={isBusy} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Cookie Settings</CardTitle>
              <p className="text-sm text-muted-foreground">Security flags applied to authentication cookies.</p>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3 sm:grid-cols-2">
                <FormSwitchField label="Secure" description="Only transmit cookies over HTTPS" checked={formValues.cookie_secure} onCheckedChange={(v) => handleUpdate({ cookie_secure: v })} disabled={isBusy} />
                <FormSwitchField label="HttpOnly" description="Block JavaScript access to cookies" checked={formValues.cookie_http_only} onCheckedChange={(v) => handleUpdate({ cookie_http_only: v })} disabled={isBusy} />
                <FormSwitchField label="Revoke on Password Change" description="Terminate all other sessions when password is changed" checked={formValues.revoke_sessions_on_password_change} onCheckedChange={(v) => handleUpdate({ revoke_sessions_on_password_change: v })} disabled={isBusy} />
                <FormSelectField label="SameSite" options={SAME_SITE_OPTIONS} value={formValues.cookie_same_site} onValueChange={(v) => handleUpdate({ cookie_same_site: v as SessionSettingsFormData['cookie_same_site'] })} error={errors.cookie_same_site?.message} disabled={isBusy} />
              </div>
            </CardContent>
          </Card>

          <div className="flex justify-end gap-3">
            <Button type="button" variant="outline" onClick={() => guard(() => navigate(backTo))} disabled={isBusy}>
              Cancel
            </Button>
            <FormSubmitButton isSubmitting={isBusy} submitText="Save Changes" />
          </div>
        </form>

        <ConfirmationDialog
          open={isPromptOpen}
          onOpenChange={(open) => { if (!open) cancelLeave() }}
          onConfirm={confirmLeave}
          title="Discard changes?"
          description="You have unsaved changes. If you leave now, they will be lost."
          confirmText="Discard changes"
          cancelText="Keep editing"
          variant="destructive"
        />
      </div>
    </DetailsContainer>
  )
}
