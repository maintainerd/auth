import { useState } from "react"
import { useForm } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import * as yup from 'yup'
import { AlertCircle } from "lucide-react"
import { FieldGroup } from "@/components/ui/field"
import { FormInputField, FormSelectField, FormSubmitButton } from "@/components/form"
import { useProfile } from "@/hooks/useProfile"
import { useAuth } from "@/hooks/useAuth"
import { genderOptions } from "@/lib/constants"
import type { CreateProfileRequest } from "@/services/api/auth/types"
import RegisterProfileSuccess from "./RegisterProfileSuccess"

const profileSchema = yup.object({
  first_name: yup.string().required('First name is required').min(1).max(100),
  last_name: yup.string().required('Last name is required').min(1).max(100),
  gender: yup.string().required('Please select a gender'),
})

type ProfileFormData = yup.InferType<typeof profileSchema>

// The signup form may already have collected a full name (when the registration
// flow required it). Split it on the first space purely as a starting point —
// both fields stay editable, so mononyms and non-Western name orders are not
// corrupted by the guess.
function prefillFromSignup(): { first_name: string; last_name: string; gender: string } {
  const stored = sessionStorage.getItem("register_fullname")?.trim() || ""
  if (!stored) return { first_name: "", last_name: "", gender: "" }
  const space = stored.indexOf(" ")
  if (space === -1) return { first_name: stored, last_name: "", gender: "" }
  return {
    first_name: stored.slice(0, space),
    last_name: stored.slice(space + 1).trim(),
    gender: "",
  }
}

const RegisterProfileForm = () => {
  const { isLoading, createProfileForRegister } = useProfile()
  const { refreshAccount } = useAuth()
  const [isProfileCreated, setIsProfileCreated] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const savedEmail = sessionStorage.getItem('register_email') || ''

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isSubmitting }
  } = useForm<ProfileFormData>({
    resolver: yupResolver(profileSchema),
    defaultValues: prefillFromSignup(),
    mode: 'onSubmit',
  })

  const watchedValues = watch()

  const onSubmit = async (data: ProfileFormData) => {
    setError(null)
    try {
      const firstName = data.first_name.trim()
      const lastName = data.last_name.trim()
      const displayName = `${firstName} ${lastName}`

      const profileData = {
        first_name: firstName,
        last_name: lastName,
        display_name: displayName,
        gender: data.gender,
        email: savedEmail || undefined,
      } as CreateProfileRequest

      const result = await createProfileForRegister(profileData)
      if (result.success) {
        sessionStorage.removeItem('register_email')
        sessionStorage.removeItem('register_fullname')
        // Sync auth state so the now-complete account (profile present) clears
        // the RouteGuard check when the success screen routes to login-success.
        await refreshAccount()
        setIsProfileCreated(true)
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create profile')
    }
  }

  if (isProfileCreated) {
    return <RegisterProfileSuccess />
  }

  return (
    <div className="flex flex-col gap-8">
      <div className="flex flex-col items-center gap-2 text-center">
        <h1 className="text-2xl font-semibold tracking-tight">Complete your profile</h1>
        <p className="text-sm text-muted-foreground">
          Just a few details to get started.
        </p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)}>
        <FieldGroup>
          {error && (
            <div role="alert" className="flex items-start gap-2.5 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
              <AlertCircle className="mt-0.5 size-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}
          <FormInputField
            label="First Name"
            placeholder="John"
            autoComplete="given-name"
            disabled={isSubmitting}
            error={errors.first_name?.message}
            required
            {...register("first_name")}
          />
          <FormInputField
            label="Last Name"
            placeholder="Doe"
            autoComplete="family-name"
            disabled={isSubmitting}
            error={errors.last_name?.message}
            required
            {...register("last_name")}
          />
          <FormSelectField
            label="Gender"
            placeholder="Select gender"
            options={genderOptions}
            value={watchedValues.gender || ""}
            onValueChange={(v) => setValue("gender", v, { shouldValidate: true })}
            error={errors.gender?.message}
            required
          />
          <FormSubmitButton
            isSubmitting={isSubmitting || isLoading}
            submitText="Create Profile"
            submittingText="Creating..."
            className="mt-1 w-full"
          />
        </FieldGroup>
      </form>
    </div>
  )
}

export default RegisterProfileForm
