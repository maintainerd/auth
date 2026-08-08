import { useNavigate } from "react-router-dom"
import { Eye, Edit, Play, Pause, Trash2 } from "lucide-react"
import { RowActions, type RowActionItem } from "@/components/data-table"
import type { Policy, PolicyStatus } from "@/services/api/policies/types"
import { useDeletePolicy, useUpdatePolicyStatus } from "@/hooks/usePolicies"
import { useToast } from "@/hooks/useToast"

interface PolicyActionsProps {
  policy: Policy
}

export function PolicyActions({ policy }: PolicyActionsProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const deletePolicyMutation = useDeletePolicy()
  const updateStatusMutation = useUpdatePolicyStatus()

  const changeStatus = async (status: PolicyStatus) => {
    try {
      await updateStatusMutation.mutateAsync({ policyId: policy.policy_id, data: { status } })
      showSuccess(`Policy status updated to ${status}`)
    } catch (error) {
      showError(error)
    }
  }

  // Availability mirrors the backend rules: system policies cannot be edited,
  // change status, or be deleted.
  const isActive = policy.status === "active"
  const canMutate = !policy.is_system

  const items: RowActionItem[] = [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () => navigate(`/policies/${policy.policy_id}`),
    },
    ...(canMutate
      ? [
          {
            key: "edit",
            label: "Edit Policy",
            icon: Edit,
            onSelect: () => navigate(`/policies/${policy.policy_id}/edit`),
          } satisfies RowActionItem,
        ]
      : []),
    ...(canMutate
      ? isActive
        ? [
            {
              key: "deactivate",
              label: "Deactivate Policy",
              icon: Pause,
              destructive: true,
              onSelect: () => changeStatus("inactive"),
              confirm: {
                title: "Deactivate Policy",
                description:
                  "Are you sure you want to deactivate this policy? Services using it will no longer be governed by its statements.",
                confirmText: "Deactivate",
              },
            } satisfies RowActionItem,
          ]
        : [
            {
              key: "activate",
              label: "Activate Policy",
              icon: Play,
              onSelect: () => changeStatus("active"),
              confirm: {
                title: "Activate Policy",
                description: "Are you sure you want to activate this policy?",
                confirmText: "Activate",
              },
            } satisfies RowActionItem,
          ]
      : []),
    ...(canMutate
      ? [
          {
            key: "delete",
            label: "Delete Policy",
            icon: Trash2,
            destructive: true,
            separatorBefore: true,
            onSelect: async () => {
              try {
                await deletePolicyMutation.mutateAsync(policy.policy_id)
                showSuccess("Policy deleted successfully")
              } catch (error) {
                showError(error)
              }
            },
            confirm: {
              title: "Delete Policy",
              description: "This action cannot be undone. This will permanently delete the policy and all associated data.",
              destructive: true,
              itemName: policy.name,
            },
          } satisfies RowActionItem,
        ]
      : []),
  ]

  return <RowActions items={items} />
}
