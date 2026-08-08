import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { FormPasswordField } from '@/components/form'
import { FormPasswordFieldWithPolicy } from '@/components/inputs'
import { useToast } from '@/hooks/useToast'
import { useTenant } from '@/hooks/useTenant'
import { changePassword, fetchAccountInfo } from '@/services/api/account'
import { forgotPassword } from '@/services/api/auth'
import { isRateLimitError, rateLimitMessage } from '@/services/api/rateLimit'
import { SecurityFormPage, SECURITY_ROUTE } from './components/SecurityFormPage'

interface PasswordForm {
  current_password: string
  new_password: string
  confirm_password: string
}

export default function ChangePasswordPage() {
  const navigate = useNavigate()
  const { showError, showSuccess } = useToast()
  const { getCurrentTenant } = useTenant()
  const [useResetLink, setUseResetLink] = useState(false)
  const [resetSent, setResetSent] = useState(false)
  const [throttled, setThrottled] = useState<string | null>(null)

  // Drives the live requirement checklist so the rules shown match the tenant's
  // actual policy rather than a hardcoded guess.
  const passwordConfig = getCurrentTenant()?.password_config

  // The reset-link fallback emails the account's own address, so it depends on
  // this query. When the lookup fails the button used to sit there silently
  // disabled — the error is surfaced with a retry instead.
  const {
    data: account,
    error: accountError,
    isFetching: accountFetching,
    refetch: refetchAccount,
  } = useQuery({
    queryKey: ['account', 'info'],
    queryFn: fetchAccountInfo,
  })

  const form = useForm<PasswordForm>({
    defaultValues: { current_password: '', new_password: '', confirm_password: '' },
  })

  const resetMutation = useMutation({
    mutationFn: () => forgotPassword({ email: account?.email ?? '' }),
    onSuccess: () => setResetSent(true),
    onError: (err) => showError(err, 'Could not send password reset'),
  })

  const changeMutation = useMutation({
    mutationFn: (data: PasswordForm) => changePassword(data.current_password, data.new_password),
    onSuccess: (result) => {
      showSuccess(
        result.reauthentication_required
          ? 'Password changed. Please sign in again.'
          : result.other_sessions_revoked
            ? 'Password changed. Your other sessions were signed out.'
            : 'Password changed successfully.',
      )
      navigate(SECURITY_ROUTE)
    },
    onError: (err) => {
      const status = (err as { status?: number }).status
      // The throttle is not a problem with either password, so it goes above the
      // form. Left on the new-password field it read as "your new password was
      // rejected", which would send the user off inventing a different one.
      if (isRateLimitError(err)) {
        setThrottled(rateLimitMessage(err))
        return
      }
      if (status === 401) {
        form.setError('current_password', { message: 'Current password is incorrect.' })
        return
      }
      // Policy rejections (too short, reused, breached) are about the NEW
      // password, so they belong on that field rather than in a toast.
      const message = err instanceof Error ? err.message : 'Could not change password'
      form.setError('new_password', { message })
    },
  })

  const onSubmit = form.handleSubmit((data) => {
    setThrottled(null)
    if (data.new_password !== data.confirm_password) {
      form.setError('confirm_password', { message: 'Passwords do not match.' })
      return
    }
    if (data.new_password === data.current_password) {
      form.setError('new_password', { message: 'Choose a password different from your current one.' })
      return
    }
    changeMutation.mutate(data)
  })

  return (
    <SecurityFormPage
      title="Change password"
      description="Enter your current password, then choose a new one. Your other sessions may be signed out."
      onSubmit={onSubmit}
      submitLabel="Change password"
      pendingLabel="Changing…"
      pending={changeMutation.isPending}
      alert={throttled}
      footer={
        <div className="border-t pt-4 text-sm text-muted-foreground">
          {resetSent ? (
            <p role="status" className="text-emerald-600 dark:text-emerald-400">
              Check your email for a password reset link.
            </p>
          ) : useResetLink && accountError ? (
            <div role="alert" className="space-y-2 text-destructive">
              <p>We couldn't load your email address, so we can't send a reset link right now.</p>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="w-full sm:w-auto"
                disabled={accountFetching}
                onClick={() => void refetchAccount()}
              >
                {accountFetching ? 'Retrying…' : 'Try again'}
              </Button>
            </div>
          ) : useResetLink ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="w-full sm:w-auto"
              disabled={resetMutation.isPending || accountFetching || !account?.email}
              onClick={() => resetMutation.mutate()}
            >
              {resetMutation.isPending ? 'Sending…' : accountFetching ? 'Loading…' : 'Send password reset link'}
            </Button>
          ) : (
            <button
              type="button"
              className="underline underline-offset-4 hover:text-foreground"
              onClick={() => setUseResetLink(true)}
            >
              Forgot your current password?
            </button>
          )}
        </div>
      }
    >
      <FormPasswordField
        label="Current password"
        autoComplete="current-password"
        error={form.formState.errors.current_password?.message}
        {...form.register('current_password', { required: 'Current password is required.' })}
      />
      <FormPasswordFieldWithPolicy
        label="New password"
        autoComplete="new-password"
        error={form.formState.errors.new_password?.message}
        passwordConfig={passwordConfig}
        {...form.register('new_password', { required: 'New password is required.' })}
      />
      <FormPasswordField
        label="Confirm new password"
        autoComplete="new-password"
        error={form.formState.errors.confirm_password?.message}
        {...form.register('confirm_password', { required: 'Please confirm your new password.' })}
      />
    </SecurityFormPage>
  )
}
