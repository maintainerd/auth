import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { yupResolver } from '@hookform/resolvers/yup'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { DetailsContainer } from '@/components/container'
import { FormPageHeader } from '@/components/header'
import { FormSwitchField, FormInputField, FormSelectField, FormCheckboxField, FormSubmitButton } from '@/components/form'
import { useTokenConfig, useUpdateTokenConfig } from '@/hooks/useTokenConfig'
import { useToast } from '@/hooks/useToast'
import { ConfirmationDialog } from '@/components/dialog'
import { useUnsavedChangesGuard } from '@/hooks/useUnsavedChangesGuard'
import { tokenConfigSchema, type TokenConfigFormData } from '@/lib/validations'

const BACKEND_FIELD_MAP: Record<string, string> = {
  clock_skew_leeway_seconds: 'clock_skew_leeway_seconds',
  signing_algorithm: 'signing_algorithm',
  require_pkce: 'require_pkce',
  additional_id_token_claims: 'additional_id_token_claims',
  additional_access_token_claims: 'additional_access_token_claims',
}

// ES256 is intentionally absent: the server's key store is RSA-only and rejects
// ES256, so only RS256/PS256 are offered here to match what can actually sign.
const SIGNING_OPTIONS = [
  { value: 'RS256', label: 'RS256' },
  { value: 'PS256', label: 'PS256' },
]

// Only roles and tenant_id are offered — these are the server-resolved
// authorization/tenancy claims the backend accepts. Identity claims (email,
// name, phone, …) are delivered to ID tokens by the OIDC profile/email/phone
// scopes, not this list, and do not belong in access tokens (RFC 9068 §6
// least-disclosure). Auth-context claims (acr/amr/nonce/…) must never be
// operator-set. The backend rejects anything outside this set, so offering more
// here would only let an operator pick a value the API refuses.
const KNOWN_CLAIMS = [
  { value: 'roles', label: 'roles' },
  { value: 'tenant_id', label: 'tenant_id' },
]

export default function TokenConfigPage() {
  const navigate = useNavigate()
  const { showSuccess, showError, parseError } = useToast()
  const backTo = `/security?tab=tokens`

  const { data: savedConfig, isLoading, isError } = useTokenConfig()
  const updateMutation = useUpdateTokenConfig()

  const { handleSubmit, reset, watch, setValue, setError, formState: { errors, isSubmitting, isDirty } } = useForm<TokenConfigFormData>({
    resolver: yupResolver(tokenConfigSchema),
    defaultValues: {
      clock_skew_leeway_seconds: 30,
      signing_algorithm: 'RS256',
      require_pkce: true,
      additional_id_token_claims: ['roles', 'tenant_id'],
      additional_access_token_claims: ['roles', 'tenant_id'],
    },
    mode: 'onSubmit',
  })

  const formValues = watch()

  useEffect(() => {
    if (savedConfig) {
      reset({
        clock_skew_leeway_seconds: savedConfig.clock_skew_leeway_seconds ?? 30,
        signing_algorithm: savedConfig.signing_algorithm ?? 'RS256',
        require_pkce: savedConfig.require_pkce ?? true,
        additional_id_token_claims: savedConfig.additional_id_token_claims ?? ['roles', 'tenant_id'],
        additional_access_token_claims: savedConfig.additional_access_token_claims ?? ['roles', 'tenant_id'],
      })
    }
  }, [savedConfig, reset])

  const handleUpdate = (updates: Partial<TokenConfigFormData>) => {
    Object.entries(updates).forEach(([key, value]) => {
      setValue(key as keyof TokenConfigFormData, value, { shouldValidate: false, shouldDirty: true })
    })
  }

  const toggleClaim = (claims: string[], claim: string, checked: boolean) => {
    return checked ? [...claims, claim] : claims.filter((c) => c !== claim)
  }

  const onSubmit = async (data: TokenConfigFormData) => {
    try {
      await updateMutation.mutateAsync(data)
      showSuccess('Token configuration saved successfully')
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
          ['additional_access_token_claims', 'additional_access_token_claims'],
          ['additional_id_token_claims', 'additional_id_token_claims'],
          ['clock_skew_leeway_seconds', 'clock_skew_leeway_seconds'],
          ['signing_algorithm', 'signing_algorithm'],
          ['require_pkce', 'require_pkce'],
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
          <FormPageHeader backUrl={backTo} onBack={() => guard(() => navigate(backTo))} backLabel="Back to Tokens" title="Configure Tokens" description="Set JWT signing, PKCE, and token claims." />
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
          <FormPageHeader backUrl={backTo} onBack={() => guard(() => navigate(backTo))} backLabel="Back to Tokens" title="Configure Tokens" description="Set JWT signing, PKCE, and token claims." />
          <Card>
            <CardContent className="py-12 text-center text-sm text-destructive">
              Failed to load token configuration.
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
          backLabel="Back to Tokens"
          title="Configure Tokens"
          description="Configure JWT signing algorithm, clock skew, PKCE, and additional token claims."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">JWT Settings</CardTitle>
              <p className="text-sm text-muted-foreground">Signing algorithm, clock skew tolerance, and PKCE enforcement.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <FormSelectField label="Signing Algorithm" options={SIGNING_OPTIONS} value={formValues.signing_algorithm} onValueChange={(v) => handleUpdate({ signing_algorithm: v as TokenConfigFormData['signing_algorithm'] })} error={errors.signing_algorithm?.message} disabled={isBusy} />
                <FormInputField label="Clock Skew Leeway (seconds)" description="0–300 seconds" type="number" value={formValues.clock_skew_leeway_seconds.toString()} onChange={(e) => handleUpdate({ clock_skew_leeway_seconds: parseInt(e.target.value) || 0 })} error={errors.clock_skew_leeway_seconds?.message} disabled={isBusy} />
              </div>
              <FormSwitchField label="Require PKCE" description="Enforce S256 PKCE for all OAuth authorization code flows" checked={formValues.require_pkce} onCheckedChange={(v) => handleUpdate({ require_pkce: v })} disabled={isBusy} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">ID Token Claims</CardTitle>
              <p className="text-sm text-muted-foreground">Extra claims injected into the ID token.</p>
            </CardHeader>
            <CardContent>
              {errors.additional_id_token_claims?.message && (
                <p className="text-sm text-red-600 mb-2">{errors.additional_id_token_claims.message}</p>
              )}
              <div className="grid gap-2 sm:grid-cols-3">
                {KNOWN_CLAIMS.map((claim) => (
                  <FormCheckboxField
                    key={`id-${claim.value}`}
                    label={claim.label}
                    checked={formValues.additional_id_token_claims.includes(claim.value)}
                    onCheckedChange={(checked) => handleUpdate({ additional_id_token_claims: toggleClaim(formValues.additional_id_token_claims, claim.value, checked) })}
                    disabled={isBusy}
                  />
                ))}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Access Token Claims</CardTitle>
              <p className="text-sm text-muted-foreground">Extra claims injected into the access token.</p>
            </CardHeader>
            <CardContent>
              {errors.additional_access_token_claims?.message && (
                <p className="text-sm text-red-600 mb-2">{errors.additional_access_token_claims.message}</p>
              )}
              <div className="grid gap-2 sm:grid-cols-3">
                {KNOWN_CLAIMS.map((claim) => (
                  <FormCheckboxField
                    key={`access-${claim.value}`}
                    label={claim.label}
                    checked={formValues.additional_access_token_claims.includes(claim.value)}
                    onCheckedChange={(checked) => handleUpdate({ additional_access_token_claims: toggleClaim(formValues.additional_access_token_claims, claim.value, checked) })}
                    disabled={isBusy}
                  />
                ))}
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
