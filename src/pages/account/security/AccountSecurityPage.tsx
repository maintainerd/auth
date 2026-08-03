import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useMutation, useQuery } from '@tanstack/react-query'
import { AtSign, KeyRound, Mail } from 'lucide-react'
import AccountLayout from '@/components/layout/AccountLayout'
import { SettingsCard } from '@/components/card'
import { Button } from '@/components/ui/button'
import { FormInputField, FormPasswordField } from '@/components/form'
import { FormEmailField, FormPasswordFieldWithPolicy } from '@/components/inputs'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { SettingsActionRow } from '@/components/settings'
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
  const [usernameOpen, setUsernameOpen] = useState(false)
  const [emailOpen, setEmailOpen] = useState(false)
  const [passwordOpen, setPasswordOpen] = useState(false)
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
    onSuccess: () => {
      usernameForm.reset()
      setUsernameOpen(false)
      showSuccess('Username updated')
    },
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
      setPasswordOpen(false)
      setUseResetLink(false)
      setPasswordResetSent(false)
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

  const handleUsernameOpenChange = (open: boolean) => {
    setUsernameOpen(open)
    if (!open) usernameForm.reset()
  }

  const handleEmailOpenChange = (open: boolean) => {
    setEmailOpen(open)
    if (!open) {
      emailForm.reset()
      setEmailSent(false)
    }
  }

  const handlePasswordOpenChange = (open: boolean) => {
    setPasswordOpen(open)
    if (!open) {
      passwordForm.reset()
      setUseResetLink(false)
      setPasswordResetSent(false)
    }
  }

  return (
    <AccountLayout title="Security">
      <SettingsCard
        title="Sign-in details"
        description="Manage the identifiers and password used to access your account."
        icon={AtSign}
        contentClassName="p-0"
      >
        <div className="divide-y">
          <SettingsActionRow
            icon={Mail}
            title="Email address"
            description="Change the email address you use for sign-in."
            actionLabel="Change"
            onAction={() => setEmailOpen(true)}
          />
          <SettingsActionRow
            icon={AtSign}
            title="Username"
            description="Update the username attached to your account."
            actionLabel="Change"
            onAction={() => setUsernameOpen(true)}
          />
          <SettingsActionRow
            icon={KeyRound}
            title="Password"
            description="Change your password without leaving your account."
            actionLabel="Change"
            onAction={() => setPasswordOpen(true)}
          />
        </div>
      </SettingsCard>

      <Dialog open={usernameOpen} onOpenChange={handleUsernameOpenChange}>
        <DialogContent className="sm:max-w-md">
          <form onSubmit={usernameForm.handleSubmit((data) => usernameMutation.mutate(data))} className="space-y-4">
            <DialogHeader>
              <DialogTitle>Change username</DialogTitle>
              <DialogDescription>Enter the new username you want attached to your account.</DialogDescription>
            </DialogHeader>
            <FormInputField
              label="New username"
              placeholder="New username"
              error={usernameForm.formState.errors.username?.message}
              {...usernameForm.register('username', { required: 'Username is required.' })}
            />
            <DialogFooter>
              <Button type="button" variant="ghost" onClick={() => handleUsernameOpenChange(false)}>Cancel</Button>
              <Button type="submit" disabled={usernameMutation.isPending}>
                {usernameMutation.isPending ? 'Saving…' : 'Save'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={emailOpen} onOpenChange={handleEmailOpenChange}>
        <DialogContent className="sm:max-w-md">
          <form onSubmit={emailForm.handleSubmit((data) => emailMutation.mutate(data))} className="space-y-4">
            <DialogHeader>
              <DialogTitle>Change email address</DialogTitle>
              <DialogDescription>We will send a verification link before the new email becomes active.</DialogDescription>
            </DialogHeader>
            {emailSent ? (
              <p className="rounded-md border border-emerald-500/30 bg-emerald-500/10 p-3 text-sm text-emerald-700 dark:text-emerald-400">
                Check your new inbox for a verification link to confirm the change.
              </p>
            ) : (
              <FormEmailField
                label="New email address"
                placeholder="New email address"
                error={emailForm.formState.errors.new_email?.message}
                {...emailForm.register('new_email', { required: 'Email is required.' })}
              />
            )}
            <DialogFooter>
              <Button type="button" variant="ghost" onClick={() => handleEmailOpenChange(false)}>
                {emailSent ? 'Close' : 'Cancel'}
              </Button>
              {!emailSent && (
                <Button type="submit" disabled={emailMutation.isPending}>
                  {emailMutation.isPending ? 'Sending…' : 'Send link'}
                </Button>
              )}
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={passwordOpen} onOpenChange={handlePasswordOpenChange}>
        <DialogContent className="sm:max-w-md">
          <form onSubmit={passwordForm.handleSubmit(onSubmitPassword)} className="space-y-4">
            <DialogHeader>
              <DialogTitle>Change password</DialogTitle>
              <DialogDescription>Enter your current password, then choose a new one.</DialogDescription>
            </DialogHeader>
            <FormPasswordField
              label="Current password"
              autoComplete="current-password"
              error={passwordForm.formState.errors.current_password?.message}
              {...passwordForm.register('current_password', { required: 'Current password is required.' })}
            />
            <FormPasswordFieldWithPolicy
              label="New password"
              autoComplete="new-password"
              error={passwordForm.formState.errors.new_password?.message}
              passwordConfig={passwordConfig}
              {...passwordForm.register('new_password', { required: 'New password is required.' })}
            />
            <FormPasswordField
              label="Confirm new password"
              autoComplete="new-password"
              error={passwordForm.formState.errors.confirm_password?.message}
              {...passwordForm.register('confirm_password', { required: 'Please confirm your new password.' })}
            />
            <div className="border-t pt-3 text-sm text-muted-foreground">
              {passwordResetSent ? (
                <p className="text-green-600">Check your email for a password reset link.</p>
              ) : useResetLink ? (
                <Button
                  type="button"
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
            <DialogFooter>
              <Button type="button" variant="ghost" onClick={() => handlePasswordOpenChange(false)}>Cancel</Button>
              <Button type="submit" disabled={passwordChangeMutation.isPending}>
                {passwordChangeMutation.isPending ? 'Changing…' : 'Change password'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </AccountLayout>
  )
}
