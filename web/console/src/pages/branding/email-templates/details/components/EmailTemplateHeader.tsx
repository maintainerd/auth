import { useState } from "react"
import { useLocation, useNavigate } from "react-router-dom"
import { Edit, MoreVertical, Play, Pause, Mail, Activity, CalendarDays } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { ConfirmationDialog } from "@/components/dialog"
import { DetailHeaderCard, StatusBadge, type DetailAttribute } from "@/components/details"
import { useUpdateEmailTemplateStatus } from "@/hooks/useEmailTemplates"
import { useToast } from "@/hooks/useToast"
import type { EmailTemplate, EmailTemplateStatus } from "@/services/api/email-templates/types"

interface EmailTemplateHeaderProps {
  template: EmailTemplate
  templateId: string
}

interface PendingStatusAction {
  status: EmailTemplateStatus
  title: string
  description: string
}

export function EmailTemplateHeader({ template, templateId }: EmailTemplateHeaderProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const { showError } = useToast()
  const updateStatusMutation = useUpdateEmailTemplateStatus()

  const [showStatusDialog, setShowStatusDialog] = useState(false)
  const [pendingStatusAction, setPendingStatusAction] = useState<PendingStatusAction | null>(null)

  const handleStatusChange = (status: EmailTemplateStatus, title: string, description: string) => {
    setPendingStatusAction({ status, title, description })
    setShowStatusDialog(true)
  }

  const handleConfirmStatusChange = async () => {
    if (!pendingStatusAction) return
    try {
      await updateStatusMutation.mutateAsync({
        id: templateId,
        data: { status: pendingStatusAction.status }
      })
      setShowStatusDialog(false)
      setPendingStatusAction(null)
    } catch (error) {
      showError(error)
    }
  }

  const canEditStatus = !template.isSystem
  const detailPath = `${location.pathname}${location.search}`

  const attributes: DetailAttribute[] = [
    { icon: Activity, label: "Subject", value: template.subject },
    { icon: CalendarDays, label: "Created", value: formatDistanceToNow(new Date(template.createdAt), { addSuffix: true }) },
    { icon: CalendarDays, label: "Updated", value: formatDistanceToNow(new Date(template.updatedAt), { addSuffix: true }) },
  ]

  return (
    <>
      <DetailHeaderCard
        leading={
          <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
            <Mail className="size-6" />
          </div>
        }
        title={template.name}
        badge={<StatusBadge status={template.status} />}
        subtitle={template.subject}
        attributes={attributes}
        actions={
          <>
            <Button
              data-md-details-edit-button
              variant="outline"
              size="sm"
              className="h-9 gap-2"
              onClick={() =>
                navigate(`/branding/email-templates/${templateId}/edit`, {
                  state: { from: detailPath, backLabel: "Back to Email Template Details" },
                })
              }
            >
              <Edit className="size-4" />
              Edit
            </Button>
            {canEditStatus && (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button data-md-details-menu-button variant="outline" size="sm" className="h-9 w-9 p-0">
                    <span className="sr-only">Open actions</span>
                    <MoreVertical className="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  {template.status === 'inactive' ? (
                    <DropdownMenuItem
                      onClick={() =>
                        handleStatusChange(
                          'active',
                          'Activate Email Template',
                          'Are you sure you want to activate this email template? It will be used for email delivery.',
                        )
                      }
                    >
                      <Play className="mr-2 size-4" />
                      Activate Template
                    </DropdownMenuItem>
                  ) : (
                    <DropdownMenuItem
                      onClick={() =>
                        handleStatusChange(
                          'inactive',
                          'Deactivate Email Template',
                          'Are you sure you want to deactivate this email template? It will no longer be used for email delivery.',
                        )
                      }
                      className="text-destructive focus:text-destructive"
                    >
                      <Pause className="mr-2 size-4" />
                      Deactivate Template
                    </DropdownMenuItem>
                  )}
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </>
        }
      />

      <ConfirmationDialog
        open={showStatusDialog}
        onOpenChange={setShowStatusDialog}
        onConfirm={handleConfirmStatusChange}
        title={pendingStatusAction?.title || "Change Status"}
        description={pendingStatusAction?.description || "Are you sure you want to change the template status?"}
        confirmText={pendingStatusAction?.status === "active" ? "Activate" : "Deactivate"}
        cancelText="Cancel"
        variant={pendingStatusAction?.status === "inactive" ? "destructive" : "default"}
        isLoading={updateStatusMutation.isPending}
      />
    </>
  )
}
