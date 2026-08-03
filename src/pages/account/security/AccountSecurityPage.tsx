import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useMutation, useQuery } from '@tanstack/react-query'
import { AtSign, KeyRound, Mail } from 'lucide-react'
import AccountLayout from '@/components/layout/AccountLayout'
import { SettingsCard } from '@/components/card'
import { Button } from '@/components/ui/button'
import { FormInputField, FormPasswordField } from '@/components/form'
import { FormEmailField, FormPasswordFieldWithPolicy } from '@/components/inputs'
import { useToast } from '@/hooks/useToast'
import { useTenant } from '@/hooks/useTenant'
import { changePassword, changeUsername, fetchAccountInfo, initiateEmailChange } from '@/services/api/account'
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

  const passwordConfig = getCurrentTenant()?.password_config

  const { data: account } = useQuery({
    queryKey: ['account', 'info'],
    queryFn: fetchAccountInfo,
  })

  const usernameForm = useForm<UsernameForm>()
  const emailForm = useForm<EmailForm>()
  const passwordForm = useForm<PasswordForm>()

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
        return
      }
      showError(err, 'Could not initiate email change')
    },
  })

  const passwordResetMutation = useMutation({
    mutationFn: () => forgotPassword({ email: account?.email ?? '' }),
    onSuccess: () => setPasswordResetSent(true),
    onError: (err) => showError(err, 'Could not send password reset'),
  })

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
      <div className="grid gap-6">
        <SettingsCard title="Username" description="Update the username attached to your account." icon={AtSign}>
          <form
            onSubmit={usernameForm.handleSubmit((data) => usernameMutation.mutate(data))}
            className="flex flex-col gap-3 sm:flex-row sm:items-end"
          >
            <FormInputField
              label="New username"
              placeholder="New username"
              containerClassName="flex-1"
              {...usernameForm.register('username', { required: true })}
            />
            <Button type="submit" disabled={usernameMutation.isPending}>
              {usernameMutation.isPending ? 'Saving…' : 'Save'}
            </Button>
          </form>
        </SettingsCard>

        <SettingsCard title="Email address" description="Change the email you use for sign-in." icon={Mail}>
          {emailSent ? (
            <p className="text-sm text-muted-foreground">
              Check your new inbox for a verification link to confirm the change.
            </p>
          ) : (
            <form
              onSubmit={emailForm.handleSubmit((data) => emailMutation.mutate(data))}
              className="flex flex-col gap-3 sm:flex-row sm:items-end"
            >
              <FormEmailField
                label="New email address"
                placeholder="New email address"
                containerClassName="flex-1"
                {...emailForm.register('new_email', { required: true })}
              />
              <Button type="submit" disabled={emailMutation.isPending}>
                {emailMutation.isPending ? 'Sending…' : 'Send link'}
              </Button>
            </form>
          )}
        </SettingsCard>

        <SettingsCard title="Password" description="Change your password without leaving your account." icon={KeyRound}>
          <div className="space-y-4">
            <form onSubmit={passwordForm.handleSubmit(onSubmitPassword)} className="space-y-4">
              <FormPasswordField
                label="Current password"
                autoComplete="current-password"
                error={passwordForm.formState.errors.current_password?.message}
                {...passwordForm.register('current_password', { required: true })}
              />
              <FormPasswordFieldWithPolicy
                label="New password"
                autoComplete="new-password"
                error={passwordForm.formState.errors.new_password?.message}
                passwordConfig={passwordConfig}
                {...passwordForm.register('new_password', { required: true })}
              />
              <FormPasswordField
                label="Confirm new password"
                autoComplete="new-password"
                error={passwordForm.formState.errors.confirm_password?.message}
                {...passwordForm.register('confirm_password', { required: true })}
              />
              <Button type="submit" disabled={passwordChangeMutation.isPending}>
                {passwordChangeMutation.isPending ? 'Changing…' : 'Change password'}
              </Button>
            </form>

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
                  {passwordResetMutation.isPending ? 'Sending…' : 'Send password reset link'}
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
          </div>
        </SettingsCard>
      </div>
    </AccountLayout>
  )
}
