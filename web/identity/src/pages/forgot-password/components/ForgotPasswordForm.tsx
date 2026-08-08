import { useState } from "react"
import { useForm } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { FormSubmitButton } from "@/components/form"
import { FormEmailField } from "@/components/inputs"
import { FieldGroup } from "@/components/ui/field"
import { Link } from "react-router-dom"
import { forgotPasswordSchema, type ForgotPasswordFormData } from "@/lib/validations"
import { useAuth } from "@/hooks/useAuth"
import { useToast } from "@/hooks/useToast"
import { AlertCircle, CheckCircle2 } from "lucide-react"
import { Button } from "@/components/ui/button"

const ForgotPasswordForm = () => {
  const { forgotPassword } = useAuth()
  const { showSuccess, showError, parseError } = useToast()
  const [forgotPasswordError, setForgotPasswordError] = useState<string | null>(null)
  const [emailSent, setEmailSent] = useState(false)

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting }
  } = useForm<ForgotPasswordFormData>({
    resolver: yupResolver(forgotPasswordSchema),
    defaultValues: {
      email: ""
    },
    mode: 'onSubmit',
    reValidateMode: 'onSubmit'
  })

  const onSubmit = async (data: ForgotPasswordFormData) => {
    setForgotPasswordError(null)
    try {
      await forgotPassword(data.email)
      setEmailSent(true)
      showSuccess("Password reset instructions have been sent to your email")
    } catch (error: unknown) {
      const parsedError = parseError(error)
      const errorMessage = parsedError.message || "Failed to send reset email. Please try again."
      setForgotPasswordError(errorMessage)
      showError(errorMessage)
    }
  }

  if (emailSent) {
    return (
      <div className="flex flex-col gap-6 text-center">
        <div className="flex flex-col items-center gap-3">
          <div className="flex size-14 items-center justify-center rounded-full bg-emerald-500/10">
            <CheckCircle2 className="size-7 text-emerald-600" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Check your email</h1>
          <p className="max-w-xs text-sm text-muted-foreground">
            If an account exists with this email, password reset instructions will arrive shortly.
          </p>
        </div>

        <Button asChild className="w-full">
          <Link to="/login">Back to login</Link>
        </Button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col items-center gap-1 text-center">
        <h1 className="text-2xl font-semibold tracking-tight">Forgot your password?</h1>
        <p className="text-sm text-muted-foreground">
          Enter your email and we'll send reset instructions.
        </p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)}>
        <FieldGroup>
          {forgotPasswordError && (
            <div
              role="alert"
              className="flex items-start gap-2.5 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive"
            >
              <AlertCircle className="mt-0.5 size-4 shrink-0" />
              <span>{forgotPasswordError}</span>
            </div>
          )}
          <FormEmailField
            label="Email"
            placeholder="you@company.com"
            disabled={isSubmitting}
            error={errors.email?.message}
            required
            {...register("email")}
          />
          <FormSubmitButton
            isSubmitting={isSubmitting}
            submitText="Send reset instructions"
            submittingText="Sending..."
            className="mt-1 w-full"
          />
        </FieldGroup>
      </form>

      <div className="text-center text-sm text-muted-foreground">
        Remember your password?{" "}
        <Link to="/login" className="font-medium text-primary underline-offset-4 hover:underline">
          Back to login
        </Link>
      </div>
    </div>
  )
}

export default ForgotPasswordForm
