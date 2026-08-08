import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { FormPasswordField } from '@/components/form'
import { FormCodeField, FormEmailField } from '@/components/inputs'
import { useToast } from '@/hooks/useToast'
import { initiateEmailChange, verifyEmailChange } from '@/services/api/account'
import { isRateLimitError, rateLimitMessage } from '@/services/api/rateLimit'
import { SecurityFormPage, SECURITY_ROUTE } from './components/SecurityFormPage'
import { isAlreadyTakenError } from './serverErrors'

interface EmailForm {
  new_email: string
  current_password: string
}

interface CodeForm {
  otp: string
}

export default function ChangeEmailPage() {
  const navigate = useNavigate()
  const { showError, showSuccess } = useToast()
  const [sent, setSent] = useState(false)
  const [pendingEmail, setPendingEmail] = useState('')
  const [throttled, setThrottled] = useState<string | null>(null)
  const form = useForm<EmailForm>({ defaultValues: { new_email: '', current_password: '' } })

  const mutation = useMutation({
    mutationFn: (data: EmailForm) => initiateEmailChange(data.new_email.trim(), data.current_password),
    onSuccess: (_data, vars) => {
      setPendingEmail(vars.new_email.trim())
      setSent(true)
    },
    onError: (err) => {
      const status = (err as { status?: number }).status
      if (isRateLimitError(err)) {
        setThrottled(rateLimitMessage(err))
        return
      }
      // Only a 403 the server tagged `step_up_required` is a step-up problem —
      // the interceptor already ran and retried the ceremony, so reaching here
      // means it was cancelled or unavailable. Any other 403 is a genuine
      // permission denial and must keep the server's own explanation.
      if (status === 403 && (err as { code?: string }).code === 'step_up_required') {
        showError(new Error('Verify your identity with a second factor, then try again.'))
        return
      }
      if (isAlreadyTakenError(err, /already in use/i)) {
        form.setError('new_email', { message: 'That email address is already in use.' })
        return
      }
      if (status === 401) {
        form.setError('current_password', { message: 'Current password is incorrect.' })
        return
      }
      showError(err, 'Could not initiate email change')
    },
  })

  const codeForm = useForm<CodeForm>({ defaultValues: { otp: '' } })

  const verifyMutation = useMutation({
    mutationFn: (data: CodeForm) => verifyEmailChange(data.otp.trim()),
    onSuccess: () => {
      showSuccess('Email address updated')
      navigate(SECURITY_ROUTE)
    },
    onError: (err) => {
      const status = (err as { status?: number }).status
      if (status === 400 || status === 401 || status === 404) {
        codeForm.setError('otp', { message: 'That code is incorrect or has expired.' })
        return
      }
      showError(err, 'Could not confirm the new email address')
    },
  })

  // Step 2. The backend emails a 6-digit OTP, so this page has to collect it —
  // without this the flow dead-ended: the code arrived with nowhere to enter it.
  if (sent) {
    return (
      <SecurityFormPage
        title="Confirm your new email"
        description={`Enter the 6-digit code we sent to ${pendingEmail}. Your current address stays active until you confirm.`}
        onSubmit={codeForm.handleSubmit((data) => { setThrottled(null); verifyMutation.mutate(data) })}
        submitLabel="Confirm email"
        pendingLabel="Confirming…"
        pending={verifyMutation.isPending}
        alert={throttled}
      >
        <FormCodeField
          label="Verification code"
          numeric
          placeholder="000000"
          maxLength={6}
          autoComplete="one-time-code"
          error={codeForm.formState.errors.otp?.message}
          {...codeForm.register('otp', {
            required: 'Enter the code from your email.',
            pattern: { value: /^\d{6}$/, message: 'The code is 6 digits.' },
          })}
        />
        <button
          type="button"
          className="text-sm text-muted-foreground underline underline-offset-4 hover:text-foreground"
          onClick={() => { setSent(false); codeForm.reset() }}
        >
          Use a different email address
        </button>
      </SecurityFormPage>
    )
  }

  return (
    <SecurityFormPage
      title="Change email address"
      description="We will email a verification code to the new address. The change takes effect once you enter it."
      onSubmit={form.handleSubmit((data) => { setThrottled(null); mutation.mutate(data) })}
      submitLabel="Send verification code"
      pendingLabel="Sending…"
      pending={mutation.isPending}
      alert={throttled}
    >
      <FormEmailField
          label="New email address"
          placeholder="you@example.com"
          autoComplete="email"
          error={form.formState.errors.new_email?.message}
          {...form.register('new_email', {
            required: 'Email is required.',
            pattern: {
              value: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
              message: 'Enter a valid email address.',
            },
        })}
      />
      {/* Required by user.ChangeEmailRequestDTO. */}
      <FormPasswordField
        label="Current password"
        autoComplete="current-password"
        error={form.formState.errors.current_password?.message}
        {...form.register('current_password', { required: 'Current password is required.' })}
      />
    </SecurityFormPage>
  )
}
