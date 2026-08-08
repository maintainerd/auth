import { useState } from "react"
import { useNavigate } from "react-router-dom"
import {
  type LucideIcon,
  Edit,
  Trash2,
  MoreVertical,
  Play,
  Pause,
  Webhook as WebhookIcon,
  Radio,
  RefreshCw,
  Timer,
  Activity,
  CalendarDays,
} from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { ConfirmationDialog, DeleteConfirmationDialog } from "@/components/dialog"
import { DetailHeaderCard, StatusBadge, type DetailAttribute } from "@/components/details"
import { useDeleteWebhook, useUpdateWebhookStatus } from "@/hooks/useWebhooks"
import { useToast } from "@/hooks/useToast"
import { safeFormat } from "@/lib/formatDate"
import type { UpdateWebhookStatusRequest, Webhook } from "@/services/api/webhooks/types"
import { webhookDetailState } from "../../webhookNavigation"

interface WebhookHeaderProps {
  webhook: Webhook
  webhookId: string
  afterDeleteTo?: string
}

interface StatusAction {
  status: UpdateWebhookStatusRequest["status"]
  label: string
  title: string
  description: string
  icon: LucideIcon
  destructive?: boolean
}

export function WebhookHeader({ webhook, webhookId, afterDeleteTo }: WebhookHeaderProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const deleteWebhookMutation = useDeleteWebhook()
  const updateStatusMutation = useUpdateWebhookStatus()
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [statusAction, setStatusAction] = useState<StatusAction | null>(null)

  const isActive = webhook.status === "active"
  const statusActions: StatusAction[] = isActive
    ? [
        {
          status: "inactive",
          label: "Deactivate Webhook",
          title: "Deactivate Webhook",
          description:
            "Are you sure you want to deactivate this webhook? It will stop receiving event deliveries until reactivated.",
          icon: Pause,
          destructive: true,
        },
      ]
    : [
        {
          status: "active",
          label: "Activate Webhook",
          title: "Activate Webhook",
          description:
            "Are you sure you want to activate this webhook? It will start receiving event deliveries again.",
          icon: Play,
        },
      ]

  const handleDelete = async () => {
    try {
      await deleteWebhookMutation.mutateAsync(webhookId)
      showSuccess("Webhook deleted successfully")
      navigate(afterDeleteTo ?? `/events?tab=webhooks`)
    } catch (error) {
      showError(error)
    }
  }

  const handleStatusChange = async () => {
    if (!statusAction) return
    try {
      await updateStatusMutation.mutateAsync({ webhookId, data: { status: statusAction.status } })
      showSuccess(`Webhook ${statusAction.status === "active" ? "activated" : "deactivated"}`)
    } catch (error) {
      showError(error)
    } finally {
      setStatusAction(null)
    }
  }

  const attributes: DetailAttribute[] = [
    {
      icon: Radio,
      label: "Events",
      value: webhook.subscribe_all ? "All events" : "Selected events",
    },
    {
      icon: RefreshCw,
      label: "Max retries",
      value: String(webhook.max_retries),
    },
    {
      icon: Timer,
      label: "Timeout",
      value: `${webhook.timeout_seconds}s`,
    },
    {
      icon: Activity,
      label: "Last triggered",
      value: webhook.last_triggered_at
        ? formatDistanceToNow(new Date(webhook.last_triggered_at), { addSuffix: true })
        : "Never",
    },
    {
      icon: CalendarDays,
      label: "Created",
      value: safeFormat(webhook.created_at, "PP"),
    },
    {
      icon: CalendarDays,
      label: "Last updated",
      value: safeFormat(webhook.updated_at, "PP"),
    },
  ]

  return (
    <>
      <DetailHeaderCard
        leading={
          <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
            <WebhookIcon className="size-6" />
          </div>
        }
        title={<span className="font-mono text-base break-all">{webhook.url}</span>}
        badge={<StatusBadge status={webhook.status} />}
        subtitle={webhook.description ? <span>{webhook.description}</span> : undefined}
        attributes={attributes}
        actions={
          <>
            <Button
              data-md-details-edit-button
              variant="outline"
              size="sm"
              className="h-9 gap-2"
              onClick={() =>
                navigate(`/webhooks/${webhookId}/edit`, {
                  state: webhookDetailState(webhookId),
                })
              }
            >
              <Edit className="size-4" />
              Edit
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button data-md-details-menu-button variant="outline" size="sm" className="h-9 w-9 p-0">
                  <span className="sr-only">Open actions</span>
                  <MoreVertical className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {statusActions.map((action) => (
                  <DropdownMenuItem
                    key={action.status}
                    onClick={() => setStatusAction(action)}
                    className={action.destructive ? "text-destructive focus:text-destructive" : undefined}
                  >
                    <action.icon className="mr-2 size-4" />
                    {action.label}
                  </DropdownMenuItem>
                ))}
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => setShowDeleteDialog(true)}
                  className="text-destructive focus:text-destructive"
                >
                  <Trash2 className="mr-2 size-4" />
                  Delete Webhook
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </>
        }
      />

      <ConfirmationDialog
        open={!!statusAction}
        onOpenChange={(open) => { if (!open) setStatusAction(null) }}
        onConfirm={handleStatusChange}
        title={statusAction?.title ?? ""}
        description={statusAction?.description ?? ""}
        confirmText={statusAction?.label}
        variant={statusAction?.destructive ? "destructive" : "default"}
        isLoading={updateStatusMutation.isPending}
      />

      <DeleteConfirmationDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        onConfirm={handleDelete}
        title="Delete Webhook"
        description="This action cannot be undone. This will permanently delete the webhook endpoint."
        itemName={webhook.url}
        isDeleting={deleteWebhookMutation.isPending}
      />
    </>
  )
}
