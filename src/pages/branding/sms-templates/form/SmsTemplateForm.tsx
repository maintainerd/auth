import { useEffect, useRef } from "react"
import { useParams, useNavigate, useLocation } from "react-router-dom"
import { useForm, Controller } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import * as yup from "yup"
import { ArrowLeft, AlertCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { DetailsContainer } from "@/components/container"
import { FormPageHeader } from "@/components/header"
import {
  FormInputField,
  FormSelectField,
  FormTextareaField,
  FormSubmitButton,
  type SelectOption
} from "@/components/form"
import { ConfirmationDialog } from "@/components/dialog"
import { useSmsTemplate, useUpdateSmsTemplate } from "@/hooks/useSmsTemplates"
import { useToast } from "@/hooks/useToast"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import type { SmsTemplateStatus } from "@/services/api/sms-templates/types"

const smsTemplateSchema = yup.object({
  name: yup.string().required('Name is required').min(3, 'Name must be at least 3 characters'),
  description: yup.string().required('Description is required').min(3, 'Description must be at least 3 characters'),
  message: yup.string().required('Message is required').min(1, 'Message cannot be empty'),
  status: yup.string().oneOf(['active', 'inactive']).required('Status is required'),
})

type SmsTemplateFormData = yup.InferType<typeof smsTemplateSchema>

const STATUS_OPTIONS: SelectOption[] = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
]

// Backend snake_case field keys → form field names.
const BACKEND_FIELD_MAP: Record<string, keyof SmsTemplateFormData> = {
  name: "name",
  description: "description",
  message: "message",
  status: "status",
}

export default function SmsTemplateForm() {
  const { templateId } = useParams<{ templateId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError, parseError } = useToast()

  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/branding?tab=sms-templates`
  const backLabel = navState?.backLabel ?? (backTo.includes("sms-templates") ? "Back to SMS Templates" : "Back")

  const { data: templateData, isLoading: isFetchingTemplate } = useSmsTemplate(templateId || '')
  const updateMutation = useUpdateSmsTemplate()

  const {
    register,
    handleSubmit,
    control,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<SmsTemplateFormData>({
    resolver: yupResolver(smsTemplateSchema),
    defaultValues: { name: "", description: "", message: "", status: "active" },
    mode: "onTouched",
    reValidateMode: "onChange",
  })

  const initialized = useRef(false)
  useEffect(() => {
    if (templateData && !initialized.current) {
      initialized.current = true
      reset({
        name: templateData.name,
        description: templateData.description,
        message: templateData.message,
        status: templateData.status,
      })
    }
  }, [templateData, reset])

  const isBusy = updateMutation.isPending || isSubmitting

  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const onSubmit = async (data: SmsTemplateFormData) => {
    if (!templateId) return
    try {
      await updateMutation.mutateAsync({
        id: templateId,
        data: {
          description: data.description,
          message: data.message,
          status: data.status as SmsTemplateStatus,
        },
      })
      showSuccess("SMS template updated successfully")
      navigate(backTo)
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
        const keywordOrder: Array<[string, keyof SmsTemplateFormData]> = [
          ["description", "description"],
          ["message", "message"],
          ["status", "status"],
          ["name", "name"],
        ]
        const hit = keywordOrder.find(([keyword]) => lower.includes(keyword))
        if (hit) {
          setError(hit[1], { type: "server", message: parsed.message })
        }
      }
      showError(error)
    }
  }

  if (isFetchingTemplate) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader backUrl={backTo} backLabel={backLabel} onBack={() => guard(() => navigate(backTo))} title="Edit SMS Template" description="Update the SMS template details and content." />
          <Card>
            <CardContent className="space-y-4 pt-6">
              <Skeleton className="h-5 w-40" />
              <div className="grid gap-4 md:grid-cols-2">
                {Array.from({ length: 3 }).map((_, i) => (
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

  if (!isFetchingTemplate && !templateData) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader backUrl={backTo} backLabel={backLabel} onBack={() => guard(() => navigate(backTo))} title="Edit SMS Template" description="Update the SMS template details and content." />
          <Card>
            <CardContent className="flex flex-col items-center justify-center gap-4 py-12 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <AlertCircle className="size-6" />
              </div>
              <div className="space-y-1">
                <h2 className="text-lg font-semibold">Template not found</h2>
                <p className="text-sm text-muted-foreground">
                  The SMS template you're trying to edit doesn't exist or may have been removed.
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
        <FormPageHeader
          backUrl={backTo}
          backLabel={backLabel}
          onBack={() => guard(() => navigate(backTo))}
          title="Edit SMS Template"
          description="Update the SMS template details, content, and status."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6" key={templateId}>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Template Information</CardTitle>
              <p className="text-sm text-muted-foreground">Template name, description, and activation status.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormInputField
                label="Name"
                placeholder="e.g., otp_verification"
                description="Template identifier — cannot be changed."
                disabled
                error={errors.name?.message}
                required
                {...register("name")}
              />
              <FormInputField
                label="Description"
                placeholder="Brief description of this template"
                disabled={isBusy}
                error={errors.description?.message}
                required
                {...register("description")}
              />
              <Controller name="status" control={control} render={({ field }) => (
                <FormSelectField
                  label="Status"
                  placeholder="Select status"
                  options={STATUS_OPTIONS}
                  value={field.value}
                  onValueChange={field.onChange}
                  disabled={isBusy}
                  error={errors.status?.message}
                  required
                />
              )} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Message Content</CardTitle>
              <p className="text-sm text-muted-foreground">The SMS body text sent to recipients.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormTextareaField
                label="Message"
                placeholder="Enter your SMS message here..."
                description="The SMS body text sent to recipients."
                rows={6}
                disabled={isBusy}
                error={errors.message?.message}
                required
                {...register("message")}
              />
            </CardContent>
          </Card>

          {templateData?.parametersDoc && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Template Parameters</CardTitle>
                <p className="text-sm text-muted-foreground">
                  Use these variables in the message content. They will be replaced with actual values when the SMS is sent.
                </p>
              </CardHeader>
              <CardContent>
                <div className="overflow-x-auto rounded-md border">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b bg-muted/50">
                        <th className="px-4 py-2 text-left font-medium">Parameter</th>
                        <th className="px-4 py-2 text-left font-medium">Description</th>
                      </tr>
                    </thead>
                    <tbody>
                      {templateData.parametersDoc
                        .split('\n')
                        .filter((line) => line.startsWith('|') && !line.includes('---'))
                        .slice(1)
                        .map((line, i) => {
                          const cells = line.split('|').map((c) => c.trim()).filter(Boolean)
                          return (
                            <tr key={i} className="border-b last:border-0">
                              <td className="px-4 py-2 font-mono text-xs">{cells[0]?.replace(/`/g, '')}</td>
                              <td className="px-4 py-2 text-muted-foreground">{cells[1]}</td>
                            </tr>
                          )
                        })}
                    </tbody>
                  </table>
                </div>
              </CardContent>
            </Card>
          )}

          <div className="flex justify-end gap-3">
            <Button type="button" variant="outline" onClick={() => guard(() => navigate(backTo))} disabled={isBusy}>
              Cancel
            </Button>
            <FormSubmitButton isSubmitting={isBusy} submitText="Update Template" />
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
