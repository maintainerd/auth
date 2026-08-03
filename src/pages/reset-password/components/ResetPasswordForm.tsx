import { useEffect, useMemo, useState } from "react"
import { useForm } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { useNavigate, useSearchParams } from "react-router-dom"
import { FormSubmitButton, FormPasswordField, PasswordRequirements } from "@/components/form"
import { FieldGroup } from "@/components/ui/field"
import { buildResetPasswordSchema, type ResetPasswordFormData } from "@/lib/validations"
import { useAuth } from "@/hooks/useAuth"
import { useTenant } from "@/hooks/useTenant"
import { useToast } from "@/hooks/useToast"
import { AlertCircle, CheckCircle2 } from "lucide-react"
import { Button } from "@/components/ui/button"

const ResetPasswordForm = () => {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { resetPassword } = useAuth()
  const { getCurrentTenant } = useTenant()
  const { showSuccess, showError, parseError } = useToast()
  const [resetPasswordError, setResetPasswordError] = useState<string | null>(null)
  const [passwordReset, setPasswordReset] = useState(false)
  const [passwordTyped, setPasswordTyped] = useState(false)

  // Get query params
  const clientId = searchParams.get('client_id')
  const tenantId = searchParams.get('tenant_id')
  const expires = searchParams.get('expires')
  const sig = searchParams.get('sig')
  const token = searchParams.get('token')

  // Check if all required query params are present
  const hasValidParams = Boolean((clientId ? 1 : 0) + (tenantId ? 1 : 0) === 1 && expires && sig && token)
  const passwordConfig = getCurrentTenant()?.password_config
  const resetSchema = useMemo(() => buildResetPasswordSchema(passwordConfig), [passwordConfig])

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting }
  } = useForm<ResetPasswordFormData>({
    resolver: yupResolver(resetSchema),
    defaultValues: {
      password: "",
      confirmPassword: ""
    },
    mode: 'onSubmit',
    reValidateMode: 'onSubmit'
  })

  const passwordValue = watch("password") || ""

  useEffect(() => {
    if (passwordValue.length > 0) setPasswordTyped(true)
  }, [passwordValue])

  const onSubmit = async (data: ResetPasswordFormData) => {
    if (!hasValidParams) {
      setResetPasswordError("Invalid or expired reset link")
      return
    }

    setResetPasswordError(null)
    try {
      await resetPassword({
        password: { new_password: data.password },
        queryParams: {
          client_id: clientId || undefined,
          tenant_id: tenantId || undefined,
          expires: expires!,
          sig: sig!,
          token: token!
        }
      })
      setPasswordReset(true)
      showSuccess("Password has been reset successfully")
    } catch (error: unknown) {
      const parsedError = parseError(error)
      const errorMessage = parsedError.message || "Failed to reset password. Please try again."
      setResetPasswordError(errorMessage)
      showError(errorMessage)
    }
  }

  if (!hasValidParams) {
    return (
      <div className="flex flex-col gap-8 text-center">
        <div className="flex flex-col items-center gap-3">
          <div className="flex size-14 items-center justify-center rounded-full bg-destructive/10">
            <AlertCircle className="size-7 text-destructive" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Invalid reset link</h1>
          <p className="max-w-xs text-sm text-muted-foreground">
            This password reset link is invalid or has expired. Please request a new one.
          </p>
        </div>

        <div className="flex flex-col gap-2">
          <Button
            type="button"
            onClick={() => navigate('/forgot-password')}
            className="w-full"
          >
            Request new reset link
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={() => navigate('/login')}
            className="w-full"
          >
            Back to login
          </Button>
        </div>
      </div>
    )
  }

  if (passwordReset) {
    return (
      <div className="flex flex-col gap-8 text-center">
        <div className="flex flex-col items-center gap-3">
          <div className="flex size-14 items-center justify-center rounded-full bg-emerald-500/10">
            <CheckCircle2 className="size-7 text-emerald-600" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Password reset</h1>
          <p className="max-w-xs text-sm text-muted-foreground">
            You can now sign in with your new password.
          </p>
        </div>

        <Button
          type="button"
          onClick={() => navigate('/login')}
          className="w-full"
        >
          Go to login
        </Button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-8">
      <div className="flex flex-col items-center gap-2 text-center">
        <h1 className="text-2xl font-semibold tracking-tight">Reset your password</h1>
        <p className="text-sm text-muted-foreground">
          Enter your new password below.
        </p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)}>
        <FieldGroup>
          {resetPasswordError && (
            <div
              role="alert"
              className="flex items-start gap-2.5 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive"
            >
              <AlertCircle className="mt-0.5 size-4 shrink-0" />
              <span>{resetPasswordError}</span>
            </div>
          )}
          <FormPasswordField
            label="New password"
            placeholder="Enter your new password"
            autoComplete="new-password"
            disabled={isSubmitting}
            error={errors.password?.message}
            required
            footer={
              passwordTyped ? (
                <PasswordRequirements password={passwordValue} config={passwordConfig} />
              ) : undefined
            }
            {...register("password")}
          />
          <FormPasswordField
            label="Confirm password"
            placeholder="Confirm your new password"
            autoComplete="new-password"
            disabled={isSubmitting}
            error={errors.confirmPassword?.message}
            required
            {...register("confirmPassword")}
          />
          <FormSubmitButton
            isSubmitting={isSubmitting}
            submitText="Reset password"
            submittingText="Resetting..."
            className="mt-1 w-full"
          />
        </FieldGroup>
      </form>
    </div>
  )
}

export default ResetPasswordForm
