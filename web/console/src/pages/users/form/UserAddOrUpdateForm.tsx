import { useEffect, useMemo } from "react"
import { useParams, useNavigate, useLocation } from "react-router-dom"
import { useForm, Controller } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import * as yup from "yup"
import { ArrowLeft, Plus, AlertCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { DetailsContainer } from "@/components/container"
import { FormPageHeader } from "@/components/header"
import {
  FormInputField,
  FormSelectField,
  FormSubmitButton,
  type SelectOption
} from "@/components/form"
import {
  FormEmailField,
  FormPhoneFieldWithCountry,
  FormPasswordFieldWithPolicy,
  MetadataFieldEditor,
} from "@/components/inputs"
import { buildUserSchema } from "@/lib/validations"
import { useUser, useCreateUser, useUpdateUser } from "@/hooks/useUsers"
import { useToast } from "@/hooks/useToast"
import { useMetadataFields } from "@/hooks/useMetadataFields"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import { ConfirmationDialog } from "@/components/dialog"
import { useAppSelector } from "@/store/hooks"
import type { UserStatus } from "@/services/api/users/types"

// Status options for the select field
const STATUS_OPTIONS: SelectOption[] = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
  { value: "pending", label: "Pending" },
  { value: "suspended", label: "Suspended" },
]

export default function UserAddOrUpdateForm() {
  const { userId } = useParams<{ userId?: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError, parseError } = useToast()

  const isEditing = Boolean(userId)
  const isCreating = !isEditing

  // Custom metadata via shared hook
  const {
    customFields,
    metadataError,
    addCustomField,
    removeCustomField,
    updateCustomField,
    buildPayload,
    resetFields,
  } = useMetadataFields()

  // Fetch existing user if editing
  const { data: userData, isLoading: isFetchingUser } = useUser(userId || '')
  const createUserMutation = useCreateUser()

  // Read tenant password config from Redux (populated by AppBootstrap after bootstrap)
  const passwordConfig = useAppSelector(
    (state) => state.tenant.currentTenant?.password_config
  )
  const updateUserMutation = useUpdateUser()

  // Honour where the user came from (e.g. the details page) so the back button,
  // Cancel, and post-submit navigation return there. Falls back to the listing.
  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/users`
  const backLabel = navState?.backLabel ?? (backTo === `/users` ? "Back to Users" : "Back")

  // Build schema dynamically from tenant password config
  const schema = useMemo(() => buildUserSchema(passwordConfig), [passwordConfig])

  // React Hook Form setup
  const {
    register,
    handleSubmit,
    control,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty }
  } = useForm({
    resolver: yupResolver(schema),
    context: { isCreating },
    defaultValues: {
      username: "",
      email: "",
      phone: undefined,
      password: undefined,
      status: "active" as const,
    },
    mode: 'onTouched',
    reValidateMode: 'onChange',
  })

  // Load existing user data if editing
  useEffect(() => {
    if (isEditing && userData) {
      reset({
        username: userData.username,
        email: userData.email,
        phone: userData.phone || undefined,
        password: undefined,
        status: userData.status,
      })
      resetFields(userData.metadata)
    }
  }, [isEditing, userData, reset, resetFields])

  const isLoading = createUserMutation.isPending || updateUserMutation.isPending || isSubmitting

  // Warn before discarding unsaved edits (browser close/refresh + guarded exits).
  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const onSubmit = async (data: yup.InferType<typeof schema>) => {
    if (metadataError) {
      showError(metadataError)
      return
    }

    const metadataPayload = buildPayload()

    try {
      if (isCreating) {
        await createUserMutation.mutateAsync({
          username: data.username,
          email: data.email,
          phone: data.phone || undefined,
          password: data.password!,
          status: data.status as UserStatus,
          metadata: metadataPayload,
        })
        showSuccess("User created successfully")
      } else {
        await updateUserMutation.mutateAsync({
          userId: userId!,
          data: {
            username: data.username,
            email: data.email,
            phone: data.phone || undefined,
            status: data.status as UserStatus,
            metadata: metadataPayload,
          },
        })
        showSuccess("User updated successfully")
      }

      navigate(backTo)
    } catch (error) {
      // Route backend errors onto the offending field where we can: structured
      // field errors first, otherwise keyword-match the message (duplicate
      // username/email, or a password-policy rejection the client can't pre-check
      // — strength/common/HIBP). Anything unmapped still shows via the toast.
      const parsed = parseError(error)
      const known = ["username", "email", "phone", "password", "status"] as const
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
        const field = known.find((f) => lower.includes(f) && (f !== "password" || isCreating))
        if (field) {
          setError(field, { type: "server", message: parsed.message })
        }
      }
      showError(error)
    }
  }

  const pageTitle = isCreating ? "Create User" : `Edit ${userData?.fullname || userData?.username}`
  const submitButtonText = isCreating ? "Create User" : "Update User"

  // Loading state while fetching the user to edit
  if (isEditing && isFetchingUser) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit User"
            description="Update user account information and settings"
          />
          <Card>
            <CardContent className="space-y-4 pt-6">
              <Skeleton className="h-5 w-40" />
              <div className="grid gap-4 md:grid-cols-2">
                {Array.from({ length: 4 }).map((_, i) => (
                  <Skeleton key={i} className="h-10 w-full" />
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      </DetailsContainer>
    )
  }

  // Not-found state
  if (isEditing && !isFetchingUser && !userData) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit User"
            description="Update user account information and settings"
          />
          <Card>
            <CardContent className="flex flex-col items-center justify-center gap-4 py-12 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <AlertCircle className="size-6" />
              </div>
              <div className="space-y-1">
                <h2 className="text-lg font-semibold">User not found</h2>
                <p className="text-sm text-muted-foreground">
                  The user you're trying to edit doesn't exist or may have been removed.
                </p>
              </div>
              <Button variant="outline" onClick={() => navigate(backTo)}>
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
              ? "Create a new user account with credentials and basic information"
              : "Update user account information and settings"
          }
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6" key={userId || 'create'}>
          {/* Account */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Account</CardTitle>
              <p className="text-sm text-muted-foreground">
                Sign-in credentials and contact details for this user.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                {/* Username */}
                <FormInputField
                  label="Username"
                  placeholder="e.g., john.doe"
                  description="Unique username for login"
                  disabled={isLoading}
                  error={errors.username?.message}
                  required
                  {...register("username")}
                />

                {/* Email */}
                <FormEmailField
                  label="Email"
                  description="User's email address"
                  disabled={isLoading}
                  error={errors.email?.message}
                  required
                  {...register("email")}
                />

                {/* Phone */}
                <Controller
                  name="phone"
                  control={control}
                  render={({ field }) => (
                    <FormPhoneFieldWithCountry
                      label="Phone"
                      description="Optional — select country code and enter local number"
                      disabled={isLoading}
                      error={errors.phone?.message}
                      value={field.value}
                      onChange={field.onChange}
                      onBlur={field.onBlur}
                    />
                  )}
                />

                {/* Status */}
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
                      // Status is a constrained select defaulting to "active", so it
                      // cannot produce a validation error — the branch is unreachable.
                      /* v8 ignore next */
                      error={errors.status?.message}
                      required
                    />
                  )}
                />
              </div>

              {/* Password (only for create) */}
              {isCreating && (
                <FormPasswordFieldWithPolicy
                  label="Password"
                  placeholder="Enter a strong password"
                  disabled={isLoading}
                  error={errors.password?.message}
                  required
                  passwordConfig={passwordConfig}
                  {...register("password")}
                />
              )}
            </CardContent>
          </Card>

          {/* Custom Metadata */}
          <Card>
            <CardHeader>
              <div className="flex items-start justify-between gap-4">
                <div>
                  <CardTitle className="text-base">Custom Metadata</CardTitle>
                  <p className="text-sm text-muted-foreground mt-1.5">
                    Optional key-value pairs stored alongside the user.
                  </p>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={addCustomField}
                  disabled={isLoading}
                  className="h-9 shrink-0 gap-2"
                >
                  <Plus className="size-4" />
                  Add Field
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <MetadataFieldEditor
                fields={customFields}
                error={metadataError}
                disabled={isLoading}
                onAdd={addCustomField}
                onUpdate={updateCustomField}
                onRemove={removeCustomField}
              />
            </CardContent>
          </Card>

          {/* Form Actions */}
          <div className="flex justify-end gap-3">
            <Button
              type="button"
              variant="outline"
              onClick={() => guard(() => navigate(backTo))}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <FormSubmitButton isSubmitting={isLoading} submitText={submitButtonText} />
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
