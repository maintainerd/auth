import { useLocation, useNavigate } from "react-router-dom"
import { Eye, Edit, Play, Pause } from "lucide-react"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useUpdateEmailTemplateStatus } from "@/hooks/useEmailTemplates"
import { useToast } from "@/hooks/useToast"
import type { EmailTemplate } from "@/services/api/email-templates/types"

interface EmailTemplateActionsProps {
  template: EmailTemplate
}

export function EmailTemplateActions({ template }: EmailTemplateActionsProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError } = useToast()
  const updateStatusMutation = useUpdateEmailTemplateStatus()
  const from = `${location.pathname}${location.search}` || "/branding?tab=email-templates"
  const listState = { from, backLabel: "Back to Email Templates" }

  const changeStatus = async (status: "active" | "inactive") => {
    try {
      await updateStatusMutation.mutateAsync({
        id: template.emailTemplateId,
        data: { status },
      })
      showSuccess(
        `Email template ${status === "active" ? "activated" : "deactivated"} successfully`,
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
      onSelect: () => navigate(`/branding/email-templates/${template.emailTemplateId}`, { state: listState }),
    },
    {
      key: "edit",
      label: "Edit Template",
      icon: Edit,
      onSelect: () => navigate(`/branding/email-templates/${template.emailTemplateId}/edit`, { state: listState }),
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
              title: "Deactivate Email Template",
              description:
                "Are you sure you want to deactivate this email template? It will no longer be used for email delivery.",
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
              title: "Activate Email Template",
              description:
                "Are you sure you want to activate this email template? It will be used for email delivery.",
              confirmText: "Activate Template",
            },
          } satisfies RowActionItem,
        ]
      : []),
  ]

  return <RowActions items={items} />
}
