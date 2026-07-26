import { useNavigate } from "react-router-dom"
import { Eye, Edit, Trash2, Play, Pause } from "lucide-react"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useUpdateWebhookStatus, useDeleteWebhook } from "@/hooks/useWebhooks"
import { useToast } from "@/hooks/useToast"
import type { Webhook } from "@/services/api/webhooks/types"
import { WEBHOOKS_BACK_STATE } from "../webhookNavigation"

interface WebhookActionsProps {
  webhook: Webhook
}

export function WebhookActions({ webhook }: WebhookActionsProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const updateStatusMutation = useUpdateWebhookStatus()
  const deleteWebhookMutation = useDeleteWebhook()

  const changeStatus = async (status: "active" | "inactive") => {
    try {
      await updateStatusMutation.mutateAsync({
        webhookId: webhook.webhook_endpoint_id,
        data: { status },
      })
      showSuccess(`Webhook ${status === "active" ? "activated" : "deactivated"}`)
    } catch (error) {
      showError(error)
    }
  }

  const isActive = webhook.status === "active"

  const items: RowActionItem[] = [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () =>
        navigate(`/webhooks/${webhook.webhook_endpoint_id}`, { state: WEBHOOKS_BACK_STATE }),
    },
    {
      key: "edit",
      label: "Edit Webhook",
      icon: Edit,
      onSelect: () =>
        navigate(`/webhooks/${webhook.webhook_endpoint_id}/edit`, { state: WEBHOOKS_BACK_STATE }),
    },
    ...(isActive
      ? [
          {
            key: "deactivate",
            label: "Deactivate Webhook",
            icon: Pause,
            destructive: true,
            onSelect: () => changeStatus("inactive"),
            confirm: {
              title: "Deactivate Webhook",
              description:
                "Are you sure you want to deactivate this webhook? It will stop receiving event deliveries until reactivated.",
              confirmText: "Deactivate",
            },
          } satisfies RowActionItem,
        ]
      : [
          {
            key: "activate",
            label: "Activate Webhook",
            icon: Play,
            onSelect: () => changeStatus("active"),
            confirm: {
              title: "Activate Webhook",
              description: "Are you sure you want to activate this webhook? It will start receiving event deliveries.",
              confirmText: "Activate",
            },
          } satisfies RowActionItem,
        ]),
    {
      key: "delete",
      label: "Delete Webhook",
      icon: Trash2,
      destructive: true,
      separatorBefore: true,
      onSelect: async () => {
        try {
          await deleteWebhookMutation.mutateAsync(webhook.webhook_endpoint_id)
          showSuccess("Webhook deleted successfully")
        } catch (error) {
          showError(error)
        }
      },
      confirm: {
        title: "Delete Webhook",
        description: "This action cannot be undone. This will permanently delete the webhook endpoint.",
        destructive: true,
        itemName: webhook.url,
      },
    },
  ]

  return <RowActions items={items} />
}
