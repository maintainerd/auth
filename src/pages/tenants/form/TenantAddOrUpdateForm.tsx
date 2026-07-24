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
  type SelectOption,
} from "@/components/form"
import { FormSlugField } from "@/components/inputs"
import { ConfirmationDialog } from "@/components/dialog"
import { createTenantSchema, type CreateTenantFormData } from "@/lib/validations"
import { sanitizeName } from "@/lib/validations/regex"
import { useTenantById, useCreateTenant, useUpdateTenant } from "@/hooks/useTenants"
import { useToast } from "@/hooks/useToast"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import type { TenantStatus } from "@/services/api/tenants/types"

const STATUS_OPTIONS: SelectOption[] = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
  { value: "suspended", label: "Suspended" },
]

// Backend snake_case field keys → form field names.
const BACKEND_FIELD_MAP: Record<string, keyof CreateTenantFormData> = {
  name: "name",
  display_name: "display_name",
  description: "description",
  status: "status",
}

export default function TenantAddOrUpdateForm() {
  const { id } = useParams<{ id?: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError, parseError } = useToast()

  const isEditing = Boolean(id)
  const isCreating = !isEditing

  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/tenants`
  const backLabel =
    navState?.backLabel ?? (backTo === `/tenants` ? "Back to Tenants" : "Back")

  const { data: tenantData, isLoading: isFetchingTenant } = useTenantById(id)
  const createTenantMutation = useCreateTenant()
  const updateTenantMutation = useUpdateTenant()

  const {
    register,
    handleSubmit,
    control,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<CreateTenantFormData>({
    resolver: yupResolver(createTenantSchema),
    defaultValues: {
      name: "",
      display_name: "",
      description: "",
      status: "active",
    },
    mode: "onTouched",
    reValidateMode: "onChange",
  })

  useEffect(() => {
    if (isEditing && tenantData) {
      reset({
        name: tenantData.name,
        display_name: tenantData.display_name,
        description: tenantData.description,
        status: tenantData.status,
      })
    }
  }, [isEditing, tenantData, reset])

  const isLoading = createTenantMutation.isPending || updateTenantMutation.isPending || isSubmitting
  const existingTenant = tenantData

  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const onSubmit = async (data: CreateTenantFormData) => {
    try {
      const requestData = {
        name: data.name,
        display_name: data.display_name,
        description: data.description,
        status: data.status as TenantStatus,
      }

      if (isCreating) {
        await createTenantMutation.mutateAsync(requestData)
        showSuccess("Tenant created successfully")
      } else {
        await updateTenantMutation.mutateAsync({
          tenantId: id!,
          data: requestData,
        })
        showSuccess("Tenant updated successfully")
      }

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
        const keywordOrder: Array<[string, keyof CreateTenantFormData]> = [
          ["display name", "display_name"],
          ["display_name", "display_name"],
          ["description", "description"],
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

  const pageTitle = isCreating ? "Create Tenant" : `Edit ${existingTenant?.display_name || "Tenant"}`
  const submitButtonText = isCreating ? "Create Tenant" : "Update Tenant"

  if (isEditing && isFetchingTenant) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Tenant"
            description="Update tenant organization settings and configuration"
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

  if (isEditing && !isFetchingTenant && !tenantData) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Tenant"
            description="Update tenant organization settings and configuration"
          />
          <Card>
            <CardContent className="flex flex-col items-center justify-center gap-4 py-12 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <AlertCircle className="size-6" />
              </div>
              <div className="space-y-1">
                <h2 className="text-lg font-semibold">Tenant not found</h2>
                <p className="text-sm text-muted-foreground">
                  The tenant you're looking for doesn't exist or may have been removed.
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
              ? "Create a new tenant organization to manage users, roles, and authentication"
              : "Update tenant organization settings and configuration"
          }
          showSystemBadge={existingTenant?.is_system}
          showWarning={existingTenant?.is_system}
          warningMessage="This is a system tenant. Some settings may be restricted to prevent system instability."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6" key={id || "create"}>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Basic Information</CardTitle>
              <p className="text-sm text-muted-foreground">
                The tenant name, identifier, and description.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <FormSlugField
                  label="Tenant Name"
                  placeholder="e.g., acme-corp"
                  description="Unique identifier (lowercase letters, numbers, and hyphens only)"
                  sanitize={sanitizeName}
                  disabled={existingTenant?.is_system || isLoading}
                  error={errors.name?.message}
                  required
                  {...register("name")}
                />

                <FormInputField
                  label="Display Name"
                  placeholder="e.g., Acme Corporation"
                  description="Human-readable name shown in the interface"
                  disabled={isLoading}
                  error={errors.display_name?.message}
                  required
                  {...register("display_name")}
                />
              </div>

              <FormTextareaField
                label="Description"
                placeholder="Enter tenant description"
                rows={3}
                disabled={isLoading}
                error={errors.description?.message}
                required
                {...register("description")}
              />

              <div className="grid gap-4 md:grid-cols-2">
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
              disabled={existingTenant?.is_system && isEditing}
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
