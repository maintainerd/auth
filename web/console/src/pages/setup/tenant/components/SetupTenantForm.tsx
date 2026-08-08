import { useForm } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { FieldGroup } from "@/components/ui/field"
import { FormInputField, FormSubmitButton } from "@/components/form"
import { setupTenantSchema, type SetupTenantFormData } from "@/lib/validations"
import { useSetupTenant } from "@/hooks/useSetup"

const SetupTenantForm = () => {
  const { isLoading, createTenantWithDefaults } = useSetupTenant()

  const {
    register,
    handleSubmit,
    formState: { errors }
  } = useForm<SetupTenantFormData>({
    resolver: yupResolver(setupTenantSchema),
    mode: 'onChange',
    defaultValues: {
      name: "",
      display_name: "",
    }
  })

  const onSubmit = async (data: SetupTenantFormData) => {
    await createTenantWithDefaults(data.name, data.display_name, "")
  }

  return (
    <div className="flex flex-col gap-8">
      <div className="flex flex-col items-center gap-2 text-center">
        <h1 className="text-2xl font-semibold tracking-tight">Create your tenant</h1>
        <p className="text-sm text-muted-foreground">
          Set up your organization to get started.
        </p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)}>
        <FieldGroup>
          <FormInputField
            label="Tenant name"
            placeholder="e.g. my-org-1"
            disabled={isLoading}
            error={errors.name?.message}
            required
            {...register("name")}
          />
          <FormInputField
            label="Display name"
            placeholder="e.g. My Organization"
            disabled={isLoading}
            error={errors.display_name?.message}
            required
            {...register("display_name")}
          />
          <FormSubmitButton
            isSubmitting={isLoading}
            submitText="Create tenant"
            submittingText="Creating tenant..."
            className="mt-1 w-full"
          />
        </FieldGroup>
      </form>
    </div>
  )
}

export default SetupTenantForm
