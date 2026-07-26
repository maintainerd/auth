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
import { FormPhoneFieldWithCountry } from "@/components/inputs"
import { isValidPhone } from "@/lib/validations/regex"
import { useToast } from "@/hooks/useToast"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import { ConfirmationDialog } from "@/components/dialog"
import { fetchSMSConfig, updateSMSConfig, type SMSConfigUpdate } from "@/services/api/notifier"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"

type SMSProviderMeta = {
  value: string
  label: string
  accountSidLabel?: string
  authTokenLabel?: string
  showFromNumber: boolean
  showSenderId: boolean
  noCredentials?: boolean
}

const SMS_PROVIDERS: SMSProviderMeta[] = [
  { value: "twilio", label: "Twilio", accountSidLabel: "Account SID", authTokenLabel: "Auth Token", showFromNumber: true, showSenderId: false },
  { value: "sns", label: "AWS SNS", accountSidLabel: "Access Key ID", authTokenLabel: "Secret Access Key", showFromNumber: false, showSenderId: true },
  { value: "vonage", label: "Vonage", accountSidLabel: "API Key", authTokenLabel: "API Secret", showFromNumber: true, showSenderId: true },
  { value: "messagebird", label: "MessageBird", authTokenLabel: "Access Key", showFromNumber: true, showSenderId: true },
  { value: "log", label: "Log (testing)", showFromNumber: false, showSenderId: false, noCredentials: true },
]

const PROVIDER_OPTIONS: SelectOption[] = SMS_PROVIDERS.map((p) => ({ value: p.value, label: p.label }))

const schema = yup.object({
  provider: yup.string().required("Provider is required"),
  account_sid: yup.string().max(255, "Must not exceed 255 characters"),
  auth_token: yup.string(),
  from_number: yup
    .string()
    .max(50, "Must not exceed 50 characters")
    .test("phone-format", "Invalid phone number format", (value) => !value || isValidPhone(value)),
  sender_id: yup.string().max(50, "Must not exceed 50 characters"),
  daily_send_limit: yup.number().transform((v) => (isNaN(v) ? undefined : v)).min(0, "Must be zero or greater"),
  test_mode: yup.boolean(),
})

type FormData = yup.InferType<typeof schema>

export default function SMSConfigPage() {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const queryClient = useQueryClient()
  const backTo = `/messaging?tab=sms`

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["sms-config"],
    queryFn: fetchSMSConfig,
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
    defaultValues: { provider: "twilio", daily_send_limit: 1000, test_mode: false },
    mode: "onTouched",
    reValidateMode: "onChange",
  })

  useEffect(() => {
    if (!data) return
    reset({
      provider: data.provider || "twilio",
      account_sid: data.account_sid || "",
      auth_token: "",
      from_number: data.from_number || "",
      sender_id: data.sender_id || "",
      daily_send_limit: data.daily_send_limit ?? 1000,
      test_mode: data.test_mode ?? false,
    })
  }, [data, reset])

  const selectedProvider = watch("provider")
  const meta = useMemo(
    () => SMS_PROVIDERS.find((p) => p.value === selectedProvider) ?? SMS_PROVIDERS[0],
    [selectedProvider],
  )

  const isConfigured = Boolean(data?.provider)
  const providerChanged = isConfigured && selectedProvider !== data?.provider

  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const mutation = useMutation({
    mutationFn: updateSMSConfig,
  })

  const onSubmit: SubmitHandler<FormData> = async (formData) => {
    if (providerChanged && meta.authTokenLabel && !formData.auth_token) {
      setError("auth_token", {
        type: "manual",
        message: `Enter the ${meta.authTokenLabel.toLowerCase()} for ${meta.label} — the previous provider's secret can't be reused.`,
      })
      return
    }
    const payload: SMSConfigUpdate = {
      provider: formData.provider,
      daily_send_limit: formData.daily_send_limit,
      test_mode: formData.test_mode,
      ...(meta.accountSidLabel ? { account_sid: formData.account_sid } : {}),
      ...(meta.showFromNumber ? { from_number: formData.from_number } : {}),
      ...(meta.showSenderId ? { sender_id: formData.sender_id } : {}),
      ...(meta.authTokenLabel && formData.auth_token ? { auth_token: formData.auth_token } : {}),
    }
    try {
      await mutation.mutateAsync(payload)
      showSuccess("SMS delivery settings saved")
      queryClient.invalidateQueries({ queryKey: ["sms-config"] })
      navigate(backTo)
    } catch (e) {
      showError(e)
    }
  }

  const secretDescription = providerChanged
    ? `You switched providers — enter the ${(meta.authTokenLabel ?? "secret").toLowerCase()} for ${meta.label}.`
    : isConfigured
      ? "Leave blank to keep the stored value, or enter a new value to replace it."
      : "Stored encrypted at rest and never shown again after saving."

  const isBusy = isSubmitting || mutation.isPending

  if (isLoading) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader backUrl={backTo} backLabel="Back to SMS Delivery" onBack={() => guard(() => navigate(backTo))} title="Configure SMS Delivery" description="Connect an SMS provider to send platform text messages." />
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
          <FormPageHeader backUrl={backTo} backLabel="Back to SMS Delivery" onBack={() => guard(() => navigate(backTo))} title="Configure SMS Delivery" description="Connect an SMS provider to send platform text messages." />
          <Card>
            <CardContent className="py-12 text-center text-sm text-destructive">
              Failed to load SMS configuration. {(error as Error)?.message || ""}
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
          backLabel="Back to SMS Delivery"
          onBack={() => guard(() => navigate(backTo))}
          title="Configure SMS Delivery"
          description="Connect an SMS provider so the platform can send one-time codes and security alerts by text message."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Provider & Credentials</CardTitle>
              <p className="text-sm text-muted-foreground">
                Choose the service that delivers your text messages and enter its credentials.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <Controller
                  name="provider"
                  control={control}
                  render={({ field }) => (
                    <FormSelectField
                      label="SMS Provider"
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

              {!meta.noCredentials && (
                <div className="grid gap-4 md:grid-cols-2">
                  {meta.accountSidLabel && (
                    <FormInputField
                      label={meta.accountSidLabel}
                      placeholder={meta.accountSidLabel}
                      disabled={isBusy}
                      error={errors.account_sid?.message}
                      {...register("account_sid")}
                    />
                  )}
                  {meta.authTokenLabel && (
                    <FormPasswordField
                      label={meta.authTokenLabel}
                      placeholder={isConfigured && !providerChanged ? "••••••••  (unchanged)" : `Enter ${meta.authTokenLabel.toLowerCase()}`}
                      description={secretDescription}
                      required={providerChanged}
                      disabled={isBusy}
                      error={errors.auth_token?.message}
                      {...register("auth_token")}
                    />
                  )}
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Sender & Options</CardTitle>
              <p className="text-sm text-muted-foreground">
                The originator recipients see and delivery controls.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              {(meta.showFromNumber || meta.showSenderId) && (
                <div className="grid gap-4 md:grid-cols-2">
                  {meta.showFromNumber && (
                    <Controller
                      name="from_number"
                      control={control}
                      render={({ field }) => (
                        <FormPhoneFieldWithCountry
                          label="From Number"
                          description="Optional sender phone number shown to recipients"
                          disabled={isBusy}
                          error={errors.from_number?.message}
                          value={field.value}
                          onChange={field.onChange}
                          onBlur={field.onBlur}
                        />
                      )}
                    />
                  )}
                  {meta.showSenderId && (
                    <FormInputField label="Sender ID" placeholder="MyApp" disabled={isBusy} error={errors.sender_id?.message} {...register("sender_id")} />
                  )}
                </div>
              )}

              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField
                  label="Daily Send Limit"
                  type="number"
                  placeholder="1000"
                  description="Maximum messages per day. Helps cap spend if credentials are misused."
                  disabled={isBusy}
                  error={errors.daily_send_limit?.message}
                  {...register("daily_send_limit")}
                />
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
