import { useState } from "react"
import { format } from "date-fns"
import { Plus, Network, Pencil, Trash2 } from "lucide-react"
import { useForm, Controller, type Resolver } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import * as yup from "yup"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"
import { DetailsContainer } from "@/components/container"
import { PageContainer, PageHeader } from "@/components/layout"
import {
  FormInputField,
  FormSelectField,
  FormSwitchField,
  FormSubmitButton,
} from "@/components/form"
import { FormUrlField } from "@/components/inputs"
import { EmptyState, ListSkeleton } from "@/components/details"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useToast } from "@/hooks/useToast"
import { useClients } from "@/hooks/useClients"
import {
  useWorkloadIdentities,
  useCreateWorkloadIdentity,
  useUpdateWorkloadIdentity,
  useDeleteWorkloadIdentity,
} from "@/hooks/useWorkloadIdentity"
import type { WorkloadIdentityFederation } from "@/services/api/workload-identity/types"

const schema = yup.object({
  client_uuid: yup.string().required("Client is required"),
  name: yup.string().required("Name is required"),
  description: yup.string().optional(),
  issuer_url: yup
    .string()
    .url("Must be a valid URL")
    .required("Issuer URL is required"),
  audience: yup.string().required("Audience is required"),
  subject_claim: yup.string().optional().default("sub"),
  subject_pattern: yup.string().required("Subject pattern is required"),
  allowed_scopes: yup.string().optional(),
  attribute_mapping: yup.string().optional(),
  is_active: yup.boolean().default(true),
})

type WorkloadFormData = yup.InferType<typeof schema>

function parseAllowedScopes(val?: string | null): string[] | undefined {
  if (!val) return undefined
  return val
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean)
}

function parseAttributeMapping(
  val?: string | null,
): Record<string, string> | undefined {
  if (!val) return undefined
  try {
    const parsed = JSON.parse(val)
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
      return undefined
    }
    return parsed as Record<string, string>
  } catch {
    return undefined
  }
}

function serializeAllowedScopes(scopes?: string[] | null): string {
  return scopes?.join(", ") ?? ""
}

function serializeAttributeMapping(
  obj?: Record<string, string> | null,
): string {
  if (!obj) return ""
  return JSON.stringify(obj, null, 2)
}

export default function WorkloadIdentityPage() {
  const { showSuccess, showError } = useToast()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<WorkloadIdentityFederation | null>(null)

  const { data, isLoading } = useWorkloadIdentities({ page: 1, limit: 50 })
  const { data: clientsData } = useClients({ page: 1, limit: 100 })
  const createMutation = useCreateWorkloadIdentity()
  const updateMutation = useUpdateWorkloadIdentity()
  const deleteMutation = useDeleteWorkloadIdentity()

  const clientOptions = (clientsData?.rows ?? []).map((c) => ({
    value: c.client_id,
    label: c.display_name || c.name,
  }))

  const {
    register,
    control,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<WorkloadFormData>({
    resolver: yupResolver(schema) as unknown as Resolver<WorkloadFormData>,
    defaultValues: {
      client_uuid: "",
      name: "",
      issuer_url: "",
      audience: "",
      subject_claim: "sub",
      subject_pattern: "",
      is_active: true,
    },
  })

  const openCreate = () => {
    setEditing(null)
    reset({
      client_uuid: "",
      name: "",
      description: "",
      issuer_url: "",
      audience: "",
      subject_claim: "sub",
      subject_pattern: "",
      allowed_scopes: "",
      attribute_mapping: "",
      is_active: true,
    })
    setDialogOpen(true)
  }

  const openEdit = (item: WorkloadIdentityFederation) => {
    setEditing(item)
    reset({
      client_uuid: item.client_uuid,
      name: item.name,
      description: item.description ?? "",
      issuer_url: item.issuer_url,
      audience: item.audience,
      subject_claim: item.subject_claim,
      subject_pattern: item.subject_pattern,
      allowed_scopes: serializeAllowedScopes(item.allowed_scopes),
      attribute_mapping: serializeAttributeMapping(item.attribute_mapping),
      is_active: item.is_active,
    })
    setDialogOpen(true)
  }

  const onSubmit = async (formData: WorkloadFormData) => {
    try {
      if (editing) {
        await updateMutation.mutateAsync({
          federationId: editing.workload_identity_federation_uuid,
          data: {
            name: formData.name,
            description: formData.description || undefined,
            issuer_url: formData.issuer_url,
            audience: formData.audience,
            subject_claim: formData.subject_claim || undefined,
            subject_pattern: formData.subject_pattern,
            allowed_scopes: parseAllowedScopes(formData.allowed_scopes),
            attribute_mapping: parseAttributeMapping(formData.attribute_mapping),
            is_active: formData.is_active,
          },
        })
        showSuccess("Federation updated")
      } else {
        await createMutation.mutateAsync({
          client_uuid: formData.client_uuid,
          name: formData.name,
          description: formData.description || undefined,
          issuer_url: formData.issuer_url,
          audience: formData.audience,
          subject_claim: formData.subject_claim || undefined,
          subject_pattern: formData.subject_pattern,
          allowed_scopes: parseAllowedScopes(formData.allowed_scopes),
          attribute_mapping: parseAttributeMapping(formData.attribute_mapping),
          is_active: formData.is_active,
        })
        showSuccess("Federation created")
      }
      setDialogOpen(false)
    } catch (err) {
      showError(err)
    }
  }

  const getRowActions = (item: WorkloadIdentityFederation): RowActionItem[] => [
    {
      key: "edit",
      label: "Edit",
      icon: Pencil,
      onSelect: () => openEdit(item),
    },
    {
      key: "delete",
      label: "Delete",
      icon: Trash2,
      destructive: true,
      onSelect: async () => {
        try {
          await deleteMutation.mutateAsync(item.workload_identity_federation_uuid)
          showSuccess("Federation deleted")
        } catch (err) {
          showError(err)
        }
      },
      confirm: {
        title: "Delete federation",
        description: "This cannot be undone.",
        confirmText: "Delete",
      },
    },
  ]

  const items = data?.rows ?? []

  return (
    <DetailsContainer>
      <PageContainer>
        <div className="flex items-start justify-between gap-4">
          <PageHeader
            title="Workload Identity"
            description="Manage OIDC federation configurations for machine workload authentication."
            icon={Network}
          />
          <Button onClick={openCreate} className="shrink-0">
            <Plus className="mr-2 size-4" />
            New Federation
          </Button>
        </div>

        {isLoading && <ListSkeleton />}

        {!isLoading && items.length === 0 && (
          <EmptyState
            icon={Network}
            title="No workload identity federations"
            description="Create a federation to allow OIDC workload tokens to exchange for tenant access tokens."
            action={
              <Button onClick={openCreate}>
                <Plus className="mr-2 size-4" />
                New Federation
              </Button>
            }
          />
        )}

        {items.length > 0 && (
          <div className="space-y-3">
            {items.map((item) => (
              <Card key={item.workload_identity_federation_uuid}>
                <CardContent className="flex items-start justify-between gap-3 p-4">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{item.name}</span>
                      <Badge
                        variant={item.is_active ? "default" : "secondary"}
                        className="text-xs"
                      >
                        {item.is_active ? "Active" : "Inactive"}
                      </Badge>
                    </div>
                    {item.description && (
                      <p className="text-sm text-muted-foreground">
                        {item.description}
                      </p>
                    )}
                    <p className="font-mono text-xs text-muted-foreground">
                      {item.issuer_url}
                    </p>
                    <div className="flex flex-wrap gap-2 pt-1 text-xs text-muted-foreground">
                      <span>
                        Audience: <code>{item.audience}</code>
                      </span>
                      <span>
                        Subject claim: <code>{item.subject_claim}</code>
                      </span>
                      <span>
                        Pattern: <code>{item.subject_pattern}</code>
                      </span>
                      <span>Created {format(new Date(item.created_at), "PP")}</span>
                    </div>
                  </div>
                  <RowActions items={getRowActions(item)} />
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </PageContainer>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {editing ? "Edit Federation" : "New Workload Identity Federation"}
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <Controller
              name="client_uuid"
              control={control}
              render={({ field }) => (
                <FormSelectField
                  label="Client"
                  placeholder="Select a client"
                  options={clientOptions}
                  value={field.value}
                  onValueChange={field.onChange}
                  required
                  disabled={isSubmitting || !!editing}
                  error={errors.client_uuid?.message}
                  description={editing ? "Client cannot be changed after creation" : undefined}
                />
              )}
            />
            <FormInputField
              label="Name"
              required
              disabled={isSubmitting}
              error={errors.name?.message}
              {...register("name")}
            />
            <FormInputField
              label="Description"
              disabled={isSubmitting}
              error={errors.description?.message}
              {...register("description")}
            />
            <FormUrlField
              label="Issuer URL"
              placeholder="https://token.example.com"
              required
              disabled={isSubmitting}
              error={errors.issuer_url?.message}
              {...register("issuer_url")}
            />
            <FormInputField
              label="Audience"
              required
              disabled={isSubmitting}
              error={errors.audience?.message}
              {...register("audience")}
            />
            <div className="grid grid-cols-2 gap-4">
              <FormInputField
                label="Subject claim"
                placeholder="sub"
                disabled={isSubmitting}
                error={errors.subject_claim?.message}
                {...register("subject_claim")}
              />
              <FormInputField
                label="Subject pattern"
                placeholder="service-*"
                required
                disabled={isSubmitting}
                error={errors.subject_pattern?.message}
                {...register("subject_pattern")}
              />
            </div>
            <FormInputField
              label="Allowed scopes"
              placeholder="read:users, write:clients"
              description="Comma-separated list"
              disabled={isSubmitting}
              {...register("allowed_scopes")}
            />
            <div>
              <label className="text-sm font-medium">
                Attribute mapping (JSON)
              </label>
              <textarea
                className="mt-1 w-full rounded-md border bg-background px-3 py-2 font-mono text-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                rows={4}
                placeholder='{"email": "sub"}'
                disabled={isSubmitting}
                {...register("attribute_mapping")}
              />
            </div>
            <Controller
              name="is_active"
              control={control}
              render={({ field }) => (
                <FormSwitchField
                  label="Active"
                  checked={!!field.value}
                  onCheckedChange={field.onChange}
                  disabled={isSubmitting}
                />
              )}
            />
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setDialogOpen(false)}
              >
                Cancel
              </Button>
              <FormSubmitButton
                isSubmitting={isSubmitting}
                submitText={editing ? "Update" : "Create"}
              />
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </DetailsContainer>
  )
}
