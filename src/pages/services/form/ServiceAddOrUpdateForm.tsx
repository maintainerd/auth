import { useEffect } from "react"
import { useParams, useNavigate, useLocation } from "react-router-dom"
import { useForm, Controller } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { ArrowLeft, AlertCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { DetailsContainer } from "@/components/container"
import { FormPageHeader } from "@/components/header"
import {
  FormInputField,
  FormTextareaField,
  FormSelectField,
  FormSubmitButton,
  type SelectOption
} from "@/components/form"
import { FormSlugField } from "@/components/inputs"
import { ConfirmationDialog } from "@/components/dialog"
import { serviceSchema, type ServiceFormData } from "@/lib/validations"
import { sanitizeName } from "@/lib/validations/regex"
import { useService, useCreateService, useUpdateService } from "@/hooks/useServices"
import { useToast } from "@/hooks/useToast"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"

// Status options for the select field
const STATUS_OPTIONS: SelectOption[] = [
  { value: "active", label: "Active" },
  { value: "maintenance", label: "Maintenance" },
  { value: "deprecated", label: "Deprecated" },
  { value: "inactive", label: "Inactive" },
]

// Backend snake_case field keys → form field names, for routing structured
// server validation errors onto the offending inputs.
const BACKEND_FIELD_MAP: Record<string, keyof ServiceFormData> = {
  name: "name",
  display_name: "displayName",
  description: "description",
  version: "version",
  status: "status",
}

export default function ServiceAddOrUpdateForm() {
  const { serviceId } = useParams<{ serviceId?: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError, parseError } = useToast()

  const isEditing = Boolean(serviceId)
  const isCreating = !isEditing

  // Honour where the user came from so the back button and post-submit
  // navigation return there. Falls back to the listing.
  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/services`
  const backLabel = navState?.backLabel ?? (backTo === `/services` ? "Back to Services" : "Back")

  // Fetch existing service if editing
  const { data: serviceData, isLoading: isFetchingService } = useService(serviceId || '')
  const createServiceMutation = useCreateService()
  const updateServiceMutation = useUpdateService()

  // React Hook Form setup
  const {
    register,
    handleSubmit,
    control,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty }
  } = useForm<ServiceFormData>({
    resolver: yupResolver(serviceSchema),
    defaultValues: {
      name: "",
      displayName: "",
      description: "",
      version: "v0.1.0",
      status: "active",
    },
    mode: 'onTouched',
    reValidateMode: 'onChange',
  })

  // Load existing service data if editing
  useEffect(() => {
    if (isEditing && serviceData) {
      reset({
        name: serviceData.name,
        displayName: serviceData.display_name,
        description: serviceData.description,
        version: serviceData.version,
        status: serviceData.status,
      })
    }
  }, [isEditing, serviceData, reset])

  const isLoading = createServiceMutation.isPending || updateServiceMutation.isPending || isSubmitting
  const existingService = serviceData

  // Warn before discarding unsaved edits (browser close/refresh + guarded exits).
  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const onSubmit = async (data: ServiceFormData) => {
    try {
      const requestData = {
        name: data.name,
        display_name: data.displayName,
        description: data.description,
        version: data.version,
        status: data.status,
      }

      if (isCreating) {
        await createServiceMutation.mutateAsync(requestData)
        showSuccess("Service created successfully")
      } else {
        await updateServiceMutation.mutateAsync({
          serviceId: serviceId!,
          data: requestData
        })
        showSuccess("Service updated successfully")
      }

      navigate(backTo)
    } catch (error) {
      // Route backend errors onto the offending field where we can: structured
      // field errors first, otherwise keyword-match the message. Anything
      // unmapped still shows via the toast.
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
        // Most specific first, so "display name" doesn't land on "name".
        const keywordOrder: Array<[string, keyof ServiceFormData]> = [
          ["display name", "displayName"],
          ["display_name", "displayName"],
          ["description", "description"],
          ["version", "version"],
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

  const pageTitle = isCreating ? "Create Service" : `Edit ${existingService?.display_name || "Service"}`
  const submitButtonText = isCreating ? "Create Service" : "Update Service"

  // Show loading state while fetching service data
  if (isEditing && isFetchingService) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Service"
            description="Update service settings and configuration"
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

  // Show error if service not found
  if (isEditing && !isFetchingService && !serviceData) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Service"
            description="Update service settings and configuration"
          />
          <Card>
            <CardContent className="flex flex-col items-center justify-center gap-4 py-12 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <AlertCircle className="size-6" />
              </div>
              <div className="space-y-1">
                <h2 className="text-lg font-semibold">Service not found</h2>
                <p className="text-sm text-muted-foreground">
                  The service you're looking for doesn't exist or may have been removed.
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
          title={pageTitle}
          description={
            isCreating
              ? "Create a new microservice to manage APIs, permissions, and policies"
              : "Update service settings and configuration"
          }
          showSystemBadge={existingService?.is_system}
          showWarning={existingService?.is_system}
          warningMessage="This is a system service. Some settings may be restricted to prevent system instability."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6" key={serviceId || "create"}>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Basic Information</CardTitle>
              <p className="text-sm text-muted-foreground">
                The service name, status, and description.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <FormSlugField
                  label="Service Name"
                  placeholder="e.g., core, auth, payment"
                  description="Used for communications and events (lowercase, numbers, hyphens only)"
                  sanitize={sanitizeName}
                  disabled={existingService?.is_system || isLoading}
                  error={errors.name?.message}
                  required
                  {...register("name")}
                />

                <FormInputField
                  label="Display Name"
                  placeholder="e.g., Core Service, Authentication Service"
                  description="Human-readable name shown in the interface"
                  disabled={isLoading}
                  error={errors.displayName?.message}
                  required
                  {...register("displayName")}
                />
              </div>

              {/* No `required`: serviceSchema only bounds the length (max 255) and
                  defaults to "", matching the backend, which validates Length(0, 255)
                  with no Required rule. The asterisk promised a validation error the
                  form never raises and the server never returns. */}
              <FormTextareaField
                label="Description"
                placeholder="Enter service description"
                rows={3}
                disabled={isLoading}
                error={errors.description?.message}
                {...register("description")}
              />

              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField
                  label="Version"
                  placeholder="e.g., v0.1.0, v1.0.0"
                  description="Service version number"
                  disabled={isLoading}
                  error={errors.version?.message}
                  required
                  {...register("version")}
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
                      disabled={isLoading}
                      error={errors.status?.message}
                      required
                    />
                  )}
                />
              </div>
            </CardContent>
          </Card>

          <div className="flex justify-end gap-3">
            <Button
              type="button"
              variant="outline"
              onClick={() => guard(() => navigate(backTo))}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <FormSubmitButton
              isSubmitting={isLoading}
              submitText={submitButtonText}
              disabled={existingService?.is_system && isEditing}
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
