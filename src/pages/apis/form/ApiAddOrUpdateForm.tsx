import { useMemo, useEffect, useState } from "react"
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
import { FormSlugField, FormSearchableSelectField, type SearchableSelectOption } from "@/components/inputs"
import { ConfirmationDialog } from "@/components/dialog"
import { apiSchema, type ApiFormData } from "@/lib/validations"
import { sanitizeName } from "@/lib/validations/regex"
import { useApi, useCreateApi, useUpdateApi } from "@/hooks/useApis"
import { useServices } from "@/hooks/useServices"
import { useToast } from "@/hooks/useToast"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"

// Status options for the select field
const STATUS_OPTIONS: SelectOption[] = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
]

// Backend snake_case field keys → form field names, for routing structured
// server validation errors onto the offending inputs.
const BACKEND_FIELD_MAP: Record<string, keyof ApiFormData> = {
  name: "name",
  display_name: "displayName",
  description: "description",
  status: "status",
  service_id: "serviceId",
}

export default function ApiAddOrUpdateForm() {
  const { apiId } = useParams<{ apiId?: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError, parseError } = useToast()
  const isEditing = !!apiId
  const isCreating = !isEditing

  // Honour where the user came from (e.g. the details page) so the back button,
  // Cancel, and post-submit navigation return there. Falls back to the listing.
  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/apis`
  const backLabel = navState?.backLabel ?? (backTo === `/apis` ? "Back to APIs" : "Back")

  // Fetch existing API if editing
  const { data: apiData, isLoading: isFetchingApi } = useApi(apiId || '')
  const createApiMutation = useCreateApi()
  const updateApiMutation = useUpdateApi()

  // Services for the picker. The previous single `limit: 100` fetch made every
  // service past the hundredth unselectable, and Service is required — so those
  // APIs simply could not be created. The list endpoint ILIKEs `display_name`
  // (repository_service.go FindPaginated), so the term is pushed to the server
  // and only a page of matches is held in memory.
  const [serviceSearch, setServiceSearch] = useState("")

  const { data: servicesData, isLoading: isFetchingServices } = useServices({
    display_name: serviceSearch || undefined,
    page: 1,
    limit: 20,
    sort_by: 'display_name',
    sort_order: 'asc'
  })

  // Convert services to select options. The API's own service is merged in when
  // the current page doesn't contain it, so opening the edit form on service
  // #500 still renders its name instead of an empty trigger.
  const serviceOptions: SearchableSelectOption[] = useMemo(() => {
    const options = (servicesData?.rows ?? []).map(service => ({
      value: service.service_id,
      label: service.display_name,
      keywords: service.name,
    }))

    const current = apiData?.service
    if (current && !options.some(option => option.value === current.service_id)) {
      options.unshift({
        value: current.service_id,
        label: current.display_name,
        keywords: current.name,
      })
    }

    return options
  }, [servicesData, apiData])

  // React Hook Form setup
  const {
    register,
    handleSubmit,
    control,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty }
  } = useForm<ApiFormData>({
    resolver: yupResolver(apiSchema),
    defaultValues: {
      name: "",
      displayName: "",
      description: "",
      status: "active",
      serviceId: "",
    },
    mode: 'onTouched',
    reValidateMode: 'onChange',
  })

  // Reset form once the API loads. This no longer waits on the services page:
  // the picker is server-searched, so the owning service may legitimately never
  // appear in it — serviceOptions merges it in instead.
  useEffect(() => {
    if (!isEditing) return
    if (!apiData) return

    reset({
      name: apiData.name,
      displayName: apiData.display_name,
      description: apiData.description,
      status: apiData.status,
      serviceId: apiData.service?.service_id || "",
    })
  }, [isEditing, apiData, reset])

  const isLoading = createApiMutation.isPending || updateApiMutation.isPending || isSubmitting
  const existingApi = apiData

  // Warn before discarding unsaved edits (browser close/refresh + guarded exits).
  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const onSubmit = async (data: ApiFormData) => {
    try {
      const requestData = {
        name: data.name,
        display_name: data.displayName,
        description: data.description,
        status: data.status,
        service_id: data.serviceId,
      }

      if (isCreating) {
        await createApiMutation.mutateAsync(requestData)
        showSuccess("API created successfully")
      } else {
        await updateApiMutation.mutateAsync({
          apiId: apiId!,
          data: requestData
        })
        showSuccess("API updated successfully")
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
        const keywordOrder: Array<[string, keyof ApiFormData]> = [
          ["display name", "displayName"],
          ["display_name", "displayName"],
          ["description", "description"],
          ["service", "serviceId"],
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

  const pageTitle = isCreating ? "Create API" : `Edit ${existingApi?.display_name || "API"}`
  const pageDescription = isCreating
    ? "Create a new API to manage permissions and access control"
    : "Update API settings and configuration"
  const submitButtonText = isCreating ? "Create API" : "Update API"

  // Loading state while fetching the API to edit
  if (isEditing && isFetchingApi) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit API"
            description={pageDescription}
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
  if (isEditing && !isFetchingApi && !apiData) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit API"
            description={pageDescription}
          />
          <Card>
            <CardContent className="flex flex-col items-center justify-center gap-4 py-12 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <AlertCircle className="size-6" />
              </div>
              <div className="space-y-1">
                <h2 className="text-lg font-semibold">API not found</h2>
                <p className="text-sm text-muted-foreground">
                  The API you're trying to edit doesn't exist or may have been removed.
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
          description={pageDescription}
          showSystemBadge={existingApi?.is_system}
          showWarning={existingApi?.is_system}
          warningMessage="This is a system API. Some settings may be restricted to prevent system instability."
        />

        {/* Form */}
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6" key={apiId || 'create'}>
          {/* Basic Information */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Basic Information</CardTitle>
              <p className="text-sm text-muted-foreground">
                The API name, owning service, status, and description.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <FormSlugField
                  label="API Name"
                  placeholder="e.g., auth, users, payments"
                  description="Used for communications and events (lowercase, numbers, hyphens only)"
                  sanitize={sanitizeName}
                  disabled={existingApi?.is_system || isLoading}
                  error={errors.name?.message}
                  required
                  {...register("name")}
                />

                <FormInputField
                  label="Display Name"
                  placeholder="e.g., Auth API, Users API"
                  description="Human-readable name shown in the interface"
                  disabled={isLoading}
                  error={errors.displayName?.message}
                  required
                  {...register("displayName")}
                />
              </div>

              {/* Optional: the server's APICreateRequestDTO has no Required rule
                  on description, so the asterisk was blocking a submission the
                  API would have accepted. */}
              <FormTextareaField
                label="Description"
                placeholder="Enter API description"
                rows={3}
                disabled={isLoading}
                error={errors.description?.message}
                {...register("description")}
              />

              <div className="grid gap-4 md:grid-cols-2">
                <Controller
                  name="serviceId"
                  control={control}
                  render={({ field }) => (
                    <FormSearchableSelectField
                      id="service"
                      label="Service"
                      placeholder={isFetchingServices ? "Loading services..." : "Select service"}
                      emptyText="No service found."
                      options={serviceOptions}
                      value={field.value}
                      onValueChange={field.onChange}
                      searchValue={serviceSearch}
                      onSearchChange={setServiceSearch}
                      loading={isFetchingServices}
                      disabled={isLoading || (existingApi?.is_system && isEditing)}
                      error={errors.serviceId?.message}
                      required
                    />
                  )}
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

          {/* Actions */}
          <div className="flex items-center justify-end gap-3">
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
              disabled={existingApi?.is_system && isEditing}
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
