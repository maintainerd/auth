import { useState, useEffect } from "react"
import { useParams, useNavigate, useLocation } from "react-router-dom"
import { useForm, Controller, type Resolver } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { ArrowLeft, AlertCircle, TriangleAlert } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { DetailsContainer } from "@/components/container"
import { FormPageHeader } from "@/components/header"
import {
  FormTextareaField,
  FormSelectField,
  FormSubmitButton,
  type SelectOption
} from "@/components/form"
import { FormSlugField, FormSwitchSubContainer, FormSearchableSelectField, FormCheckboxSubContainer } from "@/components/inputs"
import { registrationFlowSchema, type RegistrationFlowFormData } from "@/lib/validations"
import { sanitizeFlowName } from "@/lib/validations/regex"
import {
  useRegistrationFlow,
  useCreateRegistrationFlow,
  useUpdateRegistrationFlow,
  useRegistrationFlowRoles,
} from "@/hooks/useRegistrationFlows"
import { useClients } from "@/hooks/useClients"
import { useRoles } from "@/hooks/useRoles"
import { useToast } from "@/hooks/useToast"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import { ConfirmationDialog } from "@/components/dialog"
import type { RegistrationFlowStatus } from "@/services/api/registration-flows/types"

// 'active' and 'inactive' are the only statuses the backend accepts (see
// validation_registration_flow.go). A 'draft' option produced a 400 on save.
const STATUS_OPTIONS: SelectOption[] = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
]

const REGISTRATION_FIELDS = [
  { value: "email", label: "Email" },
  { value: "fullname", label: "Full name" },
  { value: "phone", label: "Phone" },
] as const

export default function RegistrationFlowAddOrUpdateForm() {
  const { registrationFlowId } = useParams<{ registrationFlowId?: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError, parseError } = useToast()

  const isEditing = Boolean(registrationFlowId)
  const isCreating = !isEditing

  const navState = location.state as { from?: string; backLabel?: string } | null
  const listingUrl = `/registration-flows`
  const backTo = navState?.from ?? listingUrl
  const backLabel = navState?.backLabel ?? "Back to Registration Flows"

  const { data: registrationFlow, isLoading: isFetchingRegistrationFlow } = useRegistrationFlow(registrationFlowId || "")
  const createRegistrationFlowMutation = useCreateRegistrationFlow()
  const updateRegistrationFlowMutation = useUpdateRegistrationFlow()

  const [clientSearchValue, setClientSearchValue] = useState("")

  const { data: clientsData } = useClients({
    name: clientSearchValue || undefined,
    limit: 10,
    page: 1,
    sort_by: 'name',
    sort_order: 'asc',
  })

  const { data: rolesData, isLoading: isLoadingRoles } = useRoles({
    page: 1,
    limit: 100,
    sort_by: "name",
    sort_order: "asc",
  })
  const roleOptions = rolesData?.rows ?? []

  // The flow's currently-assigned roles, needed to hydrate the picker on edit.
  const { data: existingRolesData, isLoading: isFetchingFlowRoles } = useRegistrationFlowRoles(
    registrationFlowId || "",
    { page: 1, limit: 100 },
    { enabled: isEditing },
  )

  const {
    register,
    handleSubmit,
    control,
    watch,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<RegistrationFlowFormData>({
    resolver: yupResolver(registrationFlowSchema) as Resolver<RegistrationFlowFormData>,
    defaultValues: {
      name: "",
      description: "",
      status: "active",
      clientId: "",
      verificationRequired: false,
      requiredFields: [],
      roleIds: [],
    },
    mode: "onTouched",
    reValidateMode: "onChange",
  })

  const nameValue = watch("name")

  // Everything the form owns is hydrated in ONE reset, keyed on the fetched
  // records. verification_required, required_fields and the role selection used
  // to live in component state that the edit path never populated — so every
  // save silently posted `required_fields: []` and `verification_required: false`,
  // wiping the flow's field requirements and downgrading its verification policy.
  // Keeping them in the form also lets isDirty (the unsaved-changes guard) see them.
  const isHydrating = isEditing && (isFetchingRegistrationFlow || isFetchingFlowRoles)

  useEffect(() => {
    if (!isEditing || !registrationFlow || isFetchingFlowRoles) return

    reset({
      name: registrationFlow.name,
      description: registrationFlow.description ?? "",
      status: registrationFlow.status,
      clientId: registrationFlow.client_id ?? "",
      verificationRequired: registrationFlow.verification_required ?? false,
      requiredFields: (registrationFlow.required_fields ?? []).filter(
        (field): field is (typeof REGISTRATION_FIELDS)[number]["value"] =>
          REGISTRATION_FIELDS.some((option) => option.value === field),
      ),
      roleIds: (existingRolesData?.rows ?? []).map((role) => role.role_id),
    })
  }, [isEditing, registrationFlow, existingRolesData, isFetchingFlowRoles, reset])

  const isLoading =
    isFetchingRegistrationFlow ||
    createRegistrationFlowMutation.isPending ||
    updateRegistrationFlowMutation.isPending ||
    isSubmitting

  const existingFlow = registrationFlow
  // A rename changes the flow's public registration link.
  const isRenamed = Boolean(existingFlow && nameValue && nameValue !== existingFlow.name)
  const pageTitle = isCreating ? "Create Registration Flow" : `Edit ${existingFlow?.name || "Registration Flow"}`
  const submitButtonText = isCreating ? "Create Registration Flow" : "Update Registration Flow"

  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const onSubmit = async (data: RegistrationFlowFormData) => {
    try {
      // There is no `identifier` to submit: the name is the selector, and on
      // create and refuses to change it afterwards, so a published registration
      // link keeps resolving after a rename.
      let flowId: string
      if (isEditing) {
        await updateRegistrationFlowMutation.mutateAsync({
          registrationFlowId: registrationFlowId!,
          data: {
            name: data.name,
            description: data.description,
            status: data.status as RegistrationFlowStatus,
            verification_required: data.verificationRequired,
            required_fields: data.requiredFields,
            role_ids: data.roleIds,
          },
        })
        flowId = registrationFlowId!
      } else {
        const createdFlow = await createRegistrationFlowMutation.mutateAsync({
          name: data.name,
          description: data.description,
          status: data.status as RegistrationFlowStatus,
          client_id: data.clientId,
          verification_required: data.verificationRequired,
          required_fields: data.requiredFields,
          role_ids: data.roleIds,
        })
        flowId = createdFlow.registration_flow_id
      }

      showSuccess(isEditing ? "Registration flow updated successfully" : "Registration flow created successfully")
      navigate(`/registration-flows/${flowId}`)
    } catch (error) {
      // Route backend errors onto the offending field where we can: structured
      // field errors first, otherwise keyword-match the message (this is what
      // surfaces the 409 "registration flow with this name already exists" on the
      // name field). The backend keys field errors by its snake_case JSON tag.
      const parsed = parseError(error)
      const BACKEND_FIELD_MAP: Record<string, keyof RegistrationFlowFormData> = {
        name: "name",
        description: "description",
        status: "status",
        client_id: "clientId",
        verification_required: "verificationRequired",
        required_fields: "requiredFields",
        role_ids: "roleIds",
      }
      let mappedToField = false
      if (parsed.fieldErrors) {
        for (const [field, message] of Object.entries(parsed.fieldErrors)) {
          const rhfField = BACKEND_FIELD_MAP[field]
          if (rhfField) {
            setError(rhfField, { type: "server", message })
            mappedToField = true
          }
        }
      }
      if (!mappedToField) {
        const lower = parsed.message.toLowerCase()
        const match = Object.entries(BACKEND_FIELD_MAP).find(
          ([backendField, rhfField]) => lower.includes(backendField) || lower.includes(rhfField.toLowerCase())
        )
        if (match) {
          setError(match[1], { type: "server", message: parsed.message })
        }
      }
      showError(error)
    }
  }

  // Loading state while fetching the flow (and its roles) to edit. Rendering the
  // skeleton until BOTH have resolved means the single reset above always runs
  // before the form is interactive, so it can never clobber a user's edit.
  if (isHydrating) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Registration Flow"
            description="Update registration flow configuration and settings"
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

  if (isEditing && !isFetchingRegistrationFlow && !registrationFlow) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Registration Flow"
            description="Update registration flow configuration and settings"
          />
          <Card>
            <CardContent className="flex flex-col items-center justify-center gap-4 py-12 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <AlertCircle className="size-6" />
              </div>
              <div className="space-y-1">
                <h2 className="text-lg font-semibold">Registration flow not found</h2>
                <p className="text-sm text-muted-foreground">
                  The registration flow you're looking for doesn't exist or may have been removed.
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
              ? "Configure a new registration flow for user registration"
              : "Update registration flow configuration and settings"
          }
          showSystemBadge={existingFlow?.is_system}
          showWarning={existingFlow?.is_system}
          warningMessage="This is a system registration flow. Some settings may be restricted to prevent system instability."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6" key={registrationFlowId || "create"}>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Basic Information</CardTitle>
              <p className="text-sm text-muted-foreground">
                The name, description, status, and associated client.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                {/* The name IS the public registration-link selector, so it is a
                    slug field. The sanitizer matches the backend pattern exactly
                    (and turns spaces into hyphens) so the input can never produce
                    a value validation would reject. */}
                <FormSlugField
                  label="Name"
                  placeholder="e.g., partner-signup"
                  description="Used in this flow's registration link. Lowercase letters, numbers, hyphens and underscores."
                  sanitize={sanitizeFlowName}
                  disabled={isLoading || existingFlow?.is_system}
                  error={errors.name?.message}
                  required
                  {...register("name")}
                />

                <Controller
                  name="status"
                  control={control}
                  render={({ field }) => (
                    <FormSelectField
                      key={`status-${field.value || 'empty'}`}
                      label="Status"
                      placeholder="Select status"
                      options={STATUS_OPTIONS}
                      value={field.value}
                      onValueChange={field.onChange}
                      disabled={isLoading}
                      error={errors.status?.message}
                      description="An inactive flow refuses every registration through its link."
                      required
                    />
                  )}
                />
              </div>

              {/* Renaming changes the flow's public link, so say so plainly
                  rather than letting an operator discover it from a partner's
                  bug report. */}
              {isEditing && isRenamed && (
                <Alert variant="destructive">
                  <TriangleAlert className="size-4" />
                  <AlertDescription>
                    Renaming this flow changes its registration link. Any link already published by
                    an external app will stop working until it is updated to use{" "}
                    <code className="font-mono">{nameValue}</code>.
                  </AlertDescription>
                </Alert>
              )}

              <FormTextareaField
                label="Description"
                placeholder="Provide a detailed description of the registration flow"
                rows={4}
                disabled={isLoading}
                error={errors.description?.message}
                {...register("description")}
              />

              <div className="grid gap-4 md:grid-cols-2">
                <Controller
                  name="clientId"
                  control={control}
                  render={({ field }) => (
                    <FormSearchableSelectField
                      id="client"
                      label="Client"
                      placeholder="Select client..."
                      emptyText="No client found."
                      value={field.value}
                      onValueChange={field.onChange}
                      options={(clientsData?.rows ?? []).map((client) => ({
                        value: client.client_id,
                        label: client.name,
                        description: client.display_name,
                      }))}
                      searchValue={clientSearchValue}
                      onSearchChange={setClientSearchValue}
                      disabled={isLoading || isEditing}
                      error={errors.clientId?.message}
                      required
                      description={
                        isEditing
                          ? "The client is fixed after creation — the registration link is only valid for this client."
                          : "The client that provides branding, context, and validated callback URIs for this flow. Selecting a client does not automatically activate this flow."
                      }
                    />
                  )}
                />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Configuration</CardTitle>
              <p className="text-sm text-muted-foreground">
                Configure registration flow behavior and settings
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <Controller
                name="verificationRequired"
                control={control}
                render={({ field }) => (
                  <FormSwitchSubContainer
                    id="verification-required"
                    label="Require email verification"
                    description="Require users to verify their email before completing onboarding, even when the tenant-wide policy is less strict."
                    checked={Boolean(field.value)}
                    onCheckedChange={field.onChange}
                    disabled={isLoading}
                  />
                )}
              />

              <Controller
                name="requiredFields"
                control={control}
                render={({ field }) => {
                  const value = field.value ?? []
                  return (
                    <div data-md-listing-nested className="space-y-3 rounded-md border p-4">
                      <div>
                        <Label>Required registration fields</Label>
                        <p className="text-xs text-muted-foreground">
                          Username and password are always required. Select any additional fields this flow must collect.
                        </p>
                      </div>
                      <div className="grid gap-3 md:grid-cols-3">
                        {REGISTRATION_FIELDS.map((option) => (
                          <div key={option.value} className="flex items-center gap-2">
                            <Checkbox
                              id={`required-${option.value}`}
                              checked={value.includes(option.value)}
                              onCheckedChange={(checked) => {
                                field.onChange(
                                  checked === true
                                    ? [...new Set([...value, option.value])]
                                    : value.filter((entry) => entry !== option.value),
                                )
                              }}
                              disabled={isLoading}
                            />
                            <Label htmlFor={`required-${option.value}`} className="cursor-pointer font-normal">
                              {option.label}
                            </Label>
                          </div>
                        ))}
                      </div>
                      {errors.requiredFields && (
                        <p className="text-sm text-destructive">{errors.requiredFields.message}</p>
                      )}
                    </div>
                  )
                }}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Roles</CardTitle>
              <p className="text-sm text-muted-foreground">
                Optional — roles automatically assigned to users who complete this registration flow.
              </p>
            </CardHeader>
            <CardContent>
              <Controller
                name="roleIds"
                control={control}
                render={({ field }) => {
                  const value = field.value ?? []
                  const toggleRole = (id: string) =>
                    field.onChange(value.includes(id) ? value.filter((entry) => entry !== id) : [...value, id])

                  return (
                    <FormCheckboxSubContainer
                      options={roleOptions.map((role) => ({
                        value: role.role_id,
                        title: role.name,
                        description: role.description,
                      }))}
                      selected={value}
                      onToggle={toggleRole}
                      disabled={isLoading}
                      loading={isLoadingRoles}
                      emptyText="No roles available"
                      footer={
                        value.length > 0 ? (
                          <p className="mt-2 text-xs text-muted-foreground">
                            {value.length} role{value.length !== 1 ? "s" : ""} selected
                          </p>
                        ) : undefined
                      }
                    />
                  )
                }}
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
              submittingText="Saving..."
              submitText={submitButtonText}
              disabled={existingFlow?.is_system && isEditing}
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
