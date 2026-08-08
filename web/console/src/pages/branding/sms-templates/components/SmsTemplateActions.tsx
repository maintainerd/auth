import { useLocation, useNavigate } from "react-router-dom"
import { Eye, Edit, Play, Pause } from "lucide-react"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useUpdateSmsTemplateStatus } from "@/hooks/useSmsTemplates"
import { useToast } from "@/hooks/useToast"
import type { SmsTemplate } from "@/services/api/sms-templates/types"

interface SmsTemplateActionsProps {
  template: SmsTemplate
}

export function SmsTemplateActions({ template }: SmsTemplateActionsProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError } = useToast()
  const updateStatusMutation = useUpdateSmsTemplateStatus()
  const from = `${location.pathname}${location.search}` || "/branding?tab=sms-templates"
  const listState = { from, backLabel: "Back to SMS Templates" }

  const changeStatus = async (status: "active" | "inactive") => {
    try {
      await updateStatusMutation.mutateAsync({
        id: template.smsTemplateId,
        data: { status },
      })
      showSuccess(
        `SMS template ${status === "active" ? "activated" : "deactivated"} successfully`,
      )
    } catch (error) {
      showError(error)
    }
  }

  const isActive = template.status === "active"
  const canEditStatus = !template.isSystem

  const items: RowActionItem[] = [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () => navigate(`/branding/sms-templates/${template.smsTemplateId}`, { state: listState }),
    },
    {
      key: "edit",
      label: "Edit Template",
      icon: Edit,
      onSelect: () => navigate(`/branding/sms-templates/${template.smsTemplateId}/edit`, { state: listState }),
    },
    ...(canEditStatus && isActive
      ? [
          {
            key: "deactivate",
            label: "Deactivate Template",
            icon: Pause,
            destructive: true,
            onSelect: () => changeStatus("inactive"),
            confirm: {
              title: "Deactivate SMS Template",
              description:
                "Are you sure you want to deactivate this SMS template? It will no longer be used for SMS delivery.",
              confirmText: "Deactivate Template",
            },
          } satisfies RowActionItem,
        ]
      : []),
    ...(canEditStatus && !isActive
      ? [
          {
            key: "activate",
            label: "Activate Template",
            icon: Play,
            onSelect: () => changeStatus("active"),
            confirm: {
              title: "Activate SMS Template",
              description:
                "Are you sure you want to activate this SMS template? It will be used for SMS delivery.",
              confirmText: "Activate Template",
            },
          } satisfies RowActionItem,
        ]
      : []),
  ]

  return <RowActions items={items} />
}
