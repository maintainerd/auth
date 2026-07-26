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
import { TemplateParametersCard } from "@/pages/branding/components/TemplateParametersCard"
import { ConfirmationDialog } from "@/components/dialog"
import { useEmailTemplate, useUpdateEmailTemplate } from "@/hooks/useEmailTemplates"
import { useToast } from "@/hooks/useToast"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import type { EmailTemplateStatus } from "@/services/api/email-templates/types"

const emailTemplateSchema = yup.object({
  name: yup.string().required('Name is required').min(3, 'Name must be at least 3 characters'),
  subject: yup.string().required('Subject is required').min(3, 'Subject must be at least 3 characters'),
  body_html: yup.string().required('HTML content is required'),
  body_plain: yup.string().required('Plain text content is required'),
  status: yup.string().oneOf(['active', 'inactive']).required('Status is required'),
})

type EmailTemplateFormData = yup.InferType<typeof emailTemplateSchema>

const STATUS_OPTIONS: SelectOption[] = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
]

// Backend snake_case field keys → form field names.
const BACKEND_FIELD_MAP: Record<string, keyof EmailTemplateFormData> = {
  name: "name",
  subject: "subject",
  body_html: "body_html",
  body_plain: "body_plain",
  status: "status",
}

export default function EmailTemplateForm() {
  const { templateId } = useParams<{ templateId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError, parseError } = useToast()

  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/branding?tab=email-templates`
  const backLabel = navState?.backLabel ?? (backTo.includes("email-templates") ? "Back to Email Templates" : "Back")

  const { data: templateData, isLoading: isFetchingTemplate } = useEmailTemplate(templateId || '')
  const updateMutation = useUpdateEmailTemplate()

  const {
    register,
    handleSubmit,
    control,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<EmailTemplateFormData>({
    resolver: yupResolver(emailTemplateSchema),
    defaultValues: {
      name: "",
      subject: "",
      body_html: "",
      body_plain: "",
      status: "active",
    },
    mode: "onTouched",
    reValidateMode: "onChange",
  })

  const initialized = useRef(false)
  useEffect(() => {
    if (templateData && !initialized.current) {
      initialized.current = true
      reset({
        name: templateData.name,
        subject: templateData.subject,
        body_html: templateData.bodyHtml,
        body_plain: templateData.bodyPlain,
        status: templateData.status,
      })
    }
  }, [templateData, reset])

  const isBusy = updateMutation.isPending || isSubmitting

  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const onSubmit = async (data: EmailTemplateFormData) => {
    if (!templateId) return
    try {
      await updateMutation.mutateAsync({
        id: templateId,
        data: {
          subject: data.subject,
          body_html: data.body_html,
          body_plain: data.body_plain,
          status: data.status as EmailTemplateStatus,
        },
      })
      showSuccess("Email template updated successfully")
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
        const keywordOrder: Array<[string, keyof EmailTemplateFormData]> = [
          ["subject", "subject"],
          ["body_html", "body_html"],
          ["body_plain", "body_plain"],
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
          <FormPageHeader backUrl={backTo} backLabel={backLabel} onBack={() => guard(() => navigate(backTo))} title="Edit Email Template" description="Update this existing email template." />
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

  if (!isFetchingTemplate && !templateData) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader backUrl={backTo} backLabel={backLabel} onBack={() => guard(() => navigate(backTo))} title="Edit Email Template" description="Update this existing email template." />
          <Card>
            <CardContent className="flex flex-col items-center justify-center gap-4 py-12 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <AlertCircle className="size-6" />
              </div>
              <div className="space-y-1">
                <h2 className="text-lg font-semibold">Template not found</h2>
                <p className="text-sm text-muted-foreground">
                  The email template you're trying to edit doesn't exist or may have been removed.
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
          title="Edit Email Template"
          description="Update the content and status for this existing email template."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6" key={templateId}>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Template Information</CardTitle>
              <p className="text-sm text-muted-foreground">Template name, subject, and activation status.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormInputField
                label="Name"
                placeholder="e.g., password_reset, welcome_email"
                description="Template identifier — cannot be changed."
                disabled
                error={errors.name?.message}
                required
                {...register("name")}
              />

              <FormInputField
                label="Subject"
                placeholder="Enter email subject line"
                description="The subject line of the email."
                disabled={isBusy}
                error={errors.subject?.message}
                required
                {...register("subject")}
              />

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
                    disabled={isBusy}
                    error={errors.status?.message}
                    required
                  />
                )}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Email Content</CardTitle>
              <p className="text-sm text-muted-foreground">HTML and plain-text body content for this template.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormTextareaField
                label="HTML Content"
                placeholder="Enter HTML email template..."
                description="The HTML version of the email body."
                rows={12}
                disabled={isBusy}
                error={errors.body_html?.message}
                required
                {...register("body_html")}
              />

              <FormTextareaField
                label="Plain Text Content"
                placeholder="Enter plain text email template (fallback)..."
                description="The plain text fallback for email clients that don't support HTML."
                rows={8}
                disabled={isBusy}
                error={errors.body_plain?.message}
                required
                {...register("body_plain")}
              />
            </CardContent>
          </Card>

          <TemplateParametersCard
            parametersDoc={templateData?.parametersDoc}
            description="Use these variables in the HTML and plain text content. They will be replaced with actual values when the email is sent."
          />

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
