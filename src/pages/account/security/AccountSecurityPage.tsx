import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useQuery, useMutation } from '@tanstack/react-query'
import { Mail, AtSign, KeyRound } from 'lucide-react'
import AccountLayout from '@/components/layout/AccountLayout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { PasswordRequirements } from '@/components/form'
import { useToast } from '@/hooks/useToast'
import { useTenant } from '@/hooks/useTenant'
import { changeUsername, initiateEmailChange, fetchAccountInfo, changePassword } from '@/services/api/account'
import { forgotPassword } from '@/services/api/auth'

interface UsernameForm {
  username: string
}

interface EmailForm {
  new_email: string
}

interface PasswordForm {
  current_password: string
  new_password: string
  confirm_password: string
}

export default function AccountSecurityPage() {
  const { showError, showSuccess } = useToast()
  const { getCurrentTenant } = useTenant()
  const [emailSent, setEmailSent] = useState(false)
  const [passwordResetSent, setPasswordResetSent] = useState(false)
  const [useResetLink, setUseResetLink] = useState(false)

  // Tenant-scoped password policy drives the live requirements checklist, the
  // same source the register form uses (multitenancy: the current tenant is
  // resolved from the request host at bootstrap).
  const passwordConfig = getCurrentTenant()?.password_config

  const { data: account } = useQuery({
    queryKey: ['account', 'info'],
    queryFn: fetchAccountInfo,
  })

  const usernameForm = useForm<UsernameForm>()
  const emailForm = useForm<EmailForm>()
  const passwordForm = useForm<PasswordForm>()
  const newPasswordValue = passwordForm.watch('new_password') ?? ''

  const usernameMutation = useMutation({
    mutationFn: (data: UsernameForm) => changeUsername(data.username),
    onSuccess: () => usernameForm.reset(),
    onError: (err) => showError(err, 'Could not update username'),
  })

  const emailMutation = useMutation({
    mutationFn: (data: EmailForm) => initiateEmailChange(data.new_email),
    onSuccess: () => {
      setEmailSent(true)
      emailForm.reset()
    },
    onError: (err) => {
      const status = (err as { status?: number }).status
      if (status === 403) {
        showError(new Error('Step-up authentication required. Please verify your identity first.'))
      } else {
        showError(err, 'Could not initiate email change')
      }
    },
  })

  const passwordResetMutation = useMutation({
    mutationFn: () => forgotPassword({ email: account?.email ?? '' }),
    onSuccess: () => setPasswordResetSent(true),
    onError: (err) => showError(err, 'Could not send password reset'),
  })

  // Authenticated self-service change. Distinct from the reset-link flow: a
  // signed-in user proves knowledge of their current password rather than going
  // through their inbox. The backend enforces the tenant policy + history and
  // revokes the user's OTHER sessions on success.
  const passwordChangeMutation = useMutation({
    mutationFn: (data: PasswordForm) => changePassword(data.current_password, data.new_password),
    onSuccess: (result) => {
      passwordForm.reset()
      showSuccess(
        result.reauthentication_required
          ? 'Password changed. Please sign in again.'
          : result.other_sessions_revoked
            ? 'Password changed. Your other sessions were signed out.'
            : 'Password changed successfully.',
      )
    },
    onError: (err) => {
      const status = (err as { status?: number }).status
      if (status === 401) {
        passwordForm.setError('current_password', { message: 'Current password is incorrect.' })
        return
      }
      // The tenant policy (length, history, breach, etc.) rejected the new one;
      // surface the backend message against the new-password field.
      const message = err instanceof Error ? err.message : 'Could not change password'
      passwordForm.setError('new_password', { message })
    },
  })

  const onSubmitPassword = (data: PasswordForm) => {
    if (data.new_password !== data.confirm_password) {
      passwordForm.setError('confirm_password', { message: 'Passwords do not match.' })
      return
    }
    passwordChangeMutation.mutate(data)
  }

  return (
    <AccountLayout title="Security">
      <div className="space-y-4">
        {/* Username */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <AtSign className="size-4" />
              Username
            </CardTitle>
          </CardHeader>
          <CardContent>
            <form
              onSubmit={usernameForm.handleSubmit((data) => usernameMutation.mutate(data))}
              className="flex gap-2"
            >
              <div className="flex-1">
                <Label htmlFor="username" className="sr-only">
                  New username
                </Label>
                <Input
                  id="username"
                  placeholder="New username"
                  {...usernameForm.register('username', { required: true })}
                />
              </div>
              <Button type="submit" disabled={usernameMutation.isPending}>
                Save
              </Button>
            </form>
          </CardContent>
        </Card>

        {/* Email */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <Mail className="size-4" />
              Email address
            </CardTitle>
          </CardHeader>
          <CardContent>
            {emailSent ? (
              <p className="text-sm text-muted-foreground">
                Check your new inbox for a verification link to confirm the change.
              </p>
            ) : (
              <form
                onSubmit={emailForm.handleSubmit((data) => emailMutation.mutate(data))}
                className="flex gap-2"
              >
                <div className="flex-1">
                  <Label htmlFor="new_email" className="sr-only">
                    New email address
                  </Label>
                  <Input
                    id="new_email"
                    type="email"
                    placeholder="New email address"
                    {...emailForm.register('new_email', { required: true })}
                  />
                </div>
                <Button type="submit" disabled={emailMutation.isPending}>
                  Send link
                </Button>
              </form>
            )}
          </CardContent>
        </Card>

        {/* Password */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <KeyRound className="size-4" />
              Password
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Primary path: authenticated change — the user knows their current
                password, so no inbox round-trip is needed. */}
            <form onSubmit={passwordForm.handleSubmit(onSubmitPassword)} className="space-y-3">
              <div className="space-y-1">
                <Label htmlFor="current_password">Current password</Label>
                <Input
                  id="current_password"
                  type="password"
                  autoComplete="current-password"
                  {...passwordForm.register('current_password', { required: true })}
                />
                {passwordForm.formState.errors.current_password?.message && (
                  <p className="text-xs text-destructive">
                    {passwordForm.formState.errors.current_password.message}
                  </p>
                )}
              </div>

              <div className="space-y-1">
                <Label htmlFor="new_password">New password</Label>
                <Input
                  id="new_password"
                  type="password"
                  autoComplete="new-password"
                  {...passwordForm.register('new_password', { required: true })}
                />
                {passwordForm.formState.errors.new_password?.message && (
                  <p className="text-xs text-destructive">
                    {passwordForm.formState.errors.new_password.message}
                  </p>
                )}
                {newPasswordValue && (
                  <PasswordRequirements password={newPasswordValue} config={passwordConfig} />
                )}
              </div>

              <div className="space-y-1">
                <Label htmlFor="confirm_password">Confirm new password</Label>
                <Input
                  id="confirm_password"
                  type="password"
                  autoComplete="new-password"
                  {...passwordForm.register('confirm_password', { required: true })}
                />
                {passwordForm.formState.errors.confirm_password?.message && (
                  <p className="text-xs text-destructive">
                    {passwordForm.formState.errors.confirm_password.message}
                  </p>
                )}
              </div>

              <Button type="submit" disabled={passwordChangeMutation.isPending}>
                Change password
              </Button>
            </form>

            {/* Fallback: a user who has forgotten their current password can still
                reset it via the emailed link. */}
            <div className="border-t pt-3 text-sm text-muted-foreground">
              {passwordResetSent ? (
                <p className="text-green-600">Check your email for a password reset link.</p>
              ) : useResetLink ? (
                <Button
                  variant="outline"
                  size="sm"
                  disabled={passwordResetMutation.isPending || !account?.email}
                  onClick={() => passwordResetMutation.mutate()}
                >
                  Send password reset link
                </Button>
              ) : (
                <button
                  type="button"
                  className="underline hover:text-foreground"
                  onClick={() => setUseResetLink(true)}
                >
                  Forgot your current password?
                </button>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </AccountLayout>
  )
}
