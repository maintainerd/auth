import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { Save } from "lucide-react"
import { Button } from "@/components/ui/button"
import { FormInputField, FormSelectField } from "@/components/form"
import { FormSwitchSubContainer } from "@/components/inputs"
import { SettingsCard } from "@/components/card"
import { useAuditConfig, useUpdateAuditConfig } from "@/hooks/useAuditConfig"
import { useMaintenanceConfig, useUpdateMaintenanceConfig } from "@/hooks/useMaintenanceConfig"
import { useRateLimitConfig, useUpdateRateLimitConfig } from "@/hooks/useRateLimitConfig"
import { useToast } from "@/hooks/useToast"
import { toDatetimeLocalInput, toRfc3339 } from "@/lib/datetime"
import {
  auditConfigSchema,
  maintenanceConfigSchema,
  rateLimitConfigSchema,
  type AuditConfigFormData,
  type MaintenanceConfigFormData,
  type RateLimitConfigFormData,
} from "@/lib/validations"

const LOG_LEVELS = [
  { value: "debug", label: "Debug" },
  { value: "info", label: "Info" },
  { value: "warn", label: "Warn" },
  { value: "critical", label: "Critical" },
]

// A failed config load must never fall through to the form: it seeds from
// hardcoded schema defaults, so saving would overwrite the tenant's real
// settings with them — silent data loss on security-relevant controls.
function FailedSettings({ label }: { label: string }) {
  return (
    <div className="flex min-h-[320px] flex-col items-center justify-center gap-4">
      <p className="text-sm text-destructive">{label}</p>
    </div>
  )
}

function LoadingSettings({ label }: { label: string }) {
  return (
    <div className="flex min-h-[320px] flex-col items-center justify-center gap-4">
      <p className="text-muted-foreground">{label}</p>
    </div>
  )
}

export function RateLimitSettingsPanel() {
  const { showSuccess, showError } = useToast()
  const { data: savedConfig, isLoading, isError } = useRateLimitConfig()
  const updateMutation = useUpdateRateLimitConfig()

  const { handleSubmit, reset, watch, setValue, formState: { errors, isSubmitting } } = useForm<RateLimitConfigFormData>({
    resolver: yupResolver(rateLimitConfigSchema),
    defaultValues: { enabled: false, requests_per_window: 100, window_duration_seconds: 60, per_ip: true },
    mode: "onSubmit",
  })

  const formValues = watch()

  useEffect(() => {
    if (savedConfig) {
      reset({
        enabled: savedConfig.enabled ?? false,
        requests_per_window: savedConfig.requests_per_window ?? 100,
        window_duration_seconds: savedConfig.window_duration_seconds ?? 60,
        per_ip: savedConfig.per_ip ?? true,
      })
    }
  }, [savedConfig, reset])

  const handleUpdate = (updates: Partial<RateLimitConfigFormData>) => {
    Object.entries(updates).forEach(([key, value]) => {
      setValue(key as keyof RateLimitConfigFormData, value, { shouldValidate: false, shouldDirty: true })
    })
  }

  const onSubmit = async (data: RateLimitConfigFormData) => {
    try {
      await updateMutation.mutateAsync(data)
      showSuccess("Rate limit config saved successfully")
    } catch (error) {
      showError(error)
    }
  }

  if (isLoading) return <LoadingSettings label="Loading rate limit configuration..." />
  if (isError) return <FailedSettings label="Failed to load rate limit configuration." />

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="grid gap-6">
      <SettingsCard title="General" description="Enable or disable rate limiting.">
        <FormSwitchSubContainer
          label="Enabled"
          description="When enabled, requests exceeding configured limits will be rejected."
          checked={formValues.enabled}
          onCheckedChange={(v) => handleUpdate({ enabled: v })}
        />
      </SettingsCard>

      <SettingsCard title="Thresholds" description="Define the rate limiting window and request cap.">
        <div className="grid gap-4 md:grid-cols-2">
          <FormInputField
            label="Requests per Window"
            type="number"
            value={formValues.requests_per_window.toString()}
            onChange={(e) => handleUpdate({ requests_per_window: parseInt(e.target.value) || 1 })}
            error={errors.requests_per_window?.message}
          />
          <FormInputField
            label="Window Duration (seconds)"
            type="number"
            value={formValues.window_duration_seconds.toString()}
            onChange={(e) => handleUpdate({ window_duration_seconds: parseInt(e.target.value) || 1 })}
            error={errors.window_duration_seconds?.message}
          />
        </div>
      </SettingsCard>

      {/* Per-IP is the only scope the limiter implements — rate_limit.go reads
          per_ip and nothing else — so there is no "Per API Key" switch here. */}
      <SettingsCard title="Scope" description="Apply rate limits per IP address.">
        <div className="space-y-4">
          <FormSwitchSubContainer
            label="Per IP"
            description="Track and limit requests per unique IP address."
            checked={formValues.per_ip}
            onCheckedChange={(v) => handleUpdate({ per_ip: v })}
          />
        </div>
      </SettingsCard>

      <div className="flex justify-end">
        <Button type="submit" className="min-w-[140px] px-6" disabled={updateMutation.isPending || isSubmitting}>
          <Save className="mr-2 h-4 w-4" />
          {updateMutation.isPending || isSubmitting ? "Saving..." : "Save Changes"}
        </Button>
      </div>
    </form>
  )
}

export function AuditSettingsPanel() {
  const { showSuccess, showError } = useToast()
  const { data: savedConfig, isLoading, isError } = useAuditConfig()
  const updateMutation = useUpdateAuditConfig()

  const { handleSubmit, reset, watch, setValue, formState: { errors, isSubmitting } } = useForm<AuditConfigFormData>({
    resolver: yupResolver(auditConfigSchema),
    defaultValues: { enabled: false, retention_days: 90, pii_masking: false, log_level: "info" },
    mode: "onSubmit",
  })

  const formValues = watch()

  useEffect(() => {
    if (savedConfig) {
      reset({
        enabled: savedConfig.enabled ?? false,
        retention_days: savedConfig.retention_days ?? 90,
        pii_masking: savedConfig.pii_masking ?? false,
        log_level: savedConfig.log_level ?? "info",
      })
    }
  }, [savedConfig, reset])

  const handleUpdate = (updates: Partial<AuditConfigFormData>) => {
    Object.entries(updates).forEach(([key, value]) => {
      setValue(key as keyof AuditConfigFormData, value, { shouldValidate: false, shouldDirty: true })
    })
  }

  const onSubmit = async (data: AuditConfigFormData) => {
    try {
      await updateMutation.mutateAsync(data)
      showSuccess("Audit config saved successfully")
    } catch (error) {
      showError(error)
    }
  }

  if (isLoading) return <LoadingSettings label="Loading audit configuration..." />
  if (isError) return <FailedSettings label="Failed to load audit configuration." />

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="grid gap-6">
      <SettingsCard title="General" description="Enable or disable audit logging.">
        <FormSwitchSubContainer
          label="Enabled"
          description="When enabled, audit events will be recorded."
          checked={formValues.enabled}
          onCheckedChange={(v) => handleUpdate({ enabled: v })}
        />
      </SettingsCard>

      <SettingsCard title="Retention & Privacy" description="Configure audit log retention and privacy settings.">
        <div className="grid gap-4 md:grid-cols-2">
          <FormInputField
            label="Retention (days)"
            type="number"
            value={formValues.retention_days.toString()}
            onChange={(e) => handleUpdate({ retention_days: parseInt(e.target.value) || 1 })}
            error={errors.retention_days?.message}
          />
          <FormSelectField
            label="Log Level"
            options={LOG_LEVELS}
            value={formValues.log_level}
            onValueChange={(v) => handleUpdate({ log_level: v })}
            error={errors.log_level?.message}
          />
        </div>
        <div className="mt-4 space-y-4">
          <FormSwitchSubContainer
            label="PII Masking"
            description="Mask personally identifiable information in audit logs."
            checked={formValues.pii_masking}
            onCheckedChange={(v) => handleUpdate({ pii_masking: v })}
          />
        </div>
      </SettingsCard>

      <div className="flex justify-end">
        <Button type="submit" className="min-w-[140px] px-6" disabled={updateMutation.isPending || isSubmitting}>
          <Save className="mr-2 h-4 w-4" />
          {updateMutation.isPending || isSubmitting ? "Saving..." : "Save Changes"}
        </Button>
      </div>
    </form>
  )
}

export function MaintenanceSettingsPanel() {
  const { showSuccess, showError } = useToast()
  const { data: savedConfig, isLoading, isError } = useMaintenanceConfig()
  const updateMutation = useUpdateMaintenanceConfig()

  const { handleSubmit, reset, watch, setValue, formState: { errors, isSubmitting } } = useForm<MaintenanceConfigFormData>({
    resolver: yupResolver(maintenanceConfigSchema),
    defaultValues: { enabled: false, message: "The system is currently undergoing maintenance. Please try again later.", scheduled_start: null, scheduled_end: null },
    mode: "onSubmit",
  })

  const formValues = watch()

  useEffect(() => {
    if (savedConfig) {
      reset({
        enabled: savedConfig.enabled ?? false,
        message: savedConfig.message ?? "",
        // The schedule fields are edited as datetime-local, so the RFC3339
        // timestamps the API returns have to be narrowed to that shape here —
        // a raw RFC3339 string is not a value the input will display at all,
        // and the field would silently render blank over a live schedule.
        scheduled_start: toDatetimeLocalInput(savedConfig.scheduled_start),
        scheduled_end: toDatetimeLocalInput(savedConfig.scheduled_end),
      })
    }
  }, [savedConfig, reset])

  const handleUpdate = (updates: Partial<MaintenanceConfigFormData>) => {
    Object.entries(updates).forEach(([key, value]) => {
      setValue(key as keyof MaintenanceConfigFormData, value, { shouldValidate: false, shouldDirty: true })
    })
  }

  const onSubmit = async (data: MaintenanceConfigFormData) => {
    try {
      // The form holds datetime-local values ("2026-08-05T14:30"); the API parses
      // these with time.RFC3339 and rejects an empty string as well, so both
      // fields have to cross the boundary as a full timestamp or an explicit
      // null. Submitting the input values as-is is always a 422.
      await updateMutation.mutateAsync({
        ...data,
        scheduled_start: toRfc3339(data.scheduled_start),
        scheduled_end: toRfc3339(data.scheduled_end),
      })
      showSuccess("Maintenance config saved successfully")
    } catch (error) {
      showError(error)
    }
  }

  if (isLoading) return <LoadingSettings label="Loading maintenance configuration..." />
  if (isError) return <FailedSettings label="Failed to load maintenance configuration." />

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="grid gap-6">
      <SettingsCard title="Maintenance Mode" description="Toggle maintenance mode on or off.">
        <FormSwitchSubContainer
          label="Enabled"
          description="When enabled, users will see the maintenance message and be unable to access the application."
          checked={formValues.enabled}
          onCheckedChange={(v) => handleUpdate({ enabled: v })}
        />
      </SettingsCard>

      <SettingsCard title="Message" description="The message shown to users during maintenance.">
        <FormInputField
          label="Maintenance Message"
          value={formValues.message}
          onChange={(e) => handleUpdate({ message: e.target.value })}
          error={errors.message?.message}
        />
      </SettingsCard>

      <SettingsCard title="Schedule (optional)" description="Set scheduled start and end times for maintenance.">
        <div className="grid gap-4 md:grid-cols-2">
          <FormInputField
            label="Scheduled Start"
            type="datetime-local"
            value={formValues.scheduled_start ?? ""}
            onChange={(e) => handleUpdate({ scheduled_start: e.target.value || null })}
            error={errors.scheduled_start?.message}
          />
          <FormInputField
            label="Scheduled End"
            type="datetime-local"
            value={formValues.scheduled_end ?? ""}
            onChange={(e) => handleUpdate({ scheduled_end: e.target.value || null })}
            error={errors.scheduled_end?.message}
          />
        </div>
      </SettingsCard>

      <div className="flex justify-end">
        <Button type="submit" className="min-w-[140px] px-6" disabled={updateMutation.isPending || isSubmitting}>
          <Save className="mr-2 h-4 w-4" />
          {updateMutation.isPending || isSubmitting ? "Saving..." : "Save Changes"}
        </Button>
      </div>
    </form>
  )
}
