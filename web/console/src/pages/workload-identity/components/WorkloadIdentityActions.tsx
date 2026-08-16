import { useNavigate } from "react-router-dom"
import { Eye, Edit, Trash2, Play, Pause } from "lucide-react"
import { RowActions, type RowActionItem } from "@/components/data-table"
import {
  useUpdateWorkloadIdentity,
  useDeleteWorkloadIdentity,
} from "@/hooks/useWorkloadIdentity"
import { useToast } from "@/hooks/useToast"
import type { WorkloadIdentityFederation } from "@/services/api/workload-identity/types"

interface WorkloadIdentityActionsProps {
  federation: WorkloadIdentityFederation
}

export function WorkloadIdentityActions({ federation }: WorkloadIdentityActionsProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const updateMutation = useUpdateWorkloadIdentity()
  const deleteMutation = useDeleteWorkloadIdentity()

  const federationId = federation.workload_identity_federation_id

  /**
   * Activation is a full-replace PUT, so every field has to be resent — omitting one
   * would clear it server-side.
   */
  const setActive = async (isActive: boolean) => {
    try {
      await updateMutation.mutateAsync({
        federationId,
        data: {
          name: federation.name,
          description: federation.description,
          issuer_url: federation.issuer_url,
          audience: federation.audience,
          subject_claim: federation.subject_claim,
          subject_pattern: federation.subject_pattern,
          allowed_scopes: federation.allowed_scopes,
          attribute_mapping: federation.attribute_mapping,
          is_active: isActive,
        },
      })
      showSuccess(`Federation ${isActive ? "activated" : "deactivated"}`)
    } catch (error) {
      showError(error)
    }
  }

  const items: RowActionItem[] = [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () => navigate(`/workload-identity/${federationId}`),
    },
    {
      key: "edit",
      label: "Edit Federation",
      icon: Edit,
      onSelect: () => navigate(`/workload-identity/${federationId}/edit`),
    },
    ...(federation.is_active
      ? [
          {
            key: "deactivate",
            label: "Deactivate Federation",
            icon: Pause,
            // Red, like every other deactivate in the app: it locks a live workload
            // out. Activate is not destructive.
            destructive: true,
            onSelect: () => setActive(false),
            confirm: {
              title: "Deactivate Federation",
              // Deactivation is the kill switch for a keyless credential, so say
              // plainly what stops working.
              description:
                "Workloads matching this federation will immediately stop being able to exchange their tokens for access tokens. Existing access tokens remain valid until they expire.",
              confirmText: "Deactivate",
            },
          } satisfies RowActionItem,
        ]
      : [
          {
            key: "activate",
            label: "Activate Federation",
            icon: Play,
            onSelect: () => setActive(true),
            confirm: {
              title: "Activate Federation",
              description:
                "Workloads matching this federation's subject pattern will be able to exchange their tokens for access tokens scoped to the mapped client.",
              confirmText: "Activate",
            },
          } satisfies RowActionItem,
        ]),
    {
      key: "delete",
      label: "Delete Federation",
      icon: Trash2,
      destructive: true,
      separatorBefore: true,
      onSelect: async () => {
        try {
          await deleteMutation.mutateAsync(federationId)
          showSuccess("Federation deleted successfully")
        } catch (error) {
          showError(error)
        }
      },
      confirm: {
        title: "Delete Federation",
        description:
          "This action cannot be undone. Any workload relying on this federation will immediately lose its ability to authenticate.",
        // Type-to-confirm: this instantly breaks machine authentication for a live
        // workload, with no credential to fall back on.
        destructive: true,
        itemName: federation.name,
      },
    },
  ]

  return <RowActions items={items} />
}
