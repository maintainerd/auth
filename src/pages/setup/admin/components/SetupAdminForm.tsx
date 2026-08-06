import { useForm } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { FieldGroup } from "@/components/ui/field"
import { FormPasswordField, FormSubmitButton } from "@/components/form"
import { FormEmailField } from "@/components/inputs"
import { setupAdminSchema, type SetupAdminFormData } from "@/lib/validations"
import { useSetupAdmin } from "@/hooks/useSetup"

const SetupAdminForm = () => {
  const { isLoading, createAdminAccount } = useSetupAdmin()

  const {
    register,
    handleSubmit,
    formState: { errors }
  } = useForm<SetupAdminFormData>({
    resolver: yupResolver(setupAdminSchema),
    mode: 'onChange',
    defaultValues: {
      email: "",
      password: "",
      confirmPassword: ""
    }
  })

  const onSubmit = async (data: SetupAdminFormData) => {
    await createAdminAccount({
      email: data.email,
      password: data.password
    })
  }

  return (
    <div className="flex flex-col gap-8">
      <div className="flex flex-col items-center gap-2 text-center">
        <h1 className="text-2xl font-semibold tracking-tight">Create admin account</h1>
        <p className="text-sm text-muted-foreground">
          Set up your administrator account to complete the setup.
        </p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)}>
        <FieldGroup>
          <FormEmailField
            label="Email"
            placeholder="admin@acme.com"
            disabled={isLoading}
            error={errors.email?.message}
            required
            {...register("email")}
          />
          <FormPasswordField
            label="Password"
            placeholder="Enter a strong password"
            autoComplete="new-password"
            disabled={isLoading}
            error={errors.password?.message}
            required
            {...register("password")}
          />
          <FormPasswordField
            label="Confirm password"
            placeholder="Confirm your password"
            autoComplete="new-password"
            disabled={isLoading}
            error={errors.confirmPassword?.message}
            required
            {...register("confirmPassword")}
          />
          <FormSubmitButton
            isSubmitting={isLoading}
            submitText="Complete setup"
            submittingText="Creating admin..."
            className="mt-1 w-full"
          />
        </FieldGroup>
      </form>
    </div>
  )
}

export default SetupAdminForm
