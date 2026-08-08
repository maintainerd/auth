import { useEffect, useState } from "react"
import { useForm, Controller } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { Key } from "lucide-react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { FormInputField, FormTextareaField, FormSelectField, type SelectOption } from "@/components/form"
import { FormSearchableSelectField, type SearchableSelectOption } from "@/components/inputs"
import { permissionWithApiSchema, type PermissionWithApiFormData } from "@/lib/validations"
import { useCreatePermission } from "@/hooks/usePermissions"
import { useApis } from "@/hooks/useApis"
import { useToast } from "@/hooks/useToast"

interface PermissionCreateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

const statusOptions: SelectOption[] = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
]

const EMPTY_VALUES: PermissionWithApiFormData = {
  name: "",
  description: "",
  status: "active",
  apiId: "",
}

/**
 * Create a permission from the permissions listing, where — unlike the API
 * details Permissions tab — there is no owning API in context. `api_id` is
 * Required on the server's PermissionCreateRequestDTO, so this form picks it
 * explicitly rather than offering a create button that could never submit.
 *
 * The API picker is server-searched (the list endpoint ILIKEs `display_name`)
 * so it does not degrade once a tenant has more APIs than one page.
 */
export function PermissionCreateDialog({
  open,
  onOpenChange,
  onSuccess,
}: PermissionCreateDialogProps) {
  const { showSuccess, showError } = useToast()
  const createPermissionMutation = useCreatePermission()
  const [apiSearch, setApiSearch] = useState("")

  const { data: apisData, isLoading: isFetchingApis } = useApis({
    display_name: apiSearch || undefined,
    page: 1,
    limit: 20,
    sort_by: "display_name",
    sort_order: "asc",
  })

  const apiOptions: SearchableSelectOption[] = (apisData?.rows ?? []).map((api) => ({
    value: api.api_id,
    label: api.display_name,
    description: api.name,
    keywords: api.name,
  }))

  const {
    register,
    handleSubmit,
    control,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<PermissionWithApiFormData>({
    resolver: yupResolver(permissionWithApiSchema),
    defaultValues: EMPTY_VALUES,
    mode: "onTouched",
    reValidateMode: "onChange",
  })

  // Reopening must not show the previous attempt's values or errors.
  useEffect(() => {
    if (open) {
      reset(EMPTY_VALUES)
      setApiSearch("")
    }
  }, [open, reset])

  const isLoading = createPermissionMutation.isPending || isSubmitting

  const onSubmit = async (data: PermissionWithApiFormData) => {
    try {
      await createPermissionMutation.mutateAsync({
        name: data.name,
        description: data.description,
        status: data.status,
        api_id: data.apiId,
      })
      showSuccess("Permission created successfully")
      onOpenChange(false)
      onSuccess?.()
    } catch (error) {
      showError(error)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Key className="h-5 w-5" />
            Add Permission
          </DialogTitle>
          <DialogDescription>
            Create a new permission and assign it to an API.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <Controller
            name="apiId"
            control={control}
            render={({ field }) => (
              <FormSearchableSelectField
                id="permission-api"
                label="API"
                placeholder={isFetchingApis ? "Loading APIs..." : "Select API"}
                emptyText="No API found."
                description="The API this permission belongs to."
                options={apiOptions}
                value={field.value}
                onValueChange={field.onChange}
                searchValue={apiSearch}
                onSearchChange={setApiSearch}
                loading={isFetchingApis}
                disabled={isLoading}
                error={errors.apiId?.message}
                required
              />
            )}
          />

          <FormInputField
            label="Permission Name"
            placeholder="e.g., users:read, posts:write"
            description="Use format: resource:action (lowercase, hyphens allowed)"
            disabled={isLoading}
            error={errors.name?.message}
            required
            {...register("name")}
          />

          <FormTextareaField
            label="Description"
            placeholder="Describe what this permission allows"
            disabled={isLoading}
            error={errors.description?.message}
            required
            {...register("description")}
          />

          <Controller
            name="status"
            control={control}
            render={({ field }) => (
              <FormSelectField
                label="Status"
                placeholder="Select status"
                options={statusOptions}
                value={field.value}
                onValueChange={field.onChange}
                disabled={isLoading}
                error={errors.status?.message}
                required
              />
            )}
          />

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={isLoading}>
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? "Saving..." : "Create Permission"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
