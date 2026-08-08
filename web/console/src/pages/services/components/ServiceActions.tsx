import { useNavigate } from "react-router-dom"
import { Archive, Edit, Eye, Pause, Play, Trash2, XCircle, type LucideIcon } from "lucide-react"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useDeleteService, useUpdateServiceStatus } from "@/hooks/useServices"
import { useToast } from "@/hooks/useToast"
import type { Service, ServiceStatus } from "@/services/api/services/types"

interface ServiceActionsProps {
  service: Service
}

interface StatusAction {
  status: ServiceStatus
  label: string
  title: string
  description: string
  icon: LucideIcon
}

const STATUS_ACTIONS: Record<ServiceStatus, StatusAction> = {
  active: {
    status: "active",
    label: "Activate Service",
    title: "Activate Service",
    description: "Are you sure you want to activate this service? Its APIs and policies will be available for use.",
    icon: Play,
  },
  maintenance: {
    status: "maintenance",
    label: "Set Maintenance",
    title: "Set Maintenance Mode",
    description: "Are you sure you want to put this service into maintenance mode?",
    icon: Pause,
  },
  deprecated: {
    status: "deprecated",
    label: "Deprecate Service",
    title: "Deprecate Service",
    description: "Are you sure you want to mark this service as deprecated?",
    icon: Archive,
  },
  inactive: {
    status: "inactive",
    label: "Deactivate Service",
    title: "Deactivate Service",
    description: "Are you sure you want to deactivate this service? Its APIs and policies will no longer be available.",
    icon: XCircle,
  },
}

export function ServiceActions({ service }: ServiceActionsProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const deleteServiceMutation = useDeleteService()
  const updateStatusMutation = useUpdateServiceStatus()

  const changeStatus = async (status: ServiceStatus) => {
    try {
      await updateStatusMutation.mutateAsync({
        serviceId: service.service_id,
        data: { status },
      })
      showSuccess(`Service status updated to ${status}`)
    } catch (error) {
      showError(error)
    }
  }

  // Availability mirrors the backend rules: system services cannot be edited,
  // change status, or be deleted.
  const statusActions = service.is_system
    ? []
    : Object.values(STATUS_ACTIONS)
        .filter((action) => action.status !== service.status)
        .map((action) => ({
          key: `status-${action.status}`,
          label: action.label,
          icon: action.icon,
          // Everything except Activate takes the service (partly) out of use →
          // destructive (red confirm). Activate is restorative → default.
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
      onSelect: () => navigate(`/services/${service.service_id}`),
    },
    ...(!service.is_system
      ? [
          {
            key: "edit",
            label: "Edit Service",
            icon: Edit,
            onSelect: () => navigate(`/services/${service.service_id}/edit`),
          } satisfies RowActionItem,
        ]
      : []),
    ...statusActions,
    ...(!service.is_system
      ? [
          {
            key: "delete",
            label: "Delete Service",
            icon: Trash2,
            destructive: true,
            separatorBefore: true,
            onSelect: async () => {
              try {
                await deleteServiceMutation.mutateAsync(service.service_id)
                showSuccess("Service deleted successfully")
              } catch (error) {
                showError(error)
              }
            },
            confirm: {
              title: "Delete Service",
              description:
                "This will permanently delete this service and remove all associated APIs, policies, and relationships.",
              destructive: true,
              itemName: service.name,
            },
          } satisfies RowActionItem,
        ]
      : []),
  ]

  return <RowActions items={items} />
}
