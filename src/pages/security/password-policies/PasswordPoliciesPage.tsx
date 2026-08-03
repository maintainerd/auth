import { useEffect } from 'react'
import { AlertTriangle, RotateCcw } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { PasswordRequirements } from '@/components/inputs'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { yupResolver } from '@hookform/resolvers/yup'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { DetailsContainer } from '@/components/container'
import { FormPageHeader } from '@/components/header'
import { FormInputField, FormSelectField, FormSubmitButton } from '@/components/form'
import { FormSwitchSubContainer } from '@/components/inputs'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { ConfirmationDialog } from '@/components/dialog'
import { usePasswordPolicies, useUpdatePasswordPolicies } from '@/hooks/usePasswordPolicies'
import { useToast } from '@/hooks/useToast'
import { useUnsavedChangesGuard } from '@/hooks/useUnsavedChangesGuard'
import {
  PASSWORD_POLICY_DEFAULTS,
  PASSWORD_POLICY_LIMITS,
  passwordPoliciesSchema,
  type PasswordPoliciesFormData,
} from '@/lib/validations'

const HASH_OPTIONS = [
  { value: 'argon2id', label: 'Argon2id' },
  { value: 'bcrypt', label: 'Bcrypt' },
  { value: 'scrypt', label: 'Scrypt' },
  { value: 'pbkdf2', label: 'PBKDF2' },
]

/**
 * Maps a backend validation message onto the field it belongs to. The API returns
 * per-field messages (e.g. "min_length must be less than or equal to max_length",
 * "password_history_count must be at most 24"); without this they all surfaced as an
 * anonymous toast beside a form that looked valid.
 */
const BACKEND_FIELD_MAP: Record<string, keyof PasswordPoliciesFormData> = {
  min_length: 'min_length',
  max_length: 'max_length',
  require_uppercase: 'require_uppercase',
  require_lowercase: 'require_lowercase',
  require_number: 'require_number',
  require_symbol: 'require_symbol',
  reject_common_passwords: 'reject_common_passwords',
  check_hibp: 'check_hibp',
  min_strength_score: 'min_strength_score',
  password_history_count: 'password_history_count',
  max_age_days: 'max_age_days',
  temporary_password_validity_hours: 'temporary_password_validity_hours',
  hash_algorithm: 'hash_algorithm',
}

export default function PasswordPoliciesFormPage() {
  const navigate = useNavigate()
  const { showSuccess, showError, parseError } = useToast()
  const backTo = `/security?tab=password`

  const { data: savedPolicies, isLoading, isError } = usePasswordPolicies()
  const updateMutation = useUpdatePasswordPolicies()

  const {
    handleSubmit,
    reset,
    setError,
    watch,
    setValue,
    trigger,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<PasswordPoliciesFormData>({
    resolver: yupResolver(passwordPoliciesSchema),
    defaultValues: { ...PASSWORD_POLICY_DEFAULTS },
    // Validate on submit, not on every keystroke. With 'onChange' a number field
    // reports "Must be at least 8" the instant you clear it to type a new value,
    // so ordinary editing looks like a stream of errors.
    mode: 'onSubmit',
    reValidateMode: 'onChange',
  })

  const formValues = watch()

  useEffect(() => {
    if (savedPolicies) {
      reset({
        min_length: savedPolicies.min_length ?? PASSWORD_POLICY_DEFAULTS.min_length,
        max_length: savedPolicies.max_length ?? PASSWORD_POLICY_DEFAULTS.max_length,
        require_uppercase: savedPolicies.require_uppercase ?? PASSWORD_POLICY_DEFAULTS.require_uppercase,
        require_lowercase: savedPolicies.require_lowercase ?? PASSWORD_POLICY_DEFAULTS.require_lowercase,
        require_number: savedPolicies.require_number ?? PASSWORD_POLICY_DEFAULTS.require_number,
        require_symbol: savedPolicies.require_symbol ?? PASSWORD_POLICY_DEFAULTS.require_symbol,
        reject_common_passwords:
          savedPolicies.reject_common_passwords ?? PASSWORD_POLICY_DEFAULTS.reject_common_passwords,
        check_hibp: savedPolicies.check_hibp ?? PASSWORD_POLICY_DEFAULTS.check_hibp,
        password_history_count:
          savedPolicies.password_history_count ?? PASSWORD_POLICY_DEFAULTS.password_history_count,
        max_age_days: savedPolicies.max_age_days ?? PASSWORD_POLICY_DEFAULTS.max_age_days,
        temporary_password_validity_hours:
          savedPolicies.temporary_password_validity_hours ??
          PASSWORD_POLICY_DEFAULTS.temporary_password_validity_hours,
        hash_algorithm: savedPolicies.hash_algorithm ?? PASSWORD_POLICY_DEFAULTS.hash_algorithm,
        min_strength_score: savedPolicies.min_strength_score ?? PASSWORD_POLICY_DEFAULTS.min_strength_score,
      })
    }
  }, [savedPolicies, reset])

  // The saved algorithm, not the watched one — the warning is about changing
  // away from what is currently in force.
  const savedHashAlgorithm = savedPolicies?.hash_algorithm ?? PASSWORD_POLICY_DEFAULTS.hash_algorithm
  const hashAlgorithmChanged = formValues.hash_algorithm !== savedHashAlgorithm

  const restoreDefaults = () => {
    handleUpdate({ ...PASSWORD_POLICY_DEFAULTS })
  }

  const handleUpdate = (updates: Partial<PasswordPoliciesFormData>) => {
    Object.entries(updates).forEach(([key, value]) => {
      setValue(key as keyof PasswordPoliciesFormData, value, {
        shouldValidate: true,
        shouldDirty: true,
      })
    })
    void trigger()
  }

  const isBusy = isSubmitting || updateMutation.isPending

  // Driven by the same watched values as the rest of the form, so the preview
  // updates as the operator toggles.
  const previewConfig = {
    min_length: formValues.min_length,
    max_length: formValues.max_length,
    require_uppercase: formValues.require_uppercase,
    require_lowercase: formValues.require_lowercase,
    require_number: formValues.require_number,
    require_symbol: formValues.require_symbol,
    // buildPasswordRules renders a caveat line for these three rather than a
    // tickable rule, since they cannot be evaluated client-side.
    min_strength_score: formValues.min_strength_score,
    reject_common_passwords: formValues.reject_common_passwords,
    check_hibp: formValues.check_hibp,
  }

  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const onSubmit = async (data: PasswordPoliciesFormData) => {
    try {
      await updateMutation.mutateAsync(data)
      showSuccess('Password policy saved successfully')
      navigate(backTo)
    } catch (error) {
      // Pin the message to the offending field where the backend named one.
      const parsed = parseError(error)
      const fieldErrors = parsed.fieldErrors ?? {}
      let mapped = false
      for (const [backendField, formField] of Object.entries(BACKEND_FIELD_MAP)) {
        const message = fieldErrors[backendField]
        if (message) {
          setError(formField, { message })
          mapped = true
        }
      }
      if (!mapped && parsed.message) {
        // Fall back to a keyword match: the API also returns bare validation strings
        // that name the field inline rather than as a structured key.
        for (const [backendField, formField] of Object.entries(BACKEND_FIELD_MAP)) {
          if (parsed.message.includes(backendField)) {
            setError(formField, { message: parsed.message })
            break
          }
        }
      }
      showError(error)
    }
  }

  // Without this the form rendered fully interactive over its DEFAULT values while
  // the GET was still in flight, with Save enabled. An operator on a slow load who
  // saved before hydration overwrote the tenant's real policy with the defaults —
  // the PUT sends all 13 fields, so nothing was spared.
  if (isLoading) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader backUrl={backTo} backLabel="Back to Password Policy" onBack={() => navigate(backTo)} title="Configure Password Policy" description="Set password requirements for your tenant." />
          <Card>
            <CardContent className="space-y-4 pt-6">
              <Skeleton className="h-5 w-40" />
              <div className="grid gap-4 md:grid-cols-2">
                {Array.from({ length: 6 }).map((_, i) => (
                  <Skeleton key={i} className="h-10 w-full" />
                ))}
              </div>
              <Skeleton className="h-24 w-full" />
            </CardContent>
          </Card>
        </div>
      </DetailsContainer>
    )
  }

  if (isError) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader backUrl={backTo} backLabel="Back to Password Policy" onBack={() => guard(() => navigate(backTo))} title="Configure Password Policy" description="Set password requirements for your tenant." />
          <Card>
            <CardContent className="py-12 text-center text-sm text-destructive">
              Failed to load password policy.
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
          backLabel="Back to Password Policy"
          onBack={() => guard(() => navigate(backTo))}
          title="Configure Password Policy"
          description="Configure password length, complexity, breach screening, history, expiry, and hashing algorithm."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Complexity Requirements</CardTitle>
              <p className="text-sm text-muted-foreground">Set the minimum character types each password must contain.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField
                  label="Minimum Length"
                  description={`${PASSWORD_POLICY_DEFAULTS.min_length} or more is recommended. Length matters far more than character rules.`}
                  type="number"
                  min={PASSWORD_POLICY_LIMITS.minLength}
                  max={PASSWORD_POLICY_LIMITS.maxLength}
                  value={formValues.min_length.toString()}
                  onChange={(e) =>
                    handleUpdate({ min_length: parseInt(e.target.value) || PASSWORD_POLICY_LIMITS.minLength })
                  }
                  error={errors.min_length?.message}
                  disabled={isBusy}
                  required
                />
                <FormInputField
                  label="Maximum Length"
                  type="number"
                  min={64}
                  max={PASSWORD_POLICY_LIMITS.maxLength}
                  value={formValues.max_length.toString()}
                  onChange={(e) => handleUpdate({ max_length: parseInt(e.target.value) || 64 })}
                  error={errors.max_length?.message}
                  disabled={isBusy}
                  required
                />
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <FormSwitchSubContainer label="Require Uppercase" description="At least one A–Z character" checked={formValues.require_uppercase} onCheckedChange={(v) => handleUpdate({ require_uppercase: v })} disabled={isBusy} />
                <FormSwitchSubContainer label="Require Lowercase" description="At least one a–z character" checked={formValues.require_lowercase} onCheckedChange={(v) => handleUpdate({ require_lowercase: v })} disabled={isBusy} />
                <FormSwitchSubContainer label="Require Number" description="At least one 0–9 digit" checked={formValues.require_number} onCheckedChange={(v) => handleUpdate({ require_number: v })} disabled={isBusy} />
                <FormSwitchSubContainer label="Require Symbol" description="At least one special character" checked={formValues.require_symbol} onCheckedChange={(v) => handleUpdate({ require_symbol: v })} disabled={isBusy} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Breach & History</CardTitle>
              <p className="text-sm text-muted-foreground">Block commonly used passwords, check breach databases, and enforce password history.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-3 sm:grid-cols-2">
                <FormSwitchSubContainer label="Reject Common Passwords" description="Block passwords from known common-password lists" checked={formValues.reject_common_passwords} onCheckedChange={(v) => handleUpdate({ reject_common_passwords: v })} disabled={isBusy} />
                <FormSwitchSubContainer label="Check HIBP" description="Screen against Have I Been Pwned breach database" checked={formValues.check_hibp} onCheckedChange={(v) => handleUpdate({ check_hibp: v })} disabled={isBusy} />
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField
                  label="Password History Count"
                  description="Number of previous passwords to remember (0 = disabled)"
                  type="number"
                  min={0}
                  max={PASSWORD_POLICY_LIMITS.maxHistoryCount}
                  value={formValues.password_history_count.toString()}
                  onChange={(e) => handleUpdate({ password_history_count: parseInt(e.target.value) || 0 })}
                  error={errors.password_history_count?.message}
                  disabled={isBusy}
                />
                <FormInputField
                  label="Minimum Strength Score"
                  description="0–4 guessability threshold (0 = off, 2 = recommended). Scores how hard the password is to guess; character classes carry no weight."
                  type="number"
                  min={0}
                  max={4}
                  value={formValues.min_strength_score.toString()}
                  onChange={(e) => handleUpdate({ min_strength_score: parseInt(e.target.value) || 0 })}
                  error={errors.min_strength_score?.message}
                  disabled={isBusy}
                />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Expiry & Hashing</CardTitle>
              <p className="text-sm text-muted-foreground">Configure password expiration, temporary password validity, and hashing algorithm.</p>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField
                  label="Max Age (days)"
                  description="Days until password expires (0 = never)"
                  type="number"
                  min={0}
                  max={PASSWORD_POLICY_LIMITS.maxAgeDays}
                  value={formValues.max_age_days.toString()}
                  onChange={(e) => handleUpdate({ max_age_days: parseInt(e.target.value) || 0 })}
                  error={errors.max_age_days?.message}
                  disabled={isBusy}
                />
                <FormInputField
                  label="Temporary Password Validity (hours)"
                  type="number"
                  min={1}
                  max={PASSWORD_POLICY_LIMITS.maxTemporaryPasswordValidityHours}
                  value={formValues.temporary_password_validity_hours.toString()}
                  onChange={(e) => handleUpdate({ temporary_password_validity_hours: parseInt(e.target.value) || 1 })}
                  error={errors.temporary_password_validity_hours?.message}
                  disabled={isBusy}
                />
                <FormSelectField
                  label="Hashing Algorithm"
                  options={HASH_OPTIONS}
                  value={formValues.hash_algorithm}
                  onValueChange={(v) => handleUpdate({ hash_algorithm: v as PasswordPoliciesFormData['hash_algorithm'] })}
                  error={errors.hash_algorithm?.message}
                  disabled={isBusy}
                />
              </div>

              {/* This setting behaves unlike every other field here: the others
                  take effect on the next password check, this one only affects
                  passwords hashed AFTER the change. Nobody is locked out —
                  verification detects the stored format — but the estate stays
                  mixed, so the operator should know before saving. */}
              {hashAlgorithmChanged && (
                <Alert className="mt-4">
                  <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-600" aria-hidden="true" />
                  <AlertTitle>
                    Applies to new passwords only
                  </AlertTitle>
                  <AlertDescription>
                    Existing users keep signing in normally — stored passwords are verified with
                    the algorithm they were hashed with. They are only re-hashed with{' '}
                    {HASH_OPTIONS.find((o) => o.value === formValues.hash_algorithm)?.label ??
                      formValues.hash_algorithm}{' '}
                    the next time each user changes their password, so both algorithms will be in
                    use until then.
                  </AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>

          {/* What this configuration actually shows a user at signup. This is the
              affordance a config surface needs and a CRUD form does not: the operator
              is editing rules that appear somewhere else, so show them here. */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Preview</CardTitle>
              <p className="text-sm text-muted-foreground">
                How these requirements appear to a user setting their password.
              </p>
            </CardHeader>
            <CardContent>
              <PasswordRequirements password="" config={previewConfig} />
            </CardContent>
          </Card>

          <div className="flex flex-col-reverse gap-3 sm:flex-row sm:justify-between">
            {/* A config surface needs a way back to the shipped baseline. Without
                it, an operator who has drifted the policy has to remember 13
                values to undo it. */}
            <Button type="button" variant="ghost" onClick={restoreDefaults} disabled={isBusy}>
              <RotateCcw className="size-4" />
              Restore recommended defaults
            </Button>
            <div className="flex justify-end gap-3">
              <Button type="button" variant="outline" onClick={() => guard(() => navigate(backTo))} disabled={isBusy}>
                Cancel
              </Button>
              <FormSubmitButton isSubmitting={isBusy} submitText="Save Changes" />
            </div>
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
