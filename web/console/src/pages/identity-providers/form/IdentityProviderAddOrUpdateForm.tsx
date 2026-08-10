import { useEffect, useState } from "react"
import { useParams, useNavigate, useLocation } from "react-router-dom"
import { useForm, Controller, type Resolver } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { useMutation } from "@tanstack/react-query"
import { ArrowLeft, AlertCircle, CheckCircle2, XCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { DetailsContainer } from "@/components/container"
import { FormPageHeader } from "@/components/header"
import {
  FormInputField,
  FormPasswordField,
  FormSelectField,
  FormSubmitButton,
  FormTextareaField,
  type SelectOption
} from "@/components/form"
import { FormSlugField, FormSwitchSubContainer, FormUrlField } from "@/components/inputs"
import { sanitizeName } from "@/lib/validations/regex"
import { ConfirmationDialog } from "@/components/dialog"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import {
  ProviderConfigSection,
  useProviderConfig,
  PROVIDER_SELECT_OPTIONS,
  getProviderConnectionSchema,
  getProviderKind,
  getPrefillIssuer,
  isOAuth2OnlyProvider,
} from "@/components/provider-config"
import { identityProviderSchema, type IdentityProviderFormData } from "@/lib/validations"
import { useAppSelector } from "@/store/hooks"
import { useIdentityProvider, useCreateIdentityProvider, useUpdateIdentityProvider } from "@/hooks/useIdentityProviders"
import { useToast } from "@/hooks/useToast"
import { testIdentityProviderConnection, type TestConnectionResult } from "@/services/api/identity-providers"
import type { ProviderOption, IdentityProviderStatus } from "@/services/api/identity-providers/types"

const PROVIDER_OPTIONS: SelectOption[] = PROVIDER_SELECT_OPTIONS

const STATUS_OPTIONS: SelectOption[] = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
]

// Tokenizer for list fields submitted to the backend (email domains, allowed
// audiences). Splits on whitespace, comma, or newline — kept identical to the
// yup emailDomains tokenizer and useProviderConfig's parseList so a value that
// validates is split into exactly the same tokens on submit (space-separated
// domains no longer collapse into one invalid token the backend rejects).
function parseList(value: string | null | undefined): string[] {
  return (value ?? "")
    .split(/[\s,\n]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function formatList(values: string[] | null | undefined): string {
  return (values ?? []).join(", ")
}

export default function IdentityProviderAddOrUpdateForm() {
  const { providerId } = useParams<{ providerId?: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError, parseError } = useToast()
  const currentTenant = useAppSelector((state) => state.tenant.currentTenant)

  const isEditing = Boolean(providerId)

  // Honour where we came from (e.g. the details page) so the back button,
  // cancel, and post-save all return there. Falls back to the listing.
  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/providers/identity`
  const backLabel = navState?.backLabel ?? "Back to Identity Providers"

  // Fetch existing provider if editing
  const { data: providerData, isLoading: isFetchingProvider } = useIdentityProvider(providerId || '')
  const createProviderMutation = useCreateIdentityProvider()
  const updateProviderMutation = useUpdateIdentityProvider()
  const [testResult, setTestResult] = useState<TestConnectionResult | null>(null)

  const testConnectionMutation = useMutation({
    mutationFn: (data: Record<string, unknown>) => testIdentityProviderConnection(data),
    onSuccess: (result) => setTestResult(result),
    onError: (err: unknown) => setTestResult({ success: false, message: err instanceof Error ? err.message : "Connection test failed" }),
  })

  // React Hook Form setup
  const {
    register,
    handleSubmit,
    control,
    reset,
    watch,
    getValues,
    setValue,
    setError,
    formState: { errors, isSubmitting, isDirty }
  } = useForm<IdentityProviderFormData>({
    resolver: yupResolver(identityProviderSchema) as Resolver<IdentityProviderFormData>,
    defaultValues: {
      name: "",
      displayName: "",
      provider: "maintainerd",
      status: "active",
      issuer: "",
      clientId: "",
      clientSecret: "",
      // New providers default JIT ON: a fresh external connection auto-provisions
      // accounts on first login unless the admin unchecks it. (Edit mode below
      // always shows the provider's stored value, never this default.)
      allowJITProvisioning: true,
      allowRegistration: true,
      allowTokenFederation: false,
      allowedAudiences: "",
      emailDomains: "",
    },
    mode: 'onTouched',
    reValidateMode: 'onChange'
  })

  // Dynamic, provider-aware configuration. Re-renders its fields whenever the
  // selected provider changes; both the well-known and additional fields are
  // merged into a single `config` JSON on save.
  const selectedProvider = watch("provider")
  const selectedStatus = watch("status")
  const allowTokenFederation = watch("allowTokenFederation")
  const connectionSchema = getProviderConnectionSchema(selectedProvider)
  const providerConfig = useProviderConfig(selectedProvider)

  // The built-in provider is the seeded, undeletable system record (is_system).
  // It authenticates locally and has no external OIDC connection/config, so its
  // (read-only) edit view hides the connection + config cards. Every other
  // maintainerd provider is a regular external federation to a different
  // organization's Maintainerd instance and shows the full OIDC connection UI.
  const isBuiltInSystem = providerData?.is_system === true
  const showConnection = Boolean(connectionSchema) && !isBuiltInSystem
  // Test Connection probes OIDC discovery (/.well-known/openid-configuration) +
  // JWKS, so it's only meaningful for OIDC providers. OAuth2-only providers
  // (github/facebook/x) publish no discovery document, so the probe can't apply.
  const canTestConnection = showConnection && !isOAuth2OnlyProvider(selectedProvider)
  // Token federation accepts foreign OIDC ID tokens from this issuer, so it only
  // makes sense for OIDC providers — the ones that have an issuer. OAuth2-only
  // providers (github/facebook/x) issue no OIDC ID token and expose no issuer,
  // and SAML/system have no OIDC connection, so the toggle is hidden for them.
  // (Left visible it would demand an issuer the form correctly hides → an
  // unfillable "issuer required" error.)
  const supportsTokenFederation = showConnection && !isOAuth2OnlyProvider(selectedProvider)

  // Load existing provider data if editing
  useEffect(() => {
    if (isEditing && providerData) {
      reset({
        name: providerData.name,
        displayName: providerData.display_name,
        provider: providerData.provider,
        status: providerData.status as 'active' | 'inactive',
        issuer: providerData.issuer ?? "",
        clientId: providerData.provider_client_id ?? "",
        clientSecret: "",
        allowJITProvisioning: providerData.allow_jit_provisioning ?? false,
        allowRegistration: providerData.allow_registration ?? true,
        allowTokenFederation: providerData.allow_token_federation ?? false,
        allowedAudiences: formatList(providerData.allowed_audiences),
        emailDomains: formatList(providerData.email_domains),
      })

      providerConfig.load(providerData.config, providerData.provider)
    }
    // providerConfig.load is stable; intentionally keyed to the fetched record
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isEditing, providerData, reset])

  // On create, pre-fill the Issuer URL with the provider's well-known constant
  // (e.g. https://gitlab.com) so the operator doesn't retype a fixed value.
  // Providers with a per-install issuer (Cognito/Auth0/Microsoft/Maintainerd)
  // resolve to "" and keep their placeholder. Editable; re-applied on switch.
  // Never runs in edit mode — the reset() above restores the saved issuer.
  useEffect(() => {
    if (isEditing) return
    setValue("issuer", getPrefillIssuer(selectedProvider))
  }, [selectedProvider, isEditing, setValue])

  // Token federation is invalid for providers with no OIDC issuer. Force it off
  // whenever it isn't supported so a stale `true` (e.g. left over from switching
  // away from an OIDC provider) can't require an issuer the form hides — the
  // "issuer required" trap on OAuth2-only providers.
  useEffect(() => {
    if (!supportsTokenFederation) {
      setValue("allowTokenFederation", false)
    }
  }, [supportsTokenFederation, setValue])

  // Warn before discarding unsaved edits (browser close/refresh + guarded exits).
  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const onSubmit = async (formData: IdentityProviderFormData) => {
    if (!currentTenant) {
      showError("Tenant information not available")
      return
    }

    const activeExternalProvider = Boolean(showConnection && formData.status === "active")
    if (activeExternalProvider && !isEditing && !(formData.clientSecret ?? "").trim()) {
      setError("clientSecret", {
        type: "manual",
        message: "Client secret is required when creating an active external provider",
      })
      showError("Please complete the required connection fields.")
      return
    }

    // Thread the selected status through so config presence ("required") checks
    // only fire for an active provider — matching the backend, which requires
    // config fields only when status === "active". Format checks always run.
    if (!isBuiltInSystem && !providerConfig.validate(selectedStatus)) {
      showError("Please complete the required provider configuration fields.")
      return
    }

    try {
      // The built-in system record keeps its system type; every other provider
      // derives its type from the selected provider (maintainerd → enterprise).
      const config = isBuiltInSystem ? {} : providerConfig.buildConfig()
      const providerType = isBuiltInSystem ? "system" : getProviderKind(formData.provider)
      const clientSecret = (formData.clientSecret ?? "").trim()
      const payload = {
        name: formData.name,
        display_name: formData.displayName,
        provider: formData.provider as ProviderOption,
        provider_type: providerType,
        issuer: showConnection ? (formData.issuer ?? "").trim() || null : null,
        provider_client_id: showConnection ? (formData.clientId ?? "").trim() || null : null,
        allow_jit_provisioning: showConnection ? Boolean(formData.allowJITProvisioning) : false,
        allow_registration: Boolean(formData.allowRegistration),
        allow_token_federation: Boolean(formData.allowTokenFederation),
        allowed_audiences: parseList(formData.allowedAudiences),
        email_domains: showConnection ? parseList(formData.emailDomains) : [],
        config,
        status: formData.status as IdentityProviderStatus,
        ...(clientSecret ? { provider_client_secret: clientSecret } : {}),
      }

      if (isEditing && providerId) {
        await updateProviderMutation.mutateAsync({
          identityProviderId: providerId,
          data: payload
        })
        showSuccess("Identity provider updated successfully")
      } else {
        await createProviderMutation.mutateAsync(payload)
        showSuccess("Identity provider created successfully")
      }

      // Navigate back to wherever we came from
      navigate(backTo)
    } catch (error) {
      // Route backend errors onto the offending field where we can: structured
      // field errors first, otherwise keyword-match the message. Anything
      // unmapped still shows via the toast. The backend keys field errors by its
      // snake_case JSON tag, so translate those to the form's RHF field names.
      const parsed = parseError(error)
      const BACKEND_FIELD_MAP: Record<string, keyof IdentityProviderFormData> = {
        name: "name",
        display_name: "displayName",
        provider: "provider",
        provider_type: "provider",
        status: "status",
        issuer: "issuer",
        provider_client_id: "clientId",
        provider_client_secret: "clientSecret",
        allowed_audiences: "allowedAudiences",
        email_domains: "emailDomains",
      }
      let mappedToField = false
      if (parsed.fieldErrors) {
        for (const [field, message] of Object.entries(parsed.fieldErrors)) {
          const rhfField = BACKEND_FIELD_MAP[field]
          if (rhfField) {
            setError(rhfField, { type: "server", message })
            mappedToField = true
          }
        }
      }
      if (!mappedToField) {
        const lower = parsed.message.toLowerCase()
        const match = Object.entries(BACKEND_FIELD_MAP).find(
          ([backendField, rhfField]) => lower.includes(backendField) || lower.includes(rhfField.toLowerCase())
        )
        if (match) {
          setError(match[1], { type: "server", message: parsed.message })
        }
      }
      showError(error)
    }
  }

  const isLoading = createProviderMutation.isPending || updateProviderMutation.isPending || isFetchingProvider
  const fieldsDisabled = providerData?.is_system || isLoading

  const handleTestConnection = async () => {
    setTestResult(null)
    const formData = getValues()
    const clientSecret = (formData.clientSecret ?? "").trim()
    // Field names must match the backend TestConnectionRequestDTO
    // (provider / discovery_url / client_id / client_secret). The probe derives
    // the discovery document from discovery_url (= the OIDC issuer).
    const payload: Record<string, unknown> = {
      provider: formData.provider,
      discovery_url: (formData.issuer ?? "").trim(),
      client_id: (formData.clientId ?? "").trim(),
      ...(clientSecret ? { client_secret: clientSecret } : {}),
    }
    testConnectionMutation.mutate(payload)
  }

  // Loading state while fetching the provider to edit
  if (isEditing && isFetchingProvider) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Identity Provider"
            description="Update the identity provider configuration and settings."
          />
          <Card>
            <CardContent className="space-y-4 pt-6">
              <Skeleton className="h-5 w-40" />
              <div className="grid gap-4 md:grid-cols-2">
                {Array.from({ length: 4 }).map((_, i) => (
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

  // Not-found state
  if (isEditing && !isFetchingProvider && !providerData) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Identity Provider"
            description="Update the identity provider configuration and settings."
          />
          <Card>
            <CardContent className="flex flex-col items-center justify-center gap-4 py-12 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <AlertCircle className="size-6" />
              </div>
              <div className="space-y-1">
                <h2 className="text-lg font-semibold">Identity provider not found</h2>
                <p className="text-sm text-muted-foreground">
                  The identity provider you're looking for doesn't exist or may have been removed.
                </p>
              </div>
              <Button variant="outline" onClick={() => guard(() => navigate(backTo))}>
                <ArrowLeft className="mr-2 size-4" />
                {backLabel}
              </Button>
            </CardContent>
          </Card>
        </div>
      </DetailsContainer>
    )
  }

  return (
    <DetailsContainer>
      <div className="flex flex-col gap-6">
        {/* Header */}
        <FormPageHeader
          title={isEditing ? "Edit Identity Provider" : "Create Identity Provider"}
          description={
            isEditing
              ? "Update the identity provider configuration and settings."
              : "Configure a new identity provider to handle user authentication for your application."
          }
          backUrl={backTo}
          backLabel={backLabel}
          onBack={() => guard(() => navigate(backTo))}
          showSystemBadge={isBuiltInSystem}
          showWarning={isBuiltInSystem}
          warningMessage="This is the built-in system provider. It authenticates locally and cannot be edited."
        />

        {/* Form */}
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6" key={providerId || "create"}>
          {/* Basic Information */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Basic Information</CardTitle>
              <p className="text-sm text-muted-foreground">
                The provider name, type, status, and registration policies.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormSlugField
                label="Name"
                placeholder="e.g., corporate-auth0, aws-cognito"
                description="Lowercase letters, numbers, and hyphens only"
                sanitize={sanitizeName}
                disabled={fieldsDisabled}
                error={errors.name?.message}
                required
                {...register("name")}
              />

              <FormInputField
                label="Display Name"
                placeholder="e.g., Corporate Auth0, AWS Cognito"
                description="This will be the display name shown to users"
                disabled={fieldsDisabled}
                error={errors.displayName?.message}
                required
                {...register("displayName")}
              />

              <div className="grid gap-4 md:grid-cols-2">
                <Controller
                  name="provider"
                  control={control}
                  render={({ field }) => (
                    <FormSelectField
                      key={`provider-${field.value || 'empty'}`}
                      label="Provider"
                      placeholder="Select provider"
                      options={PROVIDER_OPTIONS}
                      value={field.value}
                      onValueChange={field.onChange}
                      disabled={fieldsDisabled}
                      error={errors.provider?.message}
                      description="Changing the provider updates the configuration fields below"
                      required
                    />
                  )}
                />

                <Controller
                  name="status"
                  control={control}
                  render={({ field }) => (
                    <FormSelectField
                      key={`status-${field.value || 'empty'}`}
                      label="Status"
                      placeholder="Select status"
                      options={STATUS_OPTIONS}
                      value={field.value}
                      onValueChange={field.onChange}
                      disabled={fieldsDisabled}
                      error={errors.status?.message}
                      required
                    />
                  )}
                />
              </div>

              <Controller
                name="allowRegistration"
                control={control}
                render={({ field }) => (
                  <FormSwitchSubContainer
                    id="allow-registration"
                    label="Allow registration"
                    description="Allow clients using this provider to create new accounts. Tenant and registration-flow registration policies still apply."
                    checked={Boolean(field.value)}
                    onCheckedChange={field.onChange}
                    disabled={fieldsDisabled}
                  />
                )}
              />

              {supportsTokenFederation && (
                <Controller
                  name="allowTokenFederation"
                  control={control}
                  render={({ field }) => (
                    <FormSwitchSubContainer
                      id="allow-token-federation"
                      label="Allow token federation"
                      description="Accept foreign OIDC ID tokens from this issuer (Mode B / PDP). Requires issuer URL and at least one allowed audience."
                      checked={Boolean(field.value)}
                      onCheckedChange={field.onChange}
                      disabled={fieldsDisabled}
                    />
                  )}
                />
              )}

              {supportsTokenFederation && allowTokenFederation && (
                <Controller
                  name="allowedAudiences"
                  control={control}
                  render={({ field, fieldState }) => (
                    <FormTextareaField
                      id="allowed-audiences"
                      label="Allowed audiences"
                      description='External app client IDs that may present tokens from this issuer. One per line.'
                      placeholder="my-external-app&#10;another-app"
                      error={fieldState.error?.message}
                      disabled={fieldsDisabled}
                      {...field}
                      value={field.value ?? ""}
                      onChange={field.onChange}
                    />
                  )}
                />
              )}
            </CardContent>
          </Card>

          {showConnection && connectionSchema && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Connection</CardTitle>
                <p className="text-sm text-muted-foreground">{connectionSchema.summary}</p>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-4 md:grid-cols-2">
                  {connectionSchema.fields.map((field) => {
                    if (field.key === "allow_jit_provisioning") {
                      return (
                        <Controller
                          key={field.key}
                          name="allowJITProvisioning"
                          control={control}
                          render={({ field: formField }) => (
                            <FormSwitchSubContainer
                              id="allow-jit-provisioning"
                              label={field.label}
                              description={field.description}
                              checked={Boolean(formField.value)}
                              onCheckedChange={formField.onChange}
                              disabled={fieldsDisabled}
                              containerClassName="md:col-span-2"
                            />
                          )}
                        />
                      )
                    }

                    if (field.key === "email_domains") {
                      return (
                        <FormTextareaField
                          key={field.key}
                          label={field.label}
                          placeholder={field.placeholder}
                          description={field.description}
                          disabled={fieldsDisabled}
                          error={errors.emailDomains?.message}
                          containerClassName="md:col-span-2"
                          {...register("emailDomains")}
                        />
                      )
                    }

                    if (field.key === "provider_client_secret") {
                      return (
                        <FormPasswordField
                          key={field.key}
                          label={field.label}
                          placeholder={field.placeholder}
                          description={
                            isEditing
                              ? "Leave blank to keep the stored secret, or enter a new value to replace it."
                              : field.description
                          }
                          disabled={fieldsDisabled}
                          error={errors.clientSecret?.message}
                          required={field.required && selectedStatus === "active" && !isEditing}
                          {...register("clientSecret")}
                        />
                      )
                    }

                    if (field.type === "url") {
                      return (
                        <FormUrlField
                          key={field.key}
                          label={field.label}
                          placeholder={field.placeholder}
                          description={field.description}
                          disabled={fieldsDisabled}
                          error={field.key === "issuer" ? errors.issuer?.message : errors.clientId?.message}
                          required={field.required && selectedStatus === "active"}
                          {...register(field.key === "issuer" ? "issuer" : "clientId")}
                        />
                      )
                    }

                    return (
                      <FormInputField
                        key={field.key}
                        label={field.label}
                        type="text"
                        placeholder={field.placeholder}
                        description={field.description}
                        disabled={fieldsDisabled}
                        error={field.key === "issuer" ? errors.issuer?.message : errors.clientId?.message}
                        required={field.required && selectedStatus === "active"}
                        {...register(field.key === "issuer" ? "issuer" : "clientId")}
                      />
                    )
                  })}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Provider-aware configuration (external providers only; the built-in
              system provider authenticates locally and has no config). */}
          {!isBuiltInSystem && (
            <ProviderConfigSection
              provider={selectedProvider}
              controller={providerConfig}
              disabled={fieldsDisabled}
            />
          )}

          {/* Actions */}
          <div className="flex items-center justify-between gap-3">
            {canTestConnection ? (
              <Button
                type="button"
                variant="secondary"
                onClick={handleTestConnection}
                disabled={isLoading || testConnectionMutation.isPending}
              >
                {testConnectionMutation.isPending ? "Testing..." : "Test Connection"}
              </Button>
            ) : (
              // Keeps the action bar's justify-between layout when there is no
              // discovery-based connection to test (OAuth2-only / SAML / built-in).
              <span />
            )}
            <div className="flex items-center gap-3">
              <Button
                type="button"
                variant="outline"
                onClick={() => guard(() => navigate(backTo))}
                disabled={isLoading}
              >
                Cancel
              </Button>
              <FormSubmitButton
                isSubmitting={isLoading || isSubmitting}
                disabled={providerData?.is_system}
                submittingText="Saving..."
                submitText={isEditing ? "Update Provider" : "Create Provider"}
              />
            </div>
          </div>
          {testResult && (
            <Alert variant={testResult.success ? "default" : "destructive"}>
              {testResult.success ? <CheckCircle2 /> : <XCircle />}
              <AlertTitle>
                {testResult.success ? "Connection test passed" : "Connection test failed"}
              </AlertTitle>
              <AlertDescription>
                {testResult.message && <p>{testResult.message}</p>}
                {testResult.checks && testResult.checks.length > 0 && (
                  <ul className="space-y-1">
                    {testResult.checks.map((c, i) => (
                      <li key={i} className="flex items-start gap-2">
                        {c.ok ? (
                          <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-primary" aria-hidden="true" />
                        ) : (
                          <XCircle className="mt-0.5 size-4 shrink-0 text-destructive" aria-hidden="true" />
                        )}
                        <span className="sr-only">{c.ok ? "Passed:" : "Failed:"}</span>
                        <span>{c.step}</span>
                        {c.error && <span className="text-muted-foreground">— {c.error}</span>}
                      </li>
                    ))}
                  </ul>
                )}
              </AlertDescription>
            </Alert>
          )}
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
