import { useNavigate } from "react-router-dom"
import { Eye, Edit, Trash2, Play, Pause, Ban, type LucideIcon } from "lucide-react"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useUpdateUserStatus, useDeleteUser } from "@/hooks/useUsers"
import { useToast } from "@/hooks/useToast"
import type { User, UserStatus } from "@/services/api/users/types"

interface UserActionsProps {
  user: User
}

interface StatusAction {
  status: UserStatus
  label: string
  title: string
  description: string
  icon: LucideIcon
}

// Available status transitions per current status (exhaustive over UserStatus).
const STATUS_ACTIONS: Record<UserStatus, StatusAction[]> = {
  inactive: [
    {
      status: "active",
      label: "Activate User",
      title: "Activate User",
      description:
        "Are you sure you want to activate this user? They will be able to sign in and access the system.",
      icon: Play,
    },
  ],
  active: [
    {
      status: "inactive",
      label: "Deactivate User",
      title: "Deactivate User",
      description: "Are you sure you want to deactivate this user? They will no longer be able to sign in.",
      icon: Pause,
    },
    {
      status: "suspended",
      label: "Suspend User",
      title: "Suspend User",
      description:
        "Are you sure you want to suspend this user? This is typically used for security concerns or policy violations. They will be immediately logged out and cannot sign in.",
      icon: Ban,
    },
  ],
  suspended: [
    {
      status: "active",
      label: "Activate User",
      title: "Activate User",
      description: "Are you sure you want to activate this suspended user? They will be able to sign in again.",
      icon: Play,
    },
  ],
  pending: [
    {
      status: "active",
      label: "Activate User",
      title: "Activate User",
      description:
        "Are you sure you want to activate this pending user? They will be able to sign in and access the system.",
      icon: Play,
    },
  ],
}

export function UserActions({ user }: UserActionsProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const updateStatusMutation = useUpdateUserStatus()
  const deleteUserMutation = useDeleteUser()

  const changeStatus = async (status: UserStatus) => {
    try {
      await updateStatusMutation.mutateAsync({ userId: user.user_id, data: { status } })
      showSuccess(`User status updated to ${status}`)
    } catch (error) {
      showError(error)
    }
  }

  const items: RowActionItem[] = [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () => navigate(`/users/${user.user_id}`),
    },
    {
      key: "edit",
      label: "Edit User",
      icon: Edit,
      onSelect: () => navigate(`/users/${user.user_id}/edit`),
    },
    // Indexed directly, an unrecognised status (backend ships a new one before
        // the frontend union is updated) yields undefined and .map throws — taking
        // down the entire Users table, not just this row. Siblings like
        // ServiceActions already use the tolerant form.
        ...(STATUS_ACTIONS[user.status] ?? []).map(
      (action): RowActionItem => ({
        key: `status-${action.status}`,
        label: action.label,
        icon: action.icon,
        // Suspend/Deactivate lock the user out → destructive (red). Activate is
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
      label: "Delete User",
      icon: Trash2,
      destructive: true,
      separatorBefore: true,
      onSelect: async () => {
        try {
          await deleteUserMutation.mutateAsync(user.user_id)
          showSuccess("User deleted successfully")
        } catch (error) {
          showError(error)
        }
      },
      confirm: {
        title: "Delete User",
        description:
          "This will permanently delete this user account, including their profile, authentication data, and activity history.",
        destructive: true,
        itemName: user.fullname || user.username,
      },
    },
  ]

  return <RowActions items={items} />
}
