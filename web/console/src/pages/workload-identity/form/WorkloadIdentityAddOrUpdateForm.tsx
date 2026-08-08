import { useEffect, useRef, useState } from "react"
import { AlertCircle, ArrowLeft } from "lucide-react"
import { useNavigate, useParams, useLocation } from "react-router-dom"
import { useForm, Controller, type Resolver } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { DetailsContainer } from "@/components/container"
import { FormPageHeader } from "@/components/header"
import {
  FormInputField,
  FormSelectField,
  FormSubmitButton,
  FormTextareaField,
  type SelectOption,
} from "@/components/form"
import { FormUrlField, FormScopeField, FormSwitchSubContainer } from "@/components/inputs"
import { ConfirmationDialog } from "@/components/dialog"
import { useToast } from "@/hooks/useToast"
import { useClients } from "@/hooks/useClients"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import {
  useWorkloadIdentity,
  useCreateWorkloadIdentity,
  useUpdateWorkloadIdentity,
} from "@/hooks/useWorkloadIdentity"
import {
  workloadIdentitySchema,
  validateAttributeMapping,
  parseAllowedScopes,
  type WorkloadIdentityFormData,
} from "@/lib/validations/workloadIdentitySchema"
import {
  AttributeMappingEditor,
  type AttributeMappingRow,
} from "../components/AttributeMappingEditor"

let rowSeq = 0
const nextRowId = () => `mapping-${++rowSeq}`

export default function WorkloadIdentityAddOrUpdateForm() {
  const { federationId } = useParams<{ federationId: string }>()
  const isEditing = Boolean(federationId)
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError, parseError } = useToast()

  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? "/workload-identity"
  const backLabel = navState?.backLabel ?? "Back to Workload Identity"

  const { data: federation, isLoading: isFetching } = useWorkloadIdentity(federationId || "")
  const createMutation = useCreateWorkloadIdentity()
  const updateMutation = useUpdateWorkloadIdentity()

  // The client picker. The server caps limit at 100; a tenant with more clients
  // would otherwise silently lose the rest from the dropdown.
  const { data: clientsData } = useClients({ page: 1, limit: 100 })
  const clientOptions: SelectOption[] = (clientsData?.rows ?? []).map((client) => ({
    value: client.client_id,
    label: client.display_name || client.name,
  }))

  // Structured mapping rows — see AttributeMappingEditor for why this is not free text.
  const [mappingRows, setMappingRows] = useState<AttributeMappingRow[]>([])
  const [mappingError, setMappingError] = useState("")
  const [isLocalDirty, setIsLocalDirty] = useState(false)
  const markDirty = () => setIsLocalDirty(true)
  const hydrated = useRef(false)

  const {
    register,
    handleSubmit,
    control,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<WorkloadIdentityFormData>({
    resolver: yupResolver(workloadIdentitySchema) as Resolver<WorkloadIdentityFormData>,
    defaultValues: {
      client_uuid: "",
      name: "",
      description: "",
      issuer_url: "",
      audience: "",
      subject_claim: "sub",
      subject_pattern: "",
      allowed_scopes: "",
      is_active: true,
    },
  })

  useEffect(() => {
    if (!isEditing || !federation || hydrated.current) return
    hydrated.current = true

    reset({
      client_uuid: federation.client_uuid,
      name: federation.name,
      description: federation.description ?? "",
      issuer_url: federation.issuer_url,
      audience: federation.audience,
      subject_claim: federation.subject_claim || "sub",
      subject_pattern: federation.subject_pattern,
      allowed_scopes: (federation.allowed_scopes ?? []).join(", "),
      is_active: federation.is_active,
    })

    setMappingRows(
      Object.entries(federation.attribute_mapping ?? {}).map(([externalClaim, tokenClaim]) => ({
        id: nextRowId(),
        externalClaim,
        tokenClaim,
      })),
    )
  }, [isEditing, federation, reset])

  const addMappingRow = () => {
    markDirty()
    setMappingRows((prev) => [...prev, { id: nextRowId(), externalClaim: "", tokenClaim: "" }])
  }
  const updateMappingRow = (id: string, patch: Partial<Omit<AttributeMappingRow, "id">>) => {
    markDirty()
    setMappingRows((prev) => prev.map((row) => (row.id === id ? { ...row, ...patch } : row)))
  }
  const removeMappingRow = (id: string) => {
    markDirty()
    setMappingRows((prev) => prev.filter((row) => row.id !== id))
  }

  /** Rows → the wire object, dropping fully-blank rows. */
  const buildAttributeMapping = (): Record<string, string> => {
    const mapping: Record<string, string> = {}
    for (const row of mappingRows) {
      const external = row.externalClaim.trim()
      const tokenClaim = row.tokenClaim.trim()
      if (!external && !tokenClaim) continue
      mapping[external] = tokenClaim
    }
    return mapping
  }

  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(
    isDirty || isLocalDirty,
  )

  const onSubmit = async (formData: WorkloadIdentityFormData) => {
    const attributeMapping = buildAttributeMapping()

    // A malformed mapping must block the save. It used to be parsed from a JSON
    // textarea whose failure was swallowed, so a typo silently replaced a saved
    // mapping with {} and still reported success.
    const mappingProblem = validateAttributeMapping(attributeMapping)
    if (mappingProblem) {
      setMappingError(mappingProblem)
      showError(mappingProblem)
      return
    }
    setMappingError("")

    const payload = {
      name: formData.name,
      description: formData.description || "",
      issuer_url: formData.issuer_url,
      audience: formData.audience,
      subject_claim: formData.subject_claim || "sub",
      subject_pattern: formData.subject_pattern,
      allowed_scopes: parseAllowedScopes(formData.allowed_scopes),
      attribute_mapping: attributeMapping,
      is_active: formData.is_active,
    }

    try {
      if (isEditing && federationId) {
        await updateMutation.mutateAsync({ federationId, data: payload })
        showSuccess("Federation updated")
      } else {
        await createMutation.mutateAsync({ ...payload, client_uuid: formData.client_uuid })
        showSuccess("Federation created")
      }
      navigate(backTo)
    } catch (error) {
      // The server owns rules the console cannot check — a live OIDC probe of the
      // issuer, and name uniqueness. Route those onto the offending field instead of
      // leaving a red toast beside a form that looks valid.
      const parsed = parseError(error)
      const message = parsed.message ?? ""
      if (/issuer/i.test(message)) {
        setError("issuer_url", { message })
      } else if (/name/i.test(message) && /exists/i.test(message)) {
        setError("name", { message })
      }
      showError(error)
    }
  }

  const isSaving = isSubmitting || createMutation.isPending || updateMutation.isPending

  const pageTitle = isEditing
    ? `Edit ${federation?.name || "Workload Identity Federation"}`
    : "New Workload Identity Federation"
  const pageDescription =
    "Let an external workload exchange its own OIDC token for an access token, with no stored secret to leak or rotate."


  if (isEditing && isFetching) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Workload Identity Federation"
            description={pageDescription}
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

  if (isEditing && !isFetching && !federation) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            title="Edit Workload Identity Federation"
            description={pageDescription}
          />
          <Card>
            <CardContent className="flex flex-col items-center justify-center gap-4 py-12 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <AlertCircle className="size-6" />
              </div>
              <div className="space-y-1">
                <h2 className="text-lg font-semibold">Federation not found</h2>
                <p className="text-sm text-muted-foreground">
                  This workload identity federation doesn&apos;t exist or may have been removed.
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
          description={pageDescription}
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Basic Information</CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <FormInputField
                label="Name"
                required
                placeholder="github-actions-deploy"
                error={errors.name?.message}
                disabled={isSaving}
                {...register("name")}
              />

              <FormTextareaField
                label="Description"
                rows={3}
                placeholder="What workload does this trust, and why?"
                error={errors.description?.message}
                disabled={isSaving}
                {...register("description")}
              />

              <Controller
                name="client_uuid"
                control={control}
                render={({ field }) => (
                  <FormSelectField
                    label="Client"
                    required
                    placeholder="Select the client this federation issues tokens for"
                    options={clientOptions}
                    value={field.value}
                    onValueChange={(value) => { markDirty(); field.onChange(value) }}
                    error={errors.client_uuid?.message}
                    // The mapped client is what the issued token acts as, so changing
                    // it would silently repoint an existing trust at another identity.
                    disabled={isSaving || isEditing}
                    description={
                      isEditing
                        ? "The mapped client cannot be changed after creation."
                        : "Tokens issued through this federation act as this client."
                    }
                  />
                )}
              />

              <Controller
                name="is_active"
                control={control}
                render={({ field }) => (
                  <FormSwitchSubContainer
                    id="is-active"
                    label="Active"
                    description="When off, matching workloads cannot exchange their tokens. This is the kill switch for this trust."
                    checked={field.value}
                    onCheckedChange={(checked) => { markDirty(); field.onChange(checked) }}
                    disabled={isSaving}
                  />
                )}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Trust</CardTitle>
              <p className="text-sm text-muted-foreground">
                Which external tokens this federation accepts. These three fields together
                are the entire trust boundary.
              </p>
            </CardHeader>
            <CardContent className="space-y-6">
              <FormUrlField
                label="Issuer URL"
                required
                placeholder="https://token.actions.githubusercontent.com"
                description="The OIDC issuer of the workload's token. Verified by live discovery when you save, so it must be reachable."
                error={errors.issuer_url?.message}
                disabled={isSaving}
                {...register("issuer_url")}
              />

              <FormInputField
                label="Audience"
                required
                placeholder="https://auth.example.com"
                description="The aud claim the workload's token must carry."
                error={errors.audience?.message}
                disabled={isSaving}
                {...register("audience")}
              />

              <FormInputField
                label="Subject claim"
                placeholder="sub"
                description="Which claim identifies the workload. Defaults to sub."
                error={errors.subject_claim?.message}
                disabled={isSaving}
                {...register("subject_claim")}
              />

              <FormInputField
                label="Subject pattern"
                required
                placeholder="repo:my-org/my-repo:*"
                description="Which workloads are trusted. * and ? are wildcards. This is the only thing separating your workloads from everyone else's on a shared issuer, so it must be anchored on your organisation or namespace."
                error={errors.subject_pattern?.message}
                disabled={isSaving}
                {...register("subject_pattern")}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Issued Token</CardTitle>
              <p className="text-sm text-muted-foreground">
                What the workload receives once its token is accepted.
              </p>
            </CardHeader>
            <CardContent className="space-y-6">
              <Controller
                name="allowed_scopes"
                control={control}
                render={({ field }) => (
                  <FormScopeField
                    label="Allowed scopes"
                    value={field.value ?? ""}
                    onValueChange={(value) => { markDirty(); field.onChange(value) }}
                    error={errors.allowed_scopes?.message}
                    disabled={isSaving}
                    description="The most this federation may ever grant. A request for anything outside this list is rejected."
                  />
                )}
              />

              <AttributeMappingEditor
                rows={mappingRows}
                error={mappingError}
                disabled={isSaving}
                onAdd={addMappingRow}
                onUpdate={updateMappingRow}
                onRemove={removeMappingRow}
              />
            </CardContent>
          </Card>

          <div className="flex justify-end gap-3">
            <Button
              type="button"
              variant="outline"
              onClick={() => guard(() => navigate(backTo))}
              disabled={isSaving}
            >
              Cancel
            </Button>
            <FormSubmitButton
              isSubmitting={isSaving}
              submitText={isEditing ? "Save Changes" : "Create Federation"}
              submittingText={isEditing ? "Saving..." : "Creating..."}
            />
          </div>
        </form>
      </div>
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
    </DetailsContainer>
  )
}
