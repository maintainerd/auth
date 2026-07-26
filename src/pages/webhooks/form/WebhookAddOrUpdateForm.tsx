import { useEffect, useState } from "react"
import { useParams, useNavigate, useLocation } from "react-router-dom"
import { useForm, Controller } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { AlertCircle, ArrowLeft, Copy, KeyRound, ShieldAlert } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { DetailsContainer } from "@/components/container"
import { FormPageHeader } from "@/components/header"
import {
  FormInputField,
  FormTextareaField,
  FormSelectField,
  FormSwitchField,
  FormSubmitButton,
  type SelectOption,
} from "@/components/form"
import { FormUrlField } from "@/components/inputs"
import { ConfirmationDialog } from "@/components/dialog"
import { webhookSchema, type WebhookFormData } from "@/lib/validations"
import {
  useWebhook,
  useCreateWebhook,
  useUpdateWebhook,
} from "@/hooks/useWebhooks"
import { useToast } from "@/hooks/useToast"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import type { CreateWebhookRequest, UpdateWebhookRequest } from "@/services/api/webhooks/types"
import { WEBHOOKS_BACK_STATE, WEBHOOKS_LIST_URL } from "../webhookNavigation"

const STATUS_OPTIONS: SelectOption[] = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
]

const BACKEND_FIELD_MAP: Record<string, keyof WebhookFormData> = {
  url: "url",
  description: "description",
  subscribe_all: "subscribeAll",
  max_retries: "maxRetries",
  timeout_seconds: "timeoutSeconds",
  status: "status",
}

export default function WebhookAddOrUpdateForm() {
  const { webhookId } = useParams<{ webhookId?: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError, parseError } = useToast()

  const isEditing = Boolean(webhookId)
  const navState = location.state as { from?: string; backLabel?: string } | null
  const detailUrl = webhookId ? `/webhooks/${webhookId}` : WEBHOOKS_LIST_URL
  const backTo = navState?.from ?? (isEditing ? detailUrl : WEBHOOKS_LIST_URL)
  const backLabel = navState?.backLabel ?? (isEditing ? "Back to Webhook Details" : "Back to Webhooks")

  const { data: webhookData, isLoading: isFetchingWebhook } = useWebhook(webhookId || "")
  const createWebhookMutation = useCreateWebhook()
  const updateWebhookMutation = useUpdateWebhook()

  // When editing, optionally generate a fresh signing secret on save.
  const [rotateSecret, setRotateSecret] = useState(false)
  // The plaintext signing secret is returned exactly once (on create, or on a
  // rotate). We hold it here to reveal it before leaving the page.
  const [revealedSecret, setRevealedSecret] = useState<string | null>(null)
  const [savedWebhookId, setSavedWebhookId] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    control,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<WebhookFormData>({
    resolver: yupResolver(webhookSchema),
    defaultValues: {
      url: "",
      description: "",
      subscribeAll: true,
      maxRetries: 3,
      timeoutSeconds: 30,
      status: "active",
    },
    mode: "onTouched",
    reValidateMode: "onChange",
  })

  useEffect(() => {
    if (isEditing && webhookData) {
      reset({
        url: webhookData.url,
        description: webhookData.description ?? "",
        subscribeAll: webhookData.subscribe_all,
        maxRetries: webhookData.max_retries,
        timeoutSeconds: webhookData.timeout_seconds,
        status: webhookData.status === "active" ? "active" : "inactive",
      })
    }
  }, [isEditing, webhookData, reset])

  const isLoading =
    isFetchingWebhook || createWebhookMutation.isPending || updateWebhookMutation.isPending || isSubmitting
  const existingWebhook = webhookData
  const pageTitle = isEditing ? `Edit ${existingWebhook?.url || "Webhook"}` : "Create Webhook"
  const submitButtonText = isEditing ? "Update Webhook" : "Create Webhook"

  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(
    !revealedSecret && (isDirty || rotateSecret),
  )

  const onSubmit = async (data: WebhookFormData) => {
    try {
      if (isEditing && webhookId) {
        const updateData: UpdateWebhookRequest = {
          url: data.url,
          subscribe_all: data.subscribeAll,
          rotate_secret: rotateSecret,
          description: data.description,
          max_retries: data.maxRetries,
          timeout_seconds: data.timeoutSeconds,
          status: data.status,
        }
        const updated = await updateWebhookMutation.mutateAsync({ webhookId, data: updateData })
        showSuccess("Webhook updated successfully")
        // A rotate returns a fresh secret once — reveal it before navigating.
        if (rotateSecret && updated.signing_secret) {
          setSavedWebhookId(updated.webhook_endpoint_id)
          setRevealedSecret(updated.signing_secret)
          return
        }
        navigate(backTo)
      } else {
        const createData: CreateWebhookRequest = {
          url: data.url,
          subscribe_all: data.subscribeAll,
          description: data.description,
          max_retries: data.maxRetries,
          timeout_seconds: data.timeoutSeconds,
          status: data.status,
        }
        const created = await createWebhookMutation.mutateAsync(createData)
        showSuccess("Webhook created successfully")
        setSavedWebhookId(created.webhook_endpoint_id)
        // The signing secret is shown exactly once — reveal it before navigating.
        if (created.signing_secret) {
          setRevealedSecret(created.signing_secret)
          return
        }
        navigate(backTo)
      }
    } catch (error) {
      const parsed = parseError(error)
      let mappedToField = false
      if (parsed.fieldErrors) {
        for (const [field, message] of Object.entries(parsed.fieldErrors)) {
          const formField = BACKEND_FIELD_MAP[field]
          if (formField) {
            setError(formField, { type: "server", message })
            mappedToField = true
          }
        }
      }
      if (!mappedToField) {
        const lower = parsed.message.toLowerCase()
        const keywordOrder: Array<[string, keyof WebhookFormData]> = [
          ["timeout", "timeoutSeconds"],
          ["retry", "maxRetries"],
          ["subscribe", "subscribeAll"],
          ["description", "description"],
          ["status", "status"],
          ["url", "url"],
        ]
        const hit = keywordOrder.find(([keyword]) => lower.includes(keyword))
        if (hit) {
          setError(hit[1], { type: "server", message: parsed.message })
        }
      }
      showError(error, "Failed to save webhook")
    }
  }

  const handleCancel = () => {
    if (isEditing && webhookId) {
      guard(() => navigate(backTo))
    } else {
      guard(() => navigate(WEBHOOKS_LIST_URL))
    }
  }

  const copySecret = async () => {
    if (!revealedSecret) return
    try {
      await navigator.clipboard.writeText(revealedSecret)
      showSuccess("Signing secret copied to clipboard")
    } catch {
      showError("Couldn't copy — copy it manually")
    }
  }

  const continueAfterSecretReveal = () => {
    if (!savedWebhookId) {
      navigate(WEBHOOKS_LIST_URL)
      return
    }

    navigate(`/webhooks/${savedWebhookId}`, {
      state: WEBHOOKS_BACK_STATE,
    })
  }

  if (isEditing && isFetchingWebhook) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Webhook"
            description="Update webhook endpoint configuration"
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

  if (isEditing && !isFetchingWebhook && !webhookData) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Webhook"
            description="Update webhook endpoint configuration"
          />
          <Card>
            <CardContent className="flex flex-col items-center justify-center gap-4 py-12 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <AlertCircle className="size-6" />
              </div>
              <div className="space-y-1">
                <h2 className="text-lg font-semibold">Webhook not found</h2>
                <p className="text-sm text-muted-foreground">
                  The webhook you're trying to edit doesn't exist or may have been removed.
                </p>
              </div>
              <Button variant="outline" onClick={() => navigate(backTo)}>
                <ArrowLeft className="mr-2 size-4" />
                {backLabel}
              </Button>
            </CardContent>
          </Card>
        </div>
      </DetailsContainer>
    )
  }

  // One-time secret reveal — shown after a create or a secret rotation.
  if (revealedSecret) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Save your signing secret"
            description="This is the only time the signing secret will be shown."
          />

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <KeyRound className="size-4" />
                Signing secret
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-start gap-2 rounded-md bg-amber-50 p-3 text-sm text-amber-900">
                <ShieldAlert className="mt-0.5 size-4 shrink-0" />
                <span>
                  Copy this secret now and store it securely. Use it to verify the signature on
                  each webhook delivery. You won't be able to view it again — you can only rotate it.
                </span>
              </div>

              <div className="flex items-center gap-2">
                <code className="min-w-0 flex-1 truncate rounded-md border bg-muted px-3 py-2 font-mono text-sm">
                  {revealedSecret}
                </code>
                <Button type="button" variant="outline" size="sm" className="h-9 gap-2" onClick={copySecret}>
                  <Copy className="size-4" />
                  Copy
                </Button>
              </div>

              <div className="flex justify-end">
                <Button type="button" onClick={continueAfterSecretReveal}>
                  I've saved it — continue
                </Button>
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
          backLabel={backLabel}
          onBack={() => guard(() => navigate(backTo))}
          title={pageTitle}
          description={
            isEditing
              ? "Update webhook endpoint configuration"
              : "Register an endpoint to receive signed event notifications"
          }
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6" key={webhookId || "create"}>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Endpoint</CardTitle>
              <p className="text-sm text-muted-foreground">
                The receiving URL, status, and event subscription behavior.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormUrlField
                label="Payload URL"
                placeholder="https://example.com/webhooks/maintainerd"
                description="Events are delivered as HTTP POST requests to this URL."
                disabled={isLoading}
                error={errors.url?.message}
                required
                {...register("url")}
              />

              <FormTextareaField
                label="Description"
                placeholder="What is this endpoint for? (optional)"
                rows={3}
                disabled={isLoading}
                error={errors.description?.message}
                {...register("description")}
              />

              <Controller
                name="subscribeAll"
                control={control}
                render={({ field }) => (
                  <FormSwitchField
                    label="Subscribe to all events"
                    description="Receive every event type. Turn this off to manage specific event subscriptions via the API."
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={isLoading}
                    error={errors.subscribeAll?.message}
                  />
                )}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Delivery Settings</CardTitle>
              <p className="text-sm text-muted-foreground">
                Control retry behavior and the per-attempt request timeout.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField
                  label="Max retries"
                  type="number"
                  min={0}
                  max={10}
                  description="Failed deliveries are retried up to this many times (0–10)."
                  disabled={isLoading}
                  error={errors.maxRetries?.message}
                  required
                  {...register("maxRetries", { valueAsNumber: true })}
                />

                <FormInputField
                  label="Timeout (seconds)"
                  type="number"
                  min={1}
                  max={120}
                  description="How long to wait for your endpoint to respond (1–120s)."
                  disabled={isLoading}
                  error={errors.timeoutSeconds?.message}
                  required
                  {...register("timeoutSeconds", { valueAsNumber: true })}
                />
              </div>

              <Controller
                name="status"
                control={control}
                render={({ field }) => (
                  <FormSelectField
                    label="Status"
                    placeholder="Select status"
                    options={STATUS_OPTIONS}
                    value={field.value}
                    onValueChange={field.onChange}
                    disabled={isLoading}
                    error={errors.status?.message}
                    required
                  />
                )}
              />

              {isEditing && (
                <FormSwitchField
                  label="Rotate signing secret"
                  description="Generate a new signing secret on save. The previous secret stops working immediately."
                  checked={rotateSecret}
                  onCheckedChange={setRotateSecret}
                  disabled={isLoading}
                />
              )}
            </CardContent>
          </Card>

          {/* Actions */}
          <div className="flex justify-end gap-3">
            <Button type="button" variant="outline" onClick={handleCancel} disabled={isLoading}>
              Cancel
            </Button>
            <FormSubmitButton
              isSubmitting={isSubmitting || isLoading}
              submittingText="Saving..."
              submitText={submitButtonText}
            />
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
