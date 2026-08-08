import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { yupResolver } from '@hookform/resolvers/yup'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { DetailsContainer } from '@/components/container'
import { FormPageHeader } from '@/components/header'
import { FormInputField, FormSubmitButton } from '@/components/form'
import { FormSwitchSubContainer } from '@/components/inputs'
import { useLockoutConfig, useUpdateLockoutConfig } from '@/hooks/useLockoutConfig'
import { useToast } from '@/hooks/useToast'
import { ConfirmationDialog } from '@/components/dialog'
import { useUnsavedChangesGuard } from '@/hooks/useUnsavedChangesGuard'
import { lockoutConfigSchema, type LockoutConfigFormData } from '@/lib/validations'

const BACKEND_FIELD_MAP: Record<string, string> = {
  enabled: 'enabled',
  max_failed_attempts: 'max_failed_attempts',
  lockout_duration_minutes: 'lockout_duration_minutes',
  progressive_lockout: 'progressive_lockout',
  auto_unlock: 'auto_unlock',
  reset_count_on_success: 'reset_count_on_success',
  observation_window_minutes: 'observation_window_minutes',
  max_lockout_duration_minutes: 'max_lockout_duration_minutes',
  progression_reset_hours: 'progression_reset_hours',
  notify_user_on_lockout: 'notify_user_on_lockout',
}

export default function LockoutConfigPage() {
  const navigate = useNavigate()
  const { showSuccess, showError, parseError } = useToast()
  const backTo = `/security?tab=lockout`

  const { data: savedConfig, isLoading, isError } = useLockoutConfig()
  const updateMutation = useUpdateLockoutConfig()

  const { handleSubmit, reset, watch, setValue, setError, formState: { errors, isSubmitting, isDirty } } = useForm<LockoutConfigFormData>({
    resolver: yupResolver(lockoutConfigSchema),
    defaultValues: {
      enabled: true,
      max_failed_attempts: 5,
      lockout_duration_minutes: 30,
      progressive_lockout: true,
      auto_unlock: true,
      reset_count_on_success: true,
      observation_window_minutes: 15,
      max_lockout_duration_minutes: 60,
      progression_reset_hours: 24,
      notify_user_on_lockout: true,
    },
    mode: 'onSubmit',
  })

  const formValues = watch()

  useEffect(() => {
    if (savedConfig) {
      reset({
        enabled: savedConfig.enabled ?? true,
        max_failed_attempts: savedConfig.max_failed_attempts ?? 5,
        lockout_duration_minutes: savedConfig.lockout_duration_minutes ?? 30,
        progressive_lockout: savedConfig.progressive_lockout ?? true,
        auto_unlock: savedConfig.auto_unlock ?? true,
        reset_count_on_success: savedConfig.reset_count_on_success ?? true,
        observation_window_minutes: savedConfig.observation_window_minutes ?? 15,
        max_lockout_duration_minutes: savedConfig.max_lockout_duration_minutes ?? 60,
        progression_reset_hours: savedConfig.progression_reset_hours ?? 24,
        notify_user_on_lockout: savedConfig.notify_user_on_lockout ?? true,
      })
    }
  }, [savedConfig, reset])

  const handleUpdate = (updates: Partial<LockoutConfigFormData>) => {
    Object.entries(updates).forEach(([key, value]) => {
      setValue(key as keyof LockoutConfigFormData, value, { shouldValidate: false, shouldDirty: true })
    })
  }

  const onSubmit = async (data: LockoutConfigFormData) => {
    try {
      await updateMutation.mutateAsync(data)
      showSuccess('Lockout configuration saved successfully')
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
          ['max_lockout_duration_minutes', 'max_lockout_duration_minutes'],
          ['lockout_duration_minutes', 'lockout_duration_minutes'],
          ['observation_window_minutes', 'observation_window_minutes'],
          ['progression_reset_hours', 'progression_reset_hours'],
          ['notify_user_on_lockout', 'notify_user_on_lockout'],
          ['reset_count_on_success', 'reset_count_on_success'],
          ['max_failed_attempts', 'max_failed_attempts'],
          ['progressive_lockout', 'progressive_lockout'],
          ['auto_unlock', 'auto_unlock'],
          ['enabled', 'enabled'],
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
          <FormPageHeader backUrl={backTo} onBack={() => guard(() => navigate(backTo))} backLabel="Back to Account Lockout" title="Configure Account Lockout" description="Set lockout policies and behavior." />
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

  if (isError) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader backUrl={backTo} onBack={() => guard(() => navigate(backTo))} backLabel="Back to Account Lockout" title="Configure Account Lockout" description="Set lockout policies and behavior." />
          <Card>
            <CardContent className="py-12 text-center text-sm text-destructive">
              Failed to load lockout configuration.
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
          backLabel="Back to Account Lockout"
          title="Configure Account Lockout"
          description="Configure failed-login lockout policies, progressive escalation, and auto-unlock behavior."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Lockout Policy</CardTitle>
              <p className="text-sm text-muted-foreground">Set thresholds and duration for account lockouts.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormSwitchSubContainer
                label="Enable Lockout"
                description="Enable account lockout after repeated failed login attempts"
                checked={formValues.enabled}
                onCheckedChange={(checked) => handleUpdate({ enabled: checked })}
                error={errors.enabled?.message}
                disabled={isBusy}
              />
              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField label="Max Failed Attempts" type="number" value={formValues.max_failed_attempts.toString()} onChange={(e) => handleUpdate({ max_failed_attempts: parseInt(e.target.value) || 1 })} error={errors.max_failed_attempts?.message} disabled={isBusy} />
                <FormInputField label="Lockout Duration (minutes)" type="number" value={formValues.lockout_duration_minutes.toString()} onChange={(e) => handleUpdate({ lockout_duration_minutes: parseInt(e.target.value) || 1 })} error={errors.lockout_duration_minutes?.message} disabled={isBusy} />
                <FormInputField label="Observation Window (minutes)" type="number" value={formValues.observation_window_minutes.toString()} onChange={(e) => handleUpdate({ observation_window_minutes: parseInt(e.target.value) || 1 })} error={errors.observation_window_minutes?.message} disabled={isBusy} />
                <FormInputField label="Max Lockout Duration (minutes)" type="number" value={formValues.max_lockout_duration_minutes.toString()} onChange={(e) => handleUpdate({ max_lockout_duration_minutes: parseInt(e.target.value) || 1 })} error={errors.max_lockout_duration_minutes?.message} disabled={isBusy} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Progressive Lockout</CardTitle>
              <p className="text-sm text-muted-foreground">Escalate lockout duration on repeated lockouts.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormSwitchSubContainer label="Progressive Lockout" description="Increase lockout duration with each successive lockout" checked={formValues.progressive_lockout} onCheckedChange={(checked) => handleUpdate({ progressive_lockout: checked })} disabled={isBusy} />
              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField label="Progression Reset (hours)" type="number" value={formValues.progression_reset_hours.toString()} onChange={(e) => handleUpdate({ progression_reset_hours: parseInt(e.target.value) || 1 })} error={errors.progression_reset_hours?.message} disabled={isBusy} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Behavior</CardTitle>
              <p className="text-sm text-muted-foreground">Additional lockout behavior settings.</p>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3 sm:grid-cols-2">
                <FormSwitchSubContainer label="Auto Unlock After Duration" description="Automatically unlock after the lockout duration expires" checked={formValues.auto_unlock} onCheckedChange={(checked) => handleUpdate({ auto_unlock: checked })} disabled={isBusy} />
                <FormSwitchSubContainer label="Reset Count On Success" description="Reset failed attempt counter on successful login" checked={formValues.reset_count_on_success} onCheckedChange={(checked) => handleUpdate({ reset_count_on_success: checked })} disabled={isBusy} />
                <FormSwitchSubContainer label="Notify User On Lockout" description="Send notification to the user when their account is locked" checked={formValues.notify_user_on_lockout} onCheckedChange={(checked) => handleUpdate({ notify_user_on_lockout: checked })} disabled={isBusy} />
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
