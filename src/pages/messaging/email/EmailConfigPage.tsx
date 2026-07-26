import { useEffect, useMemo } from "react"
import { useNavigate } from "react-router-dom"
import { useForm, Controller, type Resolver, type SubmitHandler } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import * as yup from "yup"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { DetailsContainer } from "@/components/container"
import { FormPageHeader } from "@/components/header"
import {
  FormInputField,
  FormSelectField,
  FormPasswordField,
  FormSwitchField,
  FormSubmitButton,
  type SelectOption,
} from "@/components/form"
import { FormEmailField, FormUrlField } from "@/components/inputs"
import { useToast } from "@/hooks/useToast"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import { ConfirmationDialog } from "@/components/dialog"
import { fetchEmailConfig, updateEmailConfig, type EmailConfigUpdate } from "@/services/api/notifier"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"

type EmailProviderMeta = {
  value: string
  label: string
  showServer: boolean
  usernameLabel?: string
  secretLabel: string
}

const EMAIL_PROVIDERS: EmailProviderMeta[] = [
  { value: "smtp", label: "SMTP", showServer: true, usernameLabel: "Username", secretLabel: "Password" },
  { value: "ses", label: "Amazon SES", showServer: false, usernameLabel: "Access Key ID", secretLabel: "Secret Access Key" },
  { value: "sendgrid", label: "SendGrid", showServer: false, secretLabel: "API Key" },
  { value: "mailgun", label: "Mailgun", showServer: false, secretLabel: "API Key" },
  { value: "postmark", label: "Postmark", showServer: false, secretLabel: "Server API Token" },
  { value: "resend", label: "Resend", showServer: false, secretLabel: "API Key" },
]

const PROVIDER_OPTIONS: SelectOption[] = EMAIL_PROVIDERS.map((p) => ({ value: p.value, label: p.label }))

const ENCRYPTION_OPTIONS: SelectOption[] = [
  { value: "tls", label: "TLS" },
  { value: "ssl", label: "SSL" },
  { value: "none", label: "None" },
]

const optionalUrl = (label: string) =>
  yup
    .string()
    .trim()
    .max(2048, `${label} must not exceed 2048 characters`)
    .matches(/^https?:\/\//, {
      excludeEmptyString: true,
      message: `${label} must start with http:// or https://`,
    })
    .default("")

const schema = yup.object({
  provider: yup.string().required("Provider is required"),
  host: yup.string(),
  port: yup.number().transform((v) => (isNaN(v) ? undefined : v)).min(1, "Port must be between 1 and 65535").max(65535, "Port must be between 1 and 65535"),
  username: yup.string(),
  password: yup.string(),
  from_address: yup.string().email("Enter a valid email address").required("From address is required"),
  from_name: yup.string(),
  reply_to: yup.string().email("Enter a valid email address"),
  encryption: yup.string(),
  logo_url: optionalUrl("Logo URL"),
  test_mode: yup.boolean(),
})

type FormData = yup.InferType<typeof schema>

export default function EmailConfigPage() {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const queryClient = useQueryClient()
  const backTo = `/messaging?tab=email`

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["email-config"],
    queryFn: fetchEmailConfig,
    retry: false,
  })

  const {
    register,
    handleSubmit,
    control,
    reset,
    watch,
    setError,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<FormData>({
    resolver: yupResolver(schema) as Resolver<FormData>,
    defaultValues: { provider: "smtp", from_address: "", encryption: "tls", test_mode: false },
    mode: "onTouched",
    reValidateMode: "onChange",
  })

  useEffect(() => {
    if (!data) return
    reset({
      provider: data.provider || "smtp",
      host: data.host || "",
      port: data.port || undefined,
      username: data.username || "",
      password: "",
      from_address: data.from_address || "",
      from_name: data.from_name || "",
      reply_to: data.reply_to || "",
      encryption: data.encryption || "tls",
      logo_url: data.logo_url || "",
      test_mode: data.test_mode ?? false,
    })
  }, [data, reset])

  const selectedProvider = watch("provider")
  const meta = useMemo(
    () => EMAIL_PROVIDERS.find((p) => p.value === selectedProvider) ?? EMAIL_PROVIDERS[0],
    [selectedProvider],
  )

  const isConfigured = Boolean(data?.provider && data?.from_address)
  const providerChanged = isConfigured && selectedProvider !== data?.provider

  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const mutation = useMutation({
    mutationFn: updateEmailConfig,
  })

  const onSubmit: SubmitHandler<FormData> = async (formData) => {
    if (providerChanged && !formData.password) {
      setError("password", {
        type: "manual",
        message: `Enter the ${meta.secretLabel.toLowerCase()} for ${meta.label} — the previous provider's secret can't be reused.`,
      })
      return
    }
    const payload: EmailConfigUpdate = {
      provider: formData.provider,
      from_address: formData.from_address,
      from_name: formData.from_name,
      reply_to: formData.reply_to,
      logo_url: formData.logo_url,
      test_mode: formData.test_mode,
      ...(meta.showServer
        ? { host: formData.host, port: formData.port, encryption: formData.encryption }
        : {}),
      ...(meta.usernameLabel ? { username: formData.username } : {}),
      ...(formData.password ? { password: formData.password } : {}),
    }
    try {
      await mutation.mutateAsync(payload)
      showSuccess("Email delivery settings saved")
      queryClient.invalidateQueries({ queryKey: ["email-config"] })
      navigate(backTo)
    } catch (e) {
      showError(e)
    }
  }

  const secretDescription = providerChanged
    ? `You switched providers — enter the ${meta.secretLabel.toLowerCase()} for ${meta.label}.`
    : isConfigured
      ? "Leave blank to keep the stored value, or enter a new value to replace it."
      : "Stored encrypted at rest and never shown again after saving."

  const isBusy = isSubmitting || mutation.isPending

  if (isLoading) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader backUrl={backTo} backLabel="Back to Email Delivery" onBack={() => guard(() => navigate(backTo))} title="Configure Email Delivery" description="Connect an email provider to send platform emails." />
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
          <FormPageHeader backUrl={backTo} backLabel="Back to Email Delivery" onBack={() => guard(() => navigate(backTo))} title="Configure Email Delivery" description="Connect an email provider to send platform emails." />
          <Card>
            <CardContent className="py-12 text-center text-sm text-destructive">
              Failed to load email configuration. {(error as Error)?.message || ""}
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
          backLabel="Back to Email Delivery"
          onBack={() => guard(() => navigate(backTo))}
          title="Configure Email Delivery"
          description="Connect an email provider so the platform can send verification, security, and notification emails."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Provider & Credentials</CardTitle>
              <p className="text-sm text-muted-foreground">
                Choose the service that delivers your email and enter its credentials.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <Controller
                  name="provider"
                  control={control}
                  render={({ field }) => (
                    <FormSelectField
                      label="Email Provider"
                      options={PROVIDER_OPTIONS}
                      value={field.value}
                      onValueChange={field.onChange}
                      error={errors.provider?.message}
                      disabled={isBusy}
                      required
                    />
                  )}
                />
              </div>

              {meta.showServer && (
                <div className="grid gap-4 md:grid-cols-3">
                  <FormInputField label="Host" placeholder="smtp.example.com" disabled={isBusy} error={errors.host?.message} {...register("host")} />
                  <FormInputField label="Port" type="number" placeholder="587" disabled={isBusy} error={errors.port?.message} {...register("port")} />
                  <Controller
                    name="encryption"
                    control={control}
                    render={({ field }) => (
                      <FormSelectField
                        label="Encryption"
                        options={ENCRYPTION_OPTIONS}
                        value={field.value || ""}
                        onValueChange={field.onChange}
                        disabled={isBusy}
                        error={errors.encryption?.message}
                      />
                    )}
                  />
                </div>
              )}

              <div className="grid gap-4 md:grid-cols-2">
                {meta.usernameLabel && (
                  <FormInputField
                    label={meta.usernameLabel}
                    placeholder={meta.usernameLabel}
                    disabled={isBusy}
                    error={errors.username?.message}
                    {...register("username")}
                  />
                )}
                <FormPasswordField
                  label={meta.secretLabel}
                  placeholder={isConfigured && !providerChanged ? "••••••••  (unchanged)" : `Enter ${meta.secretLabel.toLowerCase()}`}
                  description={secretDescription}
                  required={providerChanged}
                  disabled={isBusy}
                  error={errors.password?.message}
                  {...register("password")}
                />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Sender Identity</CardTitle>
              <p className="text-sm text-muted-foreground">
                How recipients see messages from your platform.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <FormEmailField label="From Address" placeholder="noreply@example.com" disabled={isBusy} error={errors.from_address?.message} required {...register("from_address")} />
                <FormInputField label="From Name" placeholder="Your App" disabled={isBusy} error={errors.from_name?.message} {...register("from_name")} />
                <FormEmailField label="Reply-To" placeholder="support@example.com" disabled={isBusy} error={errors.reply_to?.message} {...register("reply_to")} />
                <FormUrlField label="Logo URL" placeholder="https://example.com/logo.png" disabled={isBusy} error={errors.logo_url?.message} {...register("logo_url")} />
              </div>

              <Controller
                name="test_mode"
                control={control}
                render={({ field }) => (
                  <FormSwitchField
                    label="Test mode"
                    description="Keep the provider in test mode while you verify the setup. Turn off to deliver to real recipients."
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={isBusy}
                  />
                )}
              />
            </CardContent>
          </Card>

          <div className="flex justify-end gap-3">
            <Button type="button" variant="outline" onClick={() => guard(() => navigate(backTo))} disabled={isBusy}>
              Cancel
            </Button>
            <FormSubmitButton isSubmitting={isBusy} submitText="Save Changes" disabled={!isDirty} />
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
