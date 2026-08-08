import { useNavigate } from "react-router-dom"
import { Edit, Eye, Trash2, Play, XCircle, type LucideIcon } from "lucide-react"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useDeleteApi, useUpdateApiStatus } from "@/hooks/useApis"
import { useToast } from "@/hooks/useToast"
import type { Api, ApiStatus } from "@/services/api/api/types"

interface ApiActionsProps {
  api: Api
}

interface StatusAction {
  status: ApiStatus
  label: string
  title: string
  description: string
  icon: LucideIcon
}

const STATUS_ACTIONS: Record<ApiStatus, StatusAction> = {
  active: {
    status: "active",
    label: "Activate API",
    title: "Activate API",
    description: "Are you sure you want to activate this API? Its permissions will be available for use.",
    icon: Play,
  },
  inactive: {
    status: "inactive",
    label: "Deactivate API",
    title: "Deactivate API",
    description: "Are you sure you want to deactivate this API? Its permissions will no longer be available.",
    icon: XCircle,
  },
}

export function ApiActions({ api }: ApiActionsProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const deleteApiMutation = useDeleteApi()
  const updateStatusMutation = useUpdateApiStatus()

  const changeStatus = async (status: ApiStatus) => {
    try {
      await updateStatusMutation.mutateAsync({
        apiId: api.api_id,
        data: { status },
      })
      showSuccess(`API status updated to ${status}`)
    } catch (error) {
      showError(error)
    }
  }

  // Availability mirrors the backend rules: system APIs cannot be edited,
  // change status, or be deleted.
  const statusActions = api.is_system
    ? []
    : Object.values(STATUS_ACTIONS)
        .filter((action) => action.status !== api.status)
        .map((action) => ({
          key: `status-${action.status}`,
          label: action.label,
          icon: action.icon,
          // Deactivate takes the API out of use → destructive (red confirm).
          // Activate is restorative → default.
          destructive: action.status !== "active",
          onSelect: () => changeStatus(action.status),
          confirm: {
            title: action.title,
            description: action.description,
            confirmText: action.label,
          },
        }) satisfies RowActionItem)

  const items: RowActionItem[] = [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () => navigate(`/apis/${api.api_id}`),
    },
    ...(!api.is_system
      ? [
          {
            key: "edit",
            label: "Edit API",
            icon: Edit,
            onSelect: () => navigate(`/apis/${api.api_id}/edit`),
          } satisfies RowActionItem,
        ]
      : []),
    ...statusActions,
    ...(!api.is_system
      ? [
          {
            key: "delete",
            label: "Delete API",
            icon: Trash2,
            destructive: true,
            separatorBefore: true,
            onSelect: async () => {
              try {
                await deleteApiMutation.mutateAsync(api.api_id)
                showSuccess("API deleted successfully")
              } catch (error) {
                showError(error)
              }
            },
            confirm: {
              title: "Delete API",
              description:
                "This will permanently delete this API and remove all associated permissions and configurations.",
              destructive: true,
              itemName: api.name,
            },
          } satisfies RowActionItem,
        ]
      : []),
  ]

  return <RowActions items={items} />
}
