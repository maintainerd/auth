import { useNavigate } from "react-router-dom"
import { Eye, Edit, Trash2, Play, Pause } from "lucide-react"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useDeleteIdentityProvider, useUpdateIdentityProviderStatus } from "@/hooks/useIdentityProviders"
import { useToast } from "@/hooks/useToast"
import type { IdentityProvider, IdentityProviderStatus } from "@/services/api/identity-providers/types"

interface IdentityProviderActionsProps {
  provider: IdentityProvider
}

export function IdentityProviderActions({ provider }: IdentityProviderActionsProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const updateStatusMutation = useUpdateIdentityProviderStatus()
  const deleteProviderMutation = useDeleteIdentityProvider()

  const changeStatus = async (status: IdentityProviderStatus) => {
    try {
      await updateStatusMutation.mutateAsync({
        identityProviderId: provider.identity_provider_id,
        data: { status }
      })
      showSuccess(`Identity provider status updated to ${status}`)
    } catch (error) {
      showError(error)
    }
  }

  // Availability mirrors backend rules: system identity providers can't change
  // status or be deleted; the default provider also can't be deactivated/deleted.
  const isActive = provider.status === "active"
  const canActivate = !provider.is_system && !isActive
  const canDeactivate = !provider.is_system && isActive && !provider.is_default
  const canDelete = !provider.is_system && !provider.is_default

  const items: RowActionItem[] = [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () => navigate(`/providers/identity/${provider.identity_provider_id}`),
    },
    {
      key: "edit",
      label: "Edit Provider",
      icon: Edit,
      onSelect: () => navigate(`/providers/identity/${provider.identity_provider_id}/edit`),
    },
    ...(canActivate
      ? [
          {
            key: "activate",
            label: "Activate Provider",
            icon: Play,
            onSelect: () => changeStatus("active"),
            confirm: {
              title: "Activate Identity Provider",
              description: "Are you sure you want to activate this identity provider? Users will be able to authenticate through it.",
              confirmText: "Activate",
            },
          } satisfies RowActionItem,
        ]
      : []),
    ...(canDeactivate
      ? [
          {
            key: "deactivate",
            label: "Deactivate Provider",
            icon: Pause,
            destructive: true,
            onSelect: () => changeStatus("inactive"),
            confirm: {
              title: "Deactivate Identity Provider",
              description: "Are you sure you want to deactivate this identity provider? Users will not be able to authenticate through it.",
              confirmText: "Deactivate",
            },
          } satisfies RowActionItem,
        ]
      : []),
    ...(canDelete
      ? [
          {
            key: "delete",
            label: "Delete Provider",
            icon: Trash2,
            destructive: true,
            separatorBefore: true,
            onSelect: async () => {
              try {
                await deleteProviderMutation.mutateAsync(provider.identity_provider_id)
                showSuccess("Identity provider deleted successfully")
              } catch (error) {
                showError(error)
              }
            },
            confirm: {
              title: "Delete Identity Provider",
              description: "This action cannot be undone. This will permanently delete the identity provider and all associated data.",
              destructive: true,
              itemName: provider.name,
            },
          } satisfies RowActionItem,
        ]
      : []),
  ]

  return <RowActions items={items} />
}
