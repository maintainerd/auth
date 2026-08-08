import { useNavigate } from "react-router-dom"
import { Eye, Edit, Trash2, Play, Pause } from "lucide-react"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useDeleteRole, useUpdateRoleStatus } from "@/hooks/useRoles"
import { useToast } from "@/hooks/useToast"
import type { Role, RoleStatus } from "@/services/api/roles/types"

interface RoleActionsProps {
  role: Role
}

export function RoleActions({ role }: RoleActionsProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const updateStatusMutation = useUpdateRoleStatus()
  const deleteRoleMutation = useDeleteRole()

  const changeStatus = async (status: RoleStatus) => {
    try {
      await updateStatusMutation.mutateAsync({ roleId: role.role_id, data: { status } })
      showSuccess(`Role status updated to ${status}`)
    } catch (error) {
      showError(error)
    }
  }

  // Availability mirrors the backend rules: system roles can't change status or
  // be deleted; the default (registration) role can't be deactivated or deleted.
  const isActive = role.status === "active"
  const canActivate = !role.is_system && !isActive
  const canDeactivate = !role.is_system && isActive && !role.is_default
  const canDelete = !role.is_system && !role.is_default

  const items: RowActionItem[] = [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () => navigate(`/roles/${role.role_id}`),
    },
    {
      key: "edit",
      label: "Edit Role",
      icon: Edit,
      onSelect: () => navigate(`/roles/${role.role_id}/edit`),
    },
    ...(canActivate
      ? [
          {
            key: "activate",
            label: "Activate Role",
            icon: Play,
            onSelect: () => changeStatus("active"),
            confirm: {
              title: "Activate Role",
              description: "Are you sure you want to activate this role?",
              confirmText: "Activate",
            },
          } satisfies RowActionItem,
        ]
      : []),
    ...(canDeactivate
      ? [
          {
            key: "deactivate",
            label: "Deactivate Role",
            icon: Pause,
            destructive: true,
            onSelect: () => changeStatus("inactive"),
            confirm: {
              title: "Deactivate Role",
              description: "Are you sure you want to deactivate this role? It can no longer be assigned to users.",
              confirmText: "Deactivate",
            },
          } satisfies RowActionItem,
        ]
      : []),
    ...(canDelete
      ? [
          {
            key: "delete",
            label: "Delete Role",
            icon: Trash2,
            destructive: true,
            separatorBefore: true,
            onSelect: async () => {
              try {
                await deleteRoleMutation.mutateAsync(role.role_id)
                showSuccess("Role deleted successfully")
              } catch (error) {
                showError(error)
              }
            },
            confirm: {
              title: "Delete Role",
              description: "This action cannot be undone. This will permanently delete the role and all associated data.",
              destructive: true,
              itemName: role.name,
            },
          } satisfies RowActionItem,
        ]
      : []),
  ]

  return <RowActions items={items} />
}
