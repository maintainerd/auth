import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { yupResolver } from '@hookform/resolvers/yup'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { DetailsContainer } from '@/components/container'
import { FormPageHeader } from '@/components/header'
import { FormInputField, FormTextareaField, FormSubmitButton } from '@/components/form'
import { FormSwitchSubContainer } from '@/components/inputs'
import { useRegistrationConfig, useUpdateRegistrationConfig } from '@/hooks/useRegistrationConfig'
import { useToast } from '@/hooks/useToast'
import { ConfirmationDialog } from '@/components/dialog'
import { useUnsavedChangesGuard } from '@/hooks/useUnsavedChangesGuard'
import { registrationConfigSchema, type RegistrationConfigFormData } from '@/lib/validations'

const BACKEND_FIELD_MAP: Record<string, string> = {
  self_registration_enabled: 'self_registration_enabled',
  require_email_verification: 'require_email_verification',
  require_phone_verification: 'require_phone_verification',
  auto_confirm_enabled: 'auto_confirm_enabled',
  verification_token_ttl_hours: 'verification_token_ttl_hours',
  captcha_on_signup: 'captcha_on_signup',
  registration_rate_limit_per_ip_per_hour: 'registration_rate_limit_per_ip_per_hour',
  allowed_email_domains: 'allowed_email_domains',
  blocked_email_domains: 'blocked_email_domains',
}

export default function RegistrationConfigPage() {
  const navigate = useNavigate()
  const { showSuccess, showError, parseError } = useToast()
  const backTo = `/security?tab=registration`

  const { data: savedConfig, isLoading, isError } = useRegistrationConfig()
  const updateMutation = useUpdateRegistrationConfig()

  const { handleSubmit, reset, watch, setValue, setError, formState: { errors, isSubmitting, isDirty } } = useForm<RegistrationConfigFormData>({
    resolver: yupResolver(registrationConfigSchema),
    defaultValues: {
      self_registration_enabled: true,
      require_email_verification: true,
      require_phone_verification: false,
      auto_confirm_enabled: false,
      verification_token_ttl_hours: 24,
      captcha_on_signup: false,
      registration_rate_limit_per_ip_per_hour: 10,
      allowed_email_domains: [],
      blocked_email_domains: [],
    },
    mode: 'onSubmit',
  })

  const formValues = watch()

  useEffect(() => {
    if (savedConfig) {
      reset({
        self_registration_enabled: savedConfig.self_registration_enabled ?? true,
        require_email_verification: savedConfig.require_email_verification ?? true,
        require_phone_verification: savedConfig.require_phone_verification ?? false,
        auto_confirm_enabled: savedConfig.auto_confirm_enabled ?? false,
        verification_token_ttl_hours: savedConfig.verification_token_ttl_hours ?? 24,
        captcha_on_signup: savedConfig.captcha_on_signup ?? false,
        registration_rate_limit_per_ip_per_hour: savedConfig.registration_rate_limit_per_ip_per_hour ?? 10,
        allowed_email_domains: savedConfig.allowed_email_domains ?? [],
        blocked_email_domains: savedConfig.blocked_email_domains ?? [],
      })
    }
  }, [savedConfig, reset])

  const handleUpdate = (updates: Partial<RegistrationConfigFormData>) => {
    Object.entries(updates).forEach(([key, value]) => {
      setValue(key as keyof RegistrationConfigFormData, value, { shouldValidate: false, shouldDirty: true })
    })
  }

  const parseDomains = (text: string): string[] =>
    text.split(/[\n,]/).map((d) => d.trim().toLowerCase()).filter(Boolean)

  const domainsToText = (domains: string[]): string =>
    (domains || []).join('\n')

  const onSubmit = async (data: RegistrationConfigFormData) => {
    try {
      await updateMutation.mutateAsync(data)
      showSuccess('Registration configuration saved successfully')
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
          ['registration_rate_limit_per_ip_per_hour', 'registration_rate_limit_per_ip_per_hour'],
          ['verification_token_ttl_hours', 'verification_token_ttl_hours'],
          ['self_registration_enabled', 'self_registration_enabled'],
          ['require_phone_verification', 'require_phone_verification'],
          ['require_email_verification', 'require_email_verification'],
          ['auto_confirm_enabled', 'auto_confirm_enabled'],
          ['captcha_on_signup', 'captcha_on_signup'],
          ['blocked_email_domains', 'blocked_email_domains'],
          ['allowed_email_domains', 'allowed_email_domains'],
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
          <FormPageHeader backUrl={backTo} onBack={() => guard(() => navigate(backTo))} backLabel="Back to Registration" title="Configure Registration" description="Set registration and verification policies." />
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

  // A failed load must NOT fall through to the form. The form seeds itself
  // from hardcoded defaults, so an admin who saves would overwrite the
  // tenant's real settings with them — silent, security-relevant data loss.
  if (isError) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader backUrl={backTo} onBack={() => guard(() => navigate(backTo))} backLabel="Back to Registration" title="Configure Registration" description="Set registration and verification policies." />
          <Card>
            <CardContent className="py-12 text-center text-sm text-destructive">
              Failed to load registration configuration.
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
          backLabel="Back to Registration"
          title="Configure Registration"
          description="Configure self-registration, verification, domain rules, and rate limiting."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Registration Settings</CardTitle>
              <p className="text-sm text-muted-foreground">Control how users can register.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-3 sm:grid-cols-2">
                <FormSwitchSubContainer label="Self Registration" description="Allow users to register themselves" checked={formValues.self_registration_enabled} onCheckedChange={(v) => handleUpdate({ self_registration_enabled: v })} disabled={isBusy} />
                {/* Shipping in a later release: the backend verifies a captcha token when
                    CAPTCHA_SECRET is configured, but no widget is rendered on the hosted
                    registration form yet, so enabling this would make signup impossible to
                    complete. The control stays disabled — but it must show the PERSISTED
                    value, not a hardcoded false. Hardcoding it meant a tenant whose stored
                    flag was true (the old seeded default) saw "off" here while every signup
                    was being rejected, and the page offered no way to discover or fix it. */}
                <FormSwitchSubContainer label="CAPTCHA on Signup (Coming soon)" description={formValues.captcha_on_signup ? "Enabled for this tenant, but the hosted signup form does not render a captcha challenge yet — registration will be rejected until this is turned off." : "Bot protection during registration. Not yet available — the hosted signup form does not render a captcha challenge."} checked={formValues.captcha_on_signup} onCheckedChange={(v) => handleUpdate({ captcha_on_signup: v })} disabled={isBusy || !formValues.captcha_on_signup} />
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField label="Rate Limit (per IP per hour)" type="number" value={formValues.registration_rate_limit_per_ip_per_hour.toString()} onChange={(e) => handleUpdate({ registration_rate_limit_per_ip_per_hour: parseInt(e.target.value) || 1 })} error={errors.registration_rate_limit_per_ip_per_hour?.message} disabled={isBusy} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Domain Rules</CardTitle>
              <p className="text-sm text-muted-foreground">Restrict registration to specific email domains. One domain per line. Supports wildcard patterns like *.example.com.</p>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-2">
                <FormTextareaField label="Allowed Email Domains" description="Leave empty to allow all domains" value={domainsToText(formValues.allowed_email_domains)} onChange={(e) => handleUpdate({ allowed_email_domains: parseDomains(e.target.value) })} error={errors.allowed_email_domains?.message} rows={5} disabled={isBusy} />
                <FormTextareaField label="Blocked Email Domains" description="Domains explicitly denied (takes precedence)" value={domainsToText(formValues.blocked_email_domains)} onChange={(e) => handleUpdate({ blocked_email_domains: parseDomains(e.target.value) })} error={errors.blocked_email_domains?.message} rows={5} disabled={isBusy} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Verification</CardTitle>
              <p className="text-sm text-muted-foreground">Configure email and phone verification requirements.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-3 sm:grid-cols-2">
                <FormSwitchSubContainer label="Require Email Verification" description="Users must verify email before full access" checked={formValues.require_email_verification} onCheckedChange={(v) => handleUpdate({ require_email_verification: v })} disabled={isBusy} />
                <FormSwitchSubContainer label="Require Phone Verification" description="Users must verify phone before full access" checked={formValues.require_phone_verification} onCheckedChange={(v) => handleUpdate({ require_phone_verification: v })} disabled={isBusy} />
                <FormSwitchSubContainer label="Auto-Confirm Accounts" description="Skip verification and immediately activate new accounts" checked={formValues.auto_confirm_enabled} onCheckedChange={(v) => handleUpdate({ auto_confirm_enabled: v })} disabled={isBusy} />
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField label="Verification Token TTL (hours)" type="number" value={formValues.verification_token_ttl_hours.toString()} onChange={(e) => handleUpdate({ verification_token_ttl_hours: parseInt(e.target.value) || 1 })} error={errors.verification_token_ttl_hours?.message} disabled={isBusy} />
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
