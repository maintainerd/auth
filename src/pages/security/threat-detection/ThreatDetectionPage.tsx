import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { yupResolver } from '@hookform/resolvers/yup'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { DetailsContainer } from '@/components/container'
import { FormPageHeader } from '@/components/header'
import { FormSwitchField, FormInputField, FormSubmitButton } from '@/components/form'
import { useThreatDetectionSettings, useUpdateThreatDetectionSettings } from '@/hooks/useThreatDetectionSettings'
import { useToast } from '@/hooks/useToast'
import { ConfirmationDialog } from '@/components/dialog'
import { useUnsavedChangesGuard } from '@/hooks/useUnsavedChangesGuard'
import { threatDetectionSettingsSchema, type ThreatDetectionSettingsFormData } from '@/lib/validations'

const BACKEND_FIELD_MAP: Record<string, string> = {
  brute_force_detection_enabled: 'brute_force_detection_enabled',
  impossible_travel_detection_enabled: 'impossible_travel_detection_enabled',
  new_device_notification_enabled: 'new_device_notification_enabled',
  velocity_check_enabled: 'velocity_check_enabled',
  risk_based_step_up_enabled: 'risk_based_step_up_enabled',
  compromised_credential_monitoring_enabled: 'compromised_credential_monitoring_enabled',
  ip_reputation_check_enabled: 'ip_reputation_check_enabled',
  block_tor_exit_nodes: 'block_tor_exit_nodes',
  risk_step_up_threshold: 'risk_step_up_threshold',
  risk_block_threshold: 'risk_block_threshold',
  velocity_failures_per_ip_per_hour: 'velocity_failures_per_ip_per_hour',
}

export default function ThreatDetectionPage() {
  const navigate = useNavigate()
  const { showSuccess, showError, parseError } = useToast()
  const backTo = `/security?tab=threat`

  const { data: savedSettings, isLoading } = useThreatDetectionSettings()
  const updateMutation = useUpdateThreatDetectionSettings()

  const { handleSubmit, reset, watch, setValue, setError, formState: { errors, isSubmitting, isDirty } } = useForm<ThreatDetectionSettingsFormData>({
    resolver: yupResolver(threatDetectionSettingsSchema),
    defaultValues: {
      brute_force_detection_enabled: true,
      impossible_travel_detection_enabled: true,
      new_device_notification_enabled: true,
      velocity_check_enabled: true,
      risk_based_step_up_enabled: false,
      compromised_credential_monitoring_enabled: true,
      ip_reputation_check_enabled: false,
      block_tor_exit_nodes: false,
      risk_step_up_threshold: 21,
      risk_block_threshold: 81,
      velocity_failures_per_ip_per_hour: 50,
    },
    mode: 'onSubmit',
  })

  const formValues = watch()

  useEffect(() => {
    if (savedSettings) {
      reset({
        brute_force_detection_enabled: savedSettings.brute_force_detection_enabled ?? true,
        impossible_travel_detection_enabled: savedSettings.impossible_travel_detection_enabled ?? true,
        new_device_notification_enabled: savedSettings.new_device_notification_enabled ?? true,
        velocity_check_enabled: savedSettings.velocity_check_enabled ?? true,
        risk_based_step_up_enabled: savedSettings.risk_based_step_up_enabled ?? false,
        compromised_credential_monitoring_enabled: savedSettings.compromised_credential_monitoring_enabled ?? true,
        ip_reputation_check_enabled: savedSettings.ip_reputation_check_enabled ?? false,
        block_tor_exit_nodes: savedSettings.block_tor_exit_nodes ?? false,
        risk_step_up_threshold: savedSettings.risk_step_up_threshold ?? 21,
        risk_block_threshold: savedSettings.risk_block_threshold ?? 81,
        velocity_failures_per_ip_per_hour: savedSettings.velocity_failures_per_ip_per_hour ?? 50,
      })
    }
  }, [savedSettings, reset])

  const handleUpdate = (updates: Partial<ThreatDetectionSettingsFormData>) => {
    Object.entries(updates).forEach(([key, value]) => {
      setValue(key as keyof ThreatDetectionSettingsFormData, value, { shouldValidate: false, shouldDirty: true })
    })
  }

  const onSubmit = async (data: ThreatDetectionSettingsFormData) => {
    try {
      await updateMutation.mutateAsync(data)
      showSuccess('Attack protection settings saved successfully')
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
          ['compromised_credential_monitoring_enabled', 'compromised_credential_monitoring_enabled'],
          ['impossible_travel_detection_enabled', 'impossible_travel_detection_enabled'],
          ['new_device_notification_enabled', 'new_device_notification_enabled'],
          ['velocity_failures_per_ip_per_hour', 'velocity_failures_per_ip_per_hour'],
          ['velocity_check_enabled', 'velocity_check_enabled'],
          ['risk_based_step_up_enabled', 'risk_based_step_up_enabled'],
          ['risk_step_up_threshold', 'risk_step_up_threshold'],
          ['risk_block_threshold', 'risk_block_threshold'],
          ['ip_reputation_check_enabled', 'ip_reputation_check_enabled'],
          ['brute_force_detection_enabled', 'brute_force_detection_enabled'],
          ['block_tor_exit_nodes', 'block_tor_exit_nodes'],
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
          <FormPageHeader backUrl={backTo} onBack={() => guard(() => navigate(backTo))} backLabel="Back to Threat Protection" title="Configure Threat Protection" description="Set threat detection and risk policies." />
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
          backLabel="Back to Threat Protection"
          title="Configure Threat Protection"
          description="Configure brute-force detection, velocity checks, risk-based step-up, and compromised credential monitoring."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Detection Engines</CardTitle>
              <p className="text-sm text-muted-foreground">Threat detection subsystems that run on every login.</p>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3 sm:grid-cols-2">
                <FormSwitchField label="Brute Force Detection" description="Detect repeated failed login attempts per account" checked={formValues.brute_force_detection_enabled} onCheckedChange={(v) => handleUpdate({ brute_force_detection_enabled: v })} disabled={isBusy} />
                <FormSwitchField label="Impossible Travel Detection" description="Flag logins from geographically impossible locations" checked={formValues.impossible_travel_detection_enabled} onCheckedChange={(v) => handleUpdate({ impossible_travel_detection_enabled: v })} disabled={isBusy} />
                <FormSwitchField label="New Device Notification" description="Alert users when a login occurs from a new device" checked={formValues.new_device_notification_enabled} onCheckedChange={(v) => handleUpdate({ new_device_notification_enabled: v })} disabled={isBusy} />
                <FormSwitchField label="Velocity Check" description="Rate-limit login attempts per IP address" checked={formValues.velocity_check_enabled} onCheckedChange={(v) => handleUpdate({ velocity_check_enabled: v })} disabled={isBusy} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Risk-Based Controls</CardTitle>
              <p className="text-sm text-muted-foreground">Step-up and blocking based on a cumulative risk score (0-100).</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-3 sm:grid-cols-2">
                <FormSwitchField label="Risk-Based Step-Up" description="Force MFA when risk score reaches the step-up threshold" checked={formValues.risk_based_step_up_enabled} onCheckedChange={(v) => handleUpdate({ risk_based_step_up_enabled: v })} disabled={isBusy} />
                <FormSwitchField label="Compromised Credential Monitoring" description="Check credentials against known breach databases" checked={formValues.compromised_credential_monitoring_enabled} onCheckedChange={(v) => handleUpdate({ compromised_credential_monitoring_enabled: v })} disabled={isBusy} />
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField label="Step-Up Threshold" description="Risk score >= this forces MFA" type="number" value={formValues.risk_step_up_threshold.toString()} onChange={(e) => handleUpdate({ risk_step_up_threshold: parseInt(e.target.value) || 0 })} error={errors.risk_step_up_threshold?.message} disabled={isBusy} />
                <FormInputField label="Block Threshold" description="Risk score >= this blocks the login" type="number" value={formValues.risk_block_threshold.toString()} onChange={(e) => handleUpdate({ risk_block_threshold: parseInt(e.target.value) || 0 })} error={errors.risk_block_threshold?.message} disabled={isBusy} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Network & IP</CardTitle>
              <p className="text-sm text-muted-foreground">IP reputation checks, Tor blocking, and per-IP velocity limits.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-3 sm:grid-cols-2">
                <FormSwitchField label="IP Reputation Check" description="Query external IP reputation feeds" checked={formValues.ip_reputation_check_enabled} onCheckedChange={(v) => handleUpdate({ ip_reputation_check_enabled: v })} disabled={isBusy} />
                <FormSwitchField label="Block Tor Exit Nodes" description="Reject logins originating from known Tor exit nodes" checked={formValues.block_tor_exit_nodes} onCheckedChange={(v) => handleUpdate({ block_tor_exit_nodes: v })} disabled={isBusy} />
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField label="Velocity Limit (failures/IP/hour)" type="number" value={formValues.velocity_failures_per_ip_per_hour.toString()} onChange={(e) => handleUpdate({ velocity_failures_per_ip_per_hour: parseInt(e.target.value) || 1 })} error={errors.velocity_failures_per_ip_per_hour?.message} disabled={isBusy} />
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
