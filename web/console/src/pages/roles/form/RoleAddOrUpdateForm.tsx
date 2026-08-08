import { useEffect } from "react"
import { useParams, useNavigate, useLocation } from "react-router-dom"
import { useForm, Controller, type Resolver } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { ArrowLeft, AlertCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { DetailsContainer } from "@/components/container"
import { FormPageHeader } from "@/components/header"
import {
  FormTextareaField,
  FormSelectField,
  FormSubmitButton,
  type SelectOption
} from "@/components/form"
import { FormSlugField } from "@/components/inputs"
import { roleSchema, type RoleFormData } from "@/lib/validations"
import { useRole, useCreateRole, useUpdateRole } from "@/hooks/useRoles"
import { useToast } from "@/hooks/useToast"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import { ConfirmationDialog } from "@/components/dialog"

const STATUS_OPTIONS: SelectOption[] = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
]

export default function RoleAddOrUpdateForm() {
  const { roleId } = useParams<{ roleId?: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError, parseError } = useToast()

  const isEditing = Boolean(roleId)
  const isCreating = !isEditing

  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/roles`
  const backLabel = navState?.backLabel ?? (backTo === `/roles` ? "Back to Roles" : "Back")

  const { data: roleData, isLoading: isFetchingRole } = useRole(roleId || "")
  const createRoleMutation = useCreateRole()
  const updateRoleMutation = useUpdateRole()

  const {
    register,
    handleSubmit,
    control,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<RoleFormData>({
    resolver: yupResolver(roleSchema) as Resolver<RoleFormData>,
    defaultValues: { name: "", description: "", status: "active" },
    mode: "onTouched",
    reValidateMode: "onChange",
  })

  useEffect(() => {
    if (isEditing && roleData) {
      reset({ name: roleData.name, description: roleData.description, status: roleData.status })
    }
  }, [isEditing, roleData, reset])

  const isLoading = createRoleMutation.isPending || updateRoleMutation.isPending || isSubmitting
  const existingRole = roleData

  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const onSubmit = async (data: RoleFormData) => {
    try {
      const requestData = { name: data.name, description: data.description ?? "", status: data.status }
      if (isCreating) {
        await createRoleMutation.mutateAsync(requestData)
        showSuccess("Role created successfully")
      } else {
        await updateRoleMutation.mutateAsync({ roleId: roleId!, data: requestData })
        showSuccess("Role updated successfully")
      }
      navigate(backTo)
    } catch (error) {
      const parsed = parseError(error)
      const known = ["name", "description", "status"] as const
      let mappedToField = false
      if (parsed.fieldErrors) {
        for (const [field, message] of Object.entries(parsed.fieldErrors)) {
          if ((known as readonly string[]).includes(field)) {
            setError(field as (typeof known)[number], { type: "server", message })
            mappedToField = true
          }
        }
      }
      if (!mappedToField) {
        const lower = parsed.message.toLowerCase()
        const field = known.find((f) => lower.includes(f))
        if (field) {
          setError(field, { type: "server", message: parsed.message })
        }
      }
      showError(error)
    }
  }

  const pageTitle = isCreating ? "Create Role" : `Edit ${existingRole?.name || "Role"}`
  const submitButtonText = isCreating ? "Create Role" : "Update Role"

  if (isEditing && isFetchingRole) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Role"
            description="Update role settings and configuration"
          />
          <Card>
            <CardContent className="space-y-4 pt-6">
              <Skeleton className="h-5 w-40" />
              <div className="grid gap-4 md:grid-cols-2">
                {Array.from({ length: 2 }).map((_, i) => (
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

  if (isEditing && !isFetchingRole && !roleData) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Role"
            description="Update role settings and configuration"
          />
          <Card>
            <CardContent className="flex flex-col items-center justify-center gap-4 py-12 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <AlertCircle className="size-6" />
              </div>
              <div className="space-y-1">
                <h2 className="text-lg font-semibold">Role not found</h2>
                <p className="text-sm text-muted-foreground">
                  The role you're looking for doesn't exist or may have been removed.
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
              ? "Create a new role to manage user permissions and access control"
              : "Update role settings and configuration"
          }
          showSystemBadge={existingRole?.is_system}
          showWarning={existingRole?.is_system}
          warningMessage="This is a system role. Some settings may be restricted to prevent system instability."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6" key={roleId || "create"}>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Basic Information</CardTitle>
              <p className="text-sm text-muted-foreground">
                The role name, status, and description.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <FormSlugField
                  label="Name"
                  placeholder="e.g., admin, developer, viewer"
                  description="Unique identifier for the role (lowercase, numbers, hyphens, and colons only)"
                  disabled={isLoading || existingRole?.is_system}
                  error={errors.name?.message}
                  required
                  {...register("name")}
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

              <FormTextareaField
                label="Description"
                placeholder="Describe the role's purpose and permissions"
                rows={4}
                disabled={isLoading}
                error={errors.description?.message}
                {...register("description")}
              />
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
              disabled={existingRole?.is_system && isEditing}
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
