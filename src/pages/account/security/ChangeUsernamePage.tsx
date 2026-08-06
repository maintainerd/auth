import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { FormInputField, FormPasswordField } from '@/components/form'
import { useToast } from '@/hooks/useToast'
import { changeUsername } from '@/services/api/account'
import { isRateLimitError, rateLimitMessage } from '@/services/api/rateLimit'
import { SecurityFormPage, SECURITY_ROUTE } from './components/SecurityFormPage'
import { isAlreadyTakenError } from './serverErrors'

interface UsernameForm {
  username: string
  current_password: string
}

export default function ChangeUsernamePage() {
  const navigate = useNavigate()
  const { showError, showSuccess } = useToast()
  const [throttled, setThrottled] = useState<string | null>(null)
  const form = useForm<UsernameForm>({ defaultValues: { username: '', current_password: '' } })

  const mutation = useMutation({
    mutationFn: (data: UsernameForm) => changeUsername(data.username.trim(), data.current_password),
    onSuccess: () => {
      showSuccess('Username updated')
      navigate(SECURITY_ROUTE)
    },
    onError: (err) => {
      // The per-account password throttle is shared with the email, password and
      // delete-account forms, so it is not about anything on this page — it
      // belongs above the form, not on an input.
      if (isRateLimitError(err)) {
        setThrottled(rateLimitMessage(err))
        return
      }
      // Surface a conflict on the field itself rather than only in a toast —
      // "already taken" is about this input and the user needs to edit it.
      const status = (err as { status?: number }).status
      if (isAlreadyTakenError(err, /already taken/i)) {
        form.setError('username', { message: 'That username is already taken.' })
        return
      }
      if (status === 401) {
        form.setError('current_password', { message: 'Current password is incorrect.' })
        return
      }
      showError(err, 'Could not update username')
    },
  })

  return (
    <SecurityFormPage
      title="Change username"
      description="Update the username attached to your account. You will use it to sign in."
      onSubmit={form.handleSubmit((data) => { setThrottled(null); mutation.mutate(data) })}
      submitLabel="Save"
      pendingLabel="Saving…"
      pending={mutation.isPending}
      alert={throttled}
    >
      <FormInputField
        label="New username"
        placeholder="New username"
        autoComplete="username"
        error={form.formState.errors.username?.message}
        {...form.register('username', {
          required: 'Username is required.',
          // Mirrors the backend rule so the user is told before a round trip.
          minLength: { value: 3, message: 'Username must be at least 3 characters.' },
          maxLength: { value: 50, message: 'Username must be at most 50 characters.' },
        })}
      />
      {/* Required by user.ChangeUsernameDTO — the service re-checks it against
          the stored hash before allowing an identifier change. */}
      <FormPasswordField
        label="Current password"
        autoComplete="current-password"
        error={form.formState.errors.current_password?.message}
        {...form.register('current_password', { required: 'Current password is required.' })}
      />
    </SecurityFormPage>
  )
}
