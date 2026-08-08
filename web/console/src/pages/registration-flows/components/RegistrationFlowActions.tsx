import { useNavigate } from "react-router-dom"
import { Eye, Edit, Trash2, Play, Pause } from "lucide-react"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useDeleteRegistrationFlow, useUpdateRegistrationFlowStatus } from "@/hooks/useRegistrationFlows"
import { useToast } from "@/hooks/useToast"
import type { RegistrationFlow, RegistrationFlowStatus } from "@/services/api/registration-flows/types"

interface RegistrationFlowActionsProps {
  registrationFlow: RegistrationFlow
}

export function RegistrationFlowActions({ registrationFlow }: RegistrationFlowActionsProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const updateStatusMutation = useUpdateRegistrationFlowStatus()
  const deleteMutation = useDeleteRegistrationFlow()

  const changeStatus = async (status: RegistrationFlowStatus) => {
    try {
      await updateStatusMutation.mutateAsync({ registrationFlowId: registrationFlow.registration_flow_id, data: { status } })
      showSuccess(`Registration flow ${status === "active" ? "activated" : "deactivated"} successfully`)
    } catch (error) {
      showError(error)
    }
  }

  // Availability mirrors the backend rules: system flows can't change status or be deleted.
  const isActive = registrationFlow.status === "active"
  const canActivate = !registrationFlow.is_system && !isActive
  const canDeactivate = !registrationFlow.is_system && isActive
  const canDelete = !registrationFlow.is_system

  const items: RowActionItem[] = [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () => navigate(`/registration-flows/${registrationFlow.registration_flow_id}`),
    },
    {
      key: "edit",
      label: "Edit Registration Flow",
      icon: Edit,
      onSelect: () => navigate(`/registration-flows/${registrationFlow.registration_flow_id}/edit`),
    },
    ...(canActivate
      ? [
          {
            key: "activate",
            label: "Activate Registration Flow",
            icon: Play,
            onSelect: () => changeStatus("active"),
            confirm: {
              title: "Activate Registration Flow",
              description: "Are you sure you want to activate this registration flow?",
              confirmText: "Activate",
            },
          } satisfies RowActionItem,
        ]
      : []),
    ...(canDeactivate
      ? [
          {
            key: "deactivate",
            label: "Deactivate Registration Flow",
            icon: Pause,
            destructive: true,
            onSelect: () => changeStatus("inactive"),
            confirm: {
              title: "Deactivate Registration Flow",
              description: "Are you sure you want to deactivate this registration flow?",
              confirmText: "Deactivate",
            },
          } satisfies RowActionItem,
        ]
      : []),
    ...(canDelete
      ? [
          {
            key: "delete",
            label: "Delete Registration Flow",
            icon: Trash2,
            destructive: true,
            separatorBefore: true,
            onSelect: async () => {
              try {
                await deleteMutation.mutateAsync(registrationFlow.registration_flow_id)
                showSuccess("Registration flow deleted successfully")
              } catch (error) {
                showError(error)
              }
            },
            confirm: {
              title: "Delete Registration Flow",
              description: "This action cannot be undone. This will permanently delete the registration flow.",
              destructive: true,
              itemName: registrationFlow.name,
            },
          } satisfies RowActionItem,
        ]
      : []),
  ]

  return <RowActions items={items} />
}
