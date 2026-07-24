import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { yupResolver } from '@hookform/resolvers/yup'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { DetailsContainer } from '@/components/container'
import { FormPageHeader } from '@/components/header'
import { FormSwitchField, FormInputField, FormSelectField, FormSubmitButton } from '@/components/form'
import { ConfirmationDialog } from '@/components/dialog'
import { usePasswordPolicies, useUpdatePasswordPolicies } from '@/hooks/usePasswordPolicies'
import { useToast } from '@/hooks/useToast'
import { useUnsavedChangesGuard } from '@/hooks/useUnsavedChangesGuard'
import {
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

export default function PasswordPoliciesFormPage() {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const backTo = `/security?tab=password`

  const { data: savedPolicies, isError } = usePasswordPolicies()
  const updateMutation = useUpdatePasswordPolicies()

  const {
    handleSubmit,
    reset,
    watch,
    setValue,
    trigger,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<PasswordPoliciesFormData>({
    resolver: yupResolver(passwordPoliciesSchema),
    defaultValues: {
      min_length: 12,
      max_length: 128,
      require_uppercase: false,
      require_lowercase: false,
      require_number: false,
      require_symbol: false,
      reject_common_passwords: true,
      check_hibp: true,
      password_history_count: 5,
      max_age_days: 0,
      temporary_password_validity_hours: 72,
      hash_algorithm: 'argon2id',
      min_strength_score: 2,
    },
    mode: 'onChange',
    reValidateMode: 'onChange',
  })

  const formValues = watch()

  useEffect(() => {
    if (savedPolicies) {
      reset({
        min_length: savedPolicies.min_length ?? 12,
        max_length: savedPolicies.max_length ?? 128,
        require_uppercase: savedPolicies.require_uppercase ?? false,
        require_lowercase: savedPolicies.require_lowercase ?? false,
        require_number: savedPolicies.require_number ?? false,
        require_symbol: savedPolicies.require_symbol ?? false,
        reject_common_passwords: savedPolicies.reject_common_passwords ?? true,
        check_hibp: savedPolicies.check_hibp ?? true,
        password_history_count: savedPolicies.password_history_count ?? 5,
        max_age_days: savedPolicies.max_age_days ?? 0,
        temporary_password_validity_hours: savedPolicies.temporary_password_validity_hours ?? 72,
        hash_algorithm: savedPolicies.hash_algorithm ?? 'argon2id',
        min_strength_score: savedPolicies.min_strength_score ?? 2,
      })
    }
  }, [savedPolicies, reset])

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

  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const onSubmit = async (data: PasswordPoliciesFormData) => {
    try {
      await updateMutation.mutateAsync(data)
      showSuccess('Password policy saved successfully')
      navigate(backTo)
    } catch (error) {
      showError(error)
    }
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
                  type="number"
                  min={1}
                  max={PASSWORD_POLICY_LIMITS.maxLength}
                  value={formValues.min_length.toString()}
                  onChange={(e) => handleUpdate({ min_length: parseInt(e.target.value) || 1 })}
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
                <FormSwitchField label="Require Uppercase" description="At least one A–Z character" checked={formValues.require_uppercase} onCheckedChange={(v) => handleUpdate({ require_uppercase: v })} disabled={isBusy} />
                <FormSwitchField label="Require Lowercase" description="At least one a–z character" checked={formValues.require_lowercase} onCheckedChange={(v) => handleUpdate({ require_lowercase: v })} disabled={isBusy} />
                <FormSwitchField label="Require Number" description="At least one 0–9 digit" checked={formValues.require_number} onCheckedChange={(v) => handleUpdate({ require_number: v })} disabled={isBusy} />
                <FormSwitchField label="Require Symbol" description="At least one special character" checked={formValues.require_symbol} onCheckedChange={(v) => handleUpdate({ require_symbol: v })} disabled={isBusy} />
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
                <FormSwitchField label="Reject Common Passwords" description="Block passwords from known common-password lists" checked={formValues.reject_common_passwords} onCheckedChange={(v) => handleUpdate({ reject_common_passwords: v })} disabled={isBusy} />
                <FormSwitchField label="Check HIBP" description="Screen against Have I Been Pwned breach database" checked={formValues.check_hibp} onCheckedChange={(v) => handleUpdate({ check_hibp: v })} disabled={isBusy} />
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
                  description="0–4 zxcvbn-like threshold (2 = good)"
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
