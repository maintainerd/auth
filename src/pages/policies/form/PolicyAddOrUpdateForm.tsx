import { useEffect, useState } from "react"
import { useParams, useNavigate, useLocation } from "react-router-dom"
import { useForm, Controller, useFieldArray } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { AlertCircle, ArrowLeft, Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { DetailsContainer } from "@/components/container"
import { FormPageHeader } from "@/components/header"
import {
  FormInputField,
  FormTextareaField,
  FormSelectField,
  FormSubmitButton,
  type SelectOption
} from "@/components/form"
import { ConfirmationDialog } from "@/components/dialog"
import { policySchema, type PolicyFormData } from "@/lib/validations"
import { usePolicy, useCreatePolicy, useUpdatePolicy } from "@/hooks/usePolicies"
import { useToast } from "@/hooks/useToast"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import { StatementField } from "./components"

// Status options for the select field
const STATUS_OPTIONS: SelectOption[] = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
]

// Backend snake_case field keys → form field names, for routing structured
// server validation errors onto the offending inputs.
const BACKEND_FIELD_MAP: Record<string, keyof PolicyFormData> = {
  name: "name",
  description: "description",
  version: "version",
  status: "status",
}

export default function PolicyAddOrUpdateForm() {
  const { policyId } = useParams<{ policyId?: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError, parseError } = useToast()

  const isEditing = Boolean(policyId)
  const isCreating = !isEditing

  // Honour where the user came from so back/cancel/save return there.
  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/policies`
  const backLabel = navState?.backLabel ?? (backTo === `/policies` ? "Back to Policies" : "Back")

  // Fetch existing policy if editing
  const { data: policyData, isLoading: isFetchingPolicy } = usePolicy(policyId || '')
  const createPolicyMutation = useCreatePolicy()
  const updatePolicyMutation = useUpdatePolicy()

  // React Hook Form setup
  const {
    register,
    handleSubmit,
    control,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty }
  } = useForm<PolicyFormData>({
    resolver: yupResolver(policySchema),
    defaultValues: {
      name: "",
      description: "",
      version: "1.0.0",
      status: "active",
      document: {
        statement: [
          {
            effect: "allow",
            action: [""],
            resource: [""],
          }
        ]
      }
    },
    mode: 'onTouched',
    reValidateMode: 'onChange',
  })

  // Field array for managing statements
  const { fields: statements, append: appendStatement, remove: removeStatement } = useFieldArray({
    control,
    name: "document.statement"
  })

  // Load existing policy data if editing
  useEffect(() => {
    if (isEditing && policyData) {
      reset({
        name: policyData.name,
        description: policyData.description ?? "",
        version: policyData.version,
        status: policyData.status,
        document: {
          statement: policyData.document.statement,
        },
      })
    }
  }, [isEditing, policyData, reset])

  const isLoading = createPolicyMutation.isPending || updatePolicyMutation.isPending || isSubmitting
  const existingPolicy = policyData

  // Warn before discarding unsaved edits (browser close/refresh + guarded exits).
  // Recorded on the version-history row. Edit-only: a create has no prior version to
  // explain, and the history tab showed a blank author until the backend started
  // capturing this.
  const [changeReason, setChangeReason] = useState("")

  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(
    isDirty || changeReason.trim() !== "",
  )

  const onSubmit = async (data: PolicyFormData) => {
    try {
      // Ensure document.version matches the root version
      const requestData = {
        name: data.name,
        description: data.description,
        version: data.version,
        status: data.status,
        document: {
          ...data.document,
          version: data.version, // Sync document version with root version
        },
      }

      if (isCreating) {
        await createPolicyMutation.mutateAsync(requestData)
        showSuccess("Policy created successfully")
      } else {
        await updatePolicyMutation.mutateAsync({
          policyId: policyId!,
          data: {
            ...requestData,
            change_reason: changeReason.trim() || undefined,
          },
        })
        showSuccess("Policy updated successfully")
      }

      navigate(backTo)
    } catch (error) {
      // Route backend errors onto the offending field where we can: structured
      // field errors first, otherwise keyword-match the message. Anything
      // unmapped still shows via the toast.
      const parsed = parseError(error)
      let mappedToField = false
      if (parsed.fieldErrors) {
        for (const [field, message] of Object.entries(parsed.fieldErrors)) {
          const formField = BACKEND_FIELD_MAP[field]
          if (formField) {
            setError(formField, { type: "server", message })
            mappedToField = true
          }
        }
      }
      if (!mappedToField) {
        const lower = parsed.message.toLowerCase()
        // Most specific first, so "description" doesn't land on "name".
        const keywordOrder: Array<[string, keyof PolicyFormData]> = [
          ["description", "description"],
          ["version", "version"],
          ["status", "status"],
          ["name", "name"],
        ]
        const hit = keywordOrder.find(([keyword]) => lower.includes(keyword))
        if (hit) {
          setError(hit[1], { type: "server", message: parsed.message })
        }
      }
      showError(error)
    }
  }

  const pageTitle = isCreating ? "Create Policy" : `Edit ${existingPolicy?.name || "Policy"}`
  const submitButtonText = isCreating ? "Create Policy" : "Update Policy"

  // Show loading state while fetching policy data
  if (isEditing && isFetchingPolicy) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Policy"
            description="Update policy settings and configuration"
          />
          <Card>
            <CardContent className="space-y-4 pt-6">
              <Skeleton className="h-5 w-40" />
              <div className="grid gap-4 md:grid-cols-2">
                {Array.from({ length: 3 }).map((_, i) => (
                  <Skeleton key={i} className="h-10 w-full" />
                ))}
              </div>
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-48 w-full" />
            </CardContent>
          </Card>
        </div>
      </DetailsContainer>
    )
  }

  // Show error if policy not found
  if (isEditing && !isFetchingPolicy && !policyData) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Policy"
            description="Update policy settings and configuration"
          />
          <Card>
            <CardContent className="flex flex-col items-center justify-center gap-4 py-12 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <AlertCircle className="size-6" />
              </div>
              <div className="space-y-1">
                <h2 className="text-lg font-semibold">Policy not found</h2>
                <p className="text-sm text-muted-foreground">
                  The policy you're looking for doesn't exist or may have been removed.
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
              ? "Create a new AWS-style policy to control access to services and resources"
              : "Update policy settings and configuration"
          }
          showSystemBadge={existingPolicy?.is_system}
          showWarning={existingPolicy?.is_system}
          warningMessage="This is a system policy. Some settings may be restricted to prevent system instability."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6" key={policyId || "create"}>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Basic Information</CardTitle>
              <p className="text-sm text-muted-foreground">
                The policy name, version, status, and description.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField
                  label="Policy Name"
                  placeholder="e.g., user:policy, admin:policy"
                  description="Used for identification (lowercase, numbers, hyphens, colons, underscores only)"
                  disabled={existingPolicy?.is_system || isLoading}
                  error={errors.name?.message}
                  required
                  {...register("name")}
                />

                <FormInputField
                  label="Version"
                  placeholder="e.g., 1.0.0"
                  description="Semantic version for the policy"
                  disabled={isLoading}
                  error={errors.version?.message}
                  required
                  {...register("version")}
                />
              </div>

              <FormTextareaField
                label="Description"
                placeholder="Enter policy description"
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
                    key={`status-${field.value || 'empty'}`}
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
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-base">Policy Statements</CardTitle>
                  <p className="mt-1.5 text-sm text-muted-foreground">
                    Define allow or deny rules. Each statement applies only when both resource and action patterns match.
                  </p>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => appendStatement({ effect: "allow", action: [""], resource: [""] })}
                  className="gap-2"
                  disabled={isLoading}
                >
                  <Plus className="h-4 w-4" />
                  Add Statement
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-6">
              {statements.map((statement, statementIndex) => (
                <StatementField
                  key={statement.id}
                  statementIndex={statementIndex}
                  control={control}
                  errors={errors}
                  isLoading={isLoading}
                  canRemove={statements.length > 1}
                  onRemove={() => removeStatement(statementIndex)}
                />
              ))}
              {errors.document?.statement && (
                <p className="text-sm text-destructive">{errors.document.statement.message}</p>
              )}
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
              disabled={existingPolicy?.is_system && isEditing}
            />
          </div>
          {!isCreating && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Change Reason</CardTitle>
                <p className="text-sm text-muted-foreground">
                  Recorded against this edit in the policy&apos;s version history, alongside the
                  previous document. Optional, but it is what lets a reviewer tell why the
                  policy changed rather than only what changed.
                </p>
              </CardHeader>
              <CardContent>
                <FormTextareaField
                  label="Reason for this change"
                  rows={2}
                  placeholder="Tightened the billing scope after the access review"
                  value={changeReason}
                  onChange={(e) => setChangeReason(e.target.value)}
                  disabled={isLoading}
                />
              </CardContent>
            </Card>
          )}

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
