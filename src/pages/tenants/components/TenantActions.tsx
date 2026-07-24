import { Eye, Edit, Trash2, Play, Pause, Ban, type LucideIcon } from "lucide-react"
import { useNavigate } from "react-router-dom"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useDeleteTenant, useUpdateTenantStatus } from "@/hooks/useTenants"
import { useToast } from "@/hooks/useToast"
import type { TenantEntity, TenantStatus } from "@/services/api/tenants/types"

interface TenantActionsProps {
  tenant: TenantEntity
}

interface StatusAction {
  status: TenantStatus
  label: string
  title: string
  description: string
  icon: LucideIcon
}

const STATUS_ACTIONS: Record<TenantStatus, StatusAction[]> = {
  inactive: [
    {
      status: "active",
      label: "Activate Tenant",
      title: "Activate Tenant",
      description: "Are you sure you want to activate this tenant? Users will be able to sign in.",
      icon: Play,
    },
  ],
  active: [
    {
      status: "inactive",
      label: "Deactivate Tenant",
      title: "Deactivate Tenant",
      description: "Are you sure you want to deactivate this tenant? Users will no longer be able to sign in.",
      icon: Pause,
    },
    {
      status: "suspended",
      label: "Suspend Tenant",
      title: "Suspend Tenant",
      description: "Are you sure you want to suspend this tenant? All active sessions will be terminated.",
      icon: Ban,
    },
  ],
  suspended: [
    {
      status: "active",
      label: "Activate Tenant",
      title: "Activate Tenant",
      description: "Are you sure you want to reactivate this suspended tenant?",
      icon: Play,
    },
  ],
}

export function TenantActions({ tenant }: TenantActionsProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const updateStatusMutation = useUpdateTenantStatus()
  const deleteTenantMutation = useDeleteTenant()

  const changeStatus = async (status: TenantStatus) => {
    try {
      await updateStatusMutation.mutateAsync({ tenantId: tenant.tenant_id, status })
      showSuccess(`Tenant status updated to ${status}`)
    } catch (error) {
      showError(error)
    }
  }

  const items: RowActionItem[] = [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () => navigate(`/tenants/${tenant.tenant_id}`),
    },
    {
      key: "edit",
      label: "Edit Tenant",
      icon: Edit,
      onSelect: () => navigate(`/tenants/${tenant.tenant_id}/edit`),
    },
    ...STATUS_ACTIONS[tenant.status].map(
      (action): RowActionItem => ({
        key: `status-${action.status}`,
        label: action.label,
        icon: action.icon,
        // Deactivate/Suspend lock users out → destructive (red). Activate is
        // restorative → default.
        destructive: action.status !== "active",
        onSelect: () => changeStatus(action.status),
        confirm: {
          title: action.title,
          description: action.description,
          confirmText: action.label,
        },
      }),
    ),
    {
      key: "delete",
      label: "Delete Tenant",
      icon: Trash2,
      destructive: true,
      separatorBefore: true,
      onSelect: async () => {
        try {
          await deleteTenantMutation.mutateAsync(tenant.tenant_id)
          showSuccess("Tenant deleted successfully")
        } catch (error) {
          showError(error)
        }
      },
      confirm: {
        title: "Delete Tenant",
        description: "This will permanently delete this tenant, including all users, roles, and configuration.",
        destructive: true,
        itemName: tenant.display_name,
      },
    },
  ]

  return <RowActions items={items} />
}
