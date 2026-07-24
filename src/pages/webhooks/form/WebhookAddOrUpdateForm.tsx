import { useEffect, useRef, useState } from "react"
import { useParams, useNavigate } from "react-router-dom"
import { useForm, Controller } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { Copy, KeyRound, ShieldAlert } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
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
import { webhookSchema, type WebhookFormData } from "@/lib/validations"
import {
  useWebhook,
  useCreateWebhook,
  useUpdateWebhook,
} from "@/hooks/useWebhooks"
import { useToast } from "@/hooks/useToast"
import type { CreateWebhookRequest, UpdateWebhookRequest } from "@/services/api/webhooks/types"

const STATUS_OPTIONS: SelectOption[] = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
]

export default function WebhookAddOrUpdateForm() {
  const { webhookId } = useParams<{ webhookId?: string }>()
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()

  const isEditing = Boolean(webhookId)
  const webhooksListUrl = `/events?tab=webhooks`

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
    formState: { errors, isSubmitting },
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
    mode: "onSubmit",
    reValidateMode: "onSubmit",
  })

  // Populate the form from the existing webhook exactly once when editing.
  const initialized = useRef(false)
  useEffect(() => {
    if (isEditing && webhookData && !initialized.current) {
      initialized.current = true
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
        navigate(`/webhooks/${webhookId}`)
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
        navigate(`/webhooks/${created.webhook_endpoint_id}`)
      }
    } catch (error) {
      showError(error, "Failed to save webhook")
    }
  }

  const handleCancel = () => {
    if (isEditing && webhookId) {
      navigate(`/webhooks/${webhookId}`)
    } else {
      navigate(webhooksListUrl)
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

  const isLoading =
    isFetchingWebhook || createWebhookMutation.isPending || updateWebhookMutation.isPending

  if (isEditing && isFetchingWebhook) {
    return (
      <DetailsContainer>
        <div className="flex min-h-[400px] flex-col items-center justify-center gap-4">
          <div className="text-center">
            <h2 className="text-2xl font-semibold">Loading...</h2>
            <p className="mt-2 text-muted-foreground">Fetching webhook details</p>
          </div>
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
            backUrl={webhooksListUrl}
            backLabel="Back to Webhooks"
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
                <Button
                  type="button"
                  onClick={() =>
                    navigate(
                      savedWebhookId
                        ? `/webhooks/${savedWebhookId}`
                        : webhooksListUrl,
                    )
                  }
                >
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
          backUrl={isEditing ? `/webhooks/${webhookId}` : webhooksListUrl}
          backLabel="Back to Webhooks"
          title={isEditing ? "Edit Webhook" : "Create New Webhook"}
          description={
            isEditing
              ? "Update this webhook endpoint's configuration"
              : "Register an endpoint to receive signed event notifications"
          }
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          {/* Endpoint */}
          <Card>
            <CardHeader>
              <CardTitle>Endpoint</CardTitle>
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

          {/* Delivery settings */}
          <Card>
            <CardHeader>
              <CardTitle>Delivery settings</CardTitle>
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
              submitText={isEditing ? "Update Webhook" : "Create Webhook"}
            />
          </div>
        </form>
      </div>
    </DetailsContainer>
  )
}
