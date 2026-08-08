import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { CheckCircle2, Clock3, RefreshCw, RotateCcw, TriangleAlert } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton } from "@/components/details"
import { useToast } from "@/hooks/useToast"
import { safeFormat } from "@/lib/formatDate"
import { fetchDeliveryHistory, replayDelivery, type DeliveryHistoryItem } from "@/services/api/webhooks"

interface Props {
  webhookId: string
}

function deliveryStatusVariant(
  status: DeliveryHistoryItem["final_status"],
): "default" | "destructive" | "secondary" {
  if (status === "success") return "default"
  if (status === "dead_letter") return "destructive"
  return "secondary"
}

function DeliveryStatusIcon({ status }: { status: DeliveryHistoryItem["final_status"] }) {
  if (status === "success") return <CheckCircle2 className="size-4" />
  if (status === "dead_letter") return <TriangleAlert className="size-4" />
  return <Clock3 className="size-4" />
}

export function WebhookDeliveries({ webhookId }: Props) {
  const queryClient = useQueryClient()
  const { showSuccess, showError } = useToast()
  const [replayingId, setReplayingId] = useState<string | null>(null)

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ["webhook-deliveries", webhookId],
    queryFn: () => fetchDeliveryHistory(webhookId),
    refetchInterval: 15_000,
  })

  const replayMut = useMutation({
    mutationFn: (uuid: string) => replayDelivery(uuid),
    onSuccess: () => {
      showSuccess("Delivery replay initiated")
      queryClient.invalidateQueries({ queryKey: ["webhook-deliveries", webhookId] })
    },
    onError: (e) => showError(e, "Replay failed"),
    onSettled: () => setReplayingId(null),
  })

  const deliveries = Array.isArray(data) ? data : []

  return (
    <InformationCard
      title="Delivery History"
      description="Recent delivery attempts for this webhook endpoint."
      icon={RefreshCw}
      action={
        <Button variant="outline" size="sm" className="h-9 gap-2" onClick={() => refetch()}>
          <RefreshCw className="h-3 w-3 mr-1" /> Refresh
        </Button>
      }
    >
      <div className="space-y-4">
        {isLoading && <ListSkeleton />}

        {isError && (
          <p className="py-8 text-center text-sm text-destructive">
            Failed to load delivery history.
          </p>
        )}

        {!isLoading && !isError && deliveries.length === 0 && (
          <EmptyState
            icon={RefreshCw}
            title="No deliveries"
            description="Delivery attempts will appear here after subscribed events are emitted."
          />
        )}

        {deliveries.length > 0 && (
          <div className="space-y-3">
            {deliveries.map((d: DeliveryHistoryItem) => (
              <div
                key={d.delivery_history_uuid}
                data-md-listing-item
                className="flex flex-col gap-3 rounded-lg border p-4 sm:flex-row sm:items-start sm:justify-between"
              >
                <div className="flex min-w-0 items-start gap-3">
                  <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <DeliveryStatusIcon status={d.final_status} />
                  </div>
                  <div className="min-w-0 space-y-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-sm font-medium">{d.event_type}</span>
                      <Badge
                        variant={deliveryStatusVariant(d.final_status)}
                        className="font-normal capitalize"
                      >
                        {d.final_status === "dead_letter" ? "Dead-lettered" : d.final_status}
                      </Badge>
                      {d.is_replay && (
                        <Badge variant="outline" className="font-normal">
                          Replay
                        </Badge>
                      )}
                    </div>
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                      <span>Response: {d.response_status ?? "N/A"}</span>
                      <span>Attempt {d.attempt_count}</span>
                      <span>{safeFormat(d.created_at, "PPpp")}</span>
                    </div>
                    {d.response_summary && (
                      <p className="break-words text-sm text-muted-foreground">
                        {d.response_summary}
                      </p>
                    )}
                  </div>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-9 w-fit gap-2"
                  disabled={replayingId === d.delivery_history_uuid}
                  onClick={() => {
                    setReplayingId(d.delivery_history_uuid)
                    replayMut.mutate(d.event_id)
                  }}
                >
                  <RotateCcw className="h-3 w-3" />
                  Replay
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>
    </InformationCard>
  )
}
