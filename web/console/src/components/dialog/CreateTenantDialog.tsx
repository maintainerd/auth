import { useEffect } from "react"
import { useForm, Controller } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { FieldGroup } from "@/components/ui/field"
import { FormInputField, FormTextareaField, FormSelectField } from "@/components/form"
import { createTenantSchema, type CreateTenantFormData } from "@/lib/validations"
import { useCreateTenant } from "@/hooks/useTenants"
import type { TenantStatus } from "@/services/api/tenants/types"

interface CreateTenantDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function CreateTenantDialog({
  open,
  onOpenChange,
}: CreateTenantDialogProps) {
  const createTenantMutation = useCreateTenant()

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
    control,
  } = useForm<CreateTenantFormData>({
    resolver: yupResolver(createTenantSchema),
    defaultValues: {
      name: "",
      display_name: "",
      description: "",
      status: "active" as TenantStatus,
    },
  })

  // Reset form when dialog closes
  useEffect(() => {
    if (!open) {
      reset()
    }
  }, [open, reset])

  const onSubmit = async (data: CreateTenantFormData) => {
    try {
      await createTenantMutation.mutateAsync({
        ...data,
        status: data.status as TenantStatus,
      })
      onOpenChange(false)
    } catch {
      // Error is handled by the mutation hook
    }
  }

  const isLoading = createTenantMutation.isPending

  const statusOptions = [
    { value: "active", label: "Active" },
    { value: "inactive", label: "Inactive" },
    { value: "suspended", label: "Suspended" },
  ]

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create New Tenant</DialogTitle>
          <DialogDescription>
            Enter the details below to create a new tenant
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)}>
          <FieldGroup>
            <FormInputField
              label="Tenant Name"
              placeholder="e.g. tenant-1"
              disabled={isLoading}
              error={errors.name?.message}
              required
              {...register("name")}
            />

            <FormInputField
              label="Display Name"
              placeholder="e.g. Test Organization"
              disabled={isLoading}
              error={errors.display_name?.message}
              required
              {...register("display_name")}
            />

            <FormTextareaField
              label="Description"
              placeholder="Brief description of your tenant"
              rows={3}
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
                  options={statusOptions}
                  value={field.value}
                  onValueChange={field.onChange}
                  disabled={isLoading}
                  error={errors.status?.message}
                  required
                />
              )}
            />

          </FieldGroup>

          <DialogFooter className="mt-6">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? "Creating..." : "Create Tenant"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
