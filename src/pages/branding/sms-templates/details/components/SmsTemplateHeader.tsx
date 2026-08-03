import { useState } from "react"
import { useLocation, useNavigate } from "react-router-dom"
import { Edit, MoreVertical, Play, Pause, MessageSquare, CalendarDays } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { ConfirmationDialog } from "@/components/dialog"
import { DetailHeaderCard, StatusBadge, type DetailAttribute } from "@/components/details"
import { useUpdateSmsTemplateStatus } from "@/hooks/useSmsTemplates"
import { useToast } from "@/hooks/useToast"
import type { SmsTemplate, SmsTemplateStatus } from "@/services/api/sms-templates/types"

interface SmsTemplateHeaderProps { template: SmsTemplate; templateId: string }
interface PendingStatusAction { status: SmsTemplateStatus; title: string; description: string }

export function SmsTemplateHeader({ template, templateId }: SmsTemplateHeaderProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const { showError } = useToast()
  const updateStatusMutation = useUpdateSmsTemplateStatus()
  const [showStatusDialog, setShowStatusDialog] = useState(false)
  const [pendingStatusAction, setPendingStatusAction] = useState<PendingStatusAction | null>(null)

  const canEditStatus = !template.isSystem
  const detailPath = `${location.pathname}${location.search}`

  const attributes: DetailAttribute[] = [
    { icon: CalendarDays, label: "Created", value: formatDistanceToNow(new Date(template.createdAt), { addSuffix: true }) },
    { icon: CalendarDays, label: "Updated", value: formatDistanceToNow(new Date(template.updatedAt), { addSuffix: true }) },
  ]

  return (
    <>
      <DetailHeaderCard
        leading={
          <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
            <MessageSquare className="size-6" />
          </div>
        }
        title={template.name}
        badge={<StatusBadge status={template.status} />}
        subtitle={template.description}
        attributes={attributes}
        actions={
          <>
            <Button
              data-md-details-edit-button
              variant="outline"
              size="sm"
              className="h-9 gap-2"
              onClick={() =>
                navigate(`/branding/sms-templates/${templateId}/edit`, {
                  state: { from: detailPath, backLabel: "Back to SMS Template Details" },
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
                      onClick={() => {
                        setPendingStatusAction({
                          status: 'active',
                          title: 'Activate SMS Template',
                          description: 'Are you sure you want to activate this SMS template? It will be used for SMS delivery.',
                        })
                        setShowStatusDialog(true)
                      }}
                    >
                      <Play className="mr-2 size-4" />
                      Activate Template
                    </DropdownMenuItem>
                  ) : (
                    <DropdownMenuItem
                      onClick={() => {
                        setPendingStatusAction({
                          status: 'inactive',
                          title: 'Deactivate SMS Template',
                          description: 'Are you sure you want to deactivate this SMS template? It will no longer be used for SMS delivery.',
                        })
                        setShowStatusDialog(true)
                      }}
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
        onConfirm={async () => {
          if (!pendingStatusAction) return
          try {
            await updateStatusMutation.mutateAsync({ id: templateId, data: { status: pendingStatusAction.status } })
            setShowStatusDialog(false)
            setPendingStatusAction(null)
          } catch (error) {
            showError(error)
          }
        }}
        title={pendingStatusAction?.title || "Change Status"}
        description={pendingStatusAction?.description || ""}
        confirmText={pendingStatusAction?.status === "active" ? "Activate" : "Deactivate"}
        cancelText="Cancel"
        variant={pendingStatusAction?.status === "inactive" ? "destructive" : "default"}
        isLoading={updateStatusMutation.isPending}
      />
    </>
  )
}
