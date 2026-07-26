import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { RefreshCw, RotateCcw } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { useToast } from "@/hooks/useToast"
import { fetchDeliveryHistory, replayDelivery, type DeliveryHistoryItem } from "@/services/api/webhooks"

interface Props {
  webhookId: string
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

  if (isLoading) return <Skeleton className="h-48 w-full" />
  if (isError) return <p className="text-muted-foreground p-4">Failed to load delivery history.</p>

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">{deliveries.length} deliveries</p>
        <Button variant="ghost" size="sm" onClick={() => refetch()}>
          <RefreshCw className="h-3 w-3 mr-1" /> Refresh
        </Button>
      </div>

      {deliveries.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            No delivery history yet.
          </CardContent>
        </Card>
      ) : (
        deliveries.map((d: DeliveryHistoryItem) => (
          <Card key={d.delivery_history_uuid}>
            <CardContent className="flex items-center justify-between py-3">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-medium text-sm">{d.event_type}</span>
                  <span className={`text-xs px-1.5 py-0.5 rounded ${d.final_status === "success" ? "bg-green-100 text-green-700" : d.final_status === "dead_letter" ? "bg-red-100 text-red-700" : "bg-yellow-100 text-yellow-700"}`}>
                    {d.final_status === "dead_letter" ? "dead-lettered" : d.final_status}
                  </span>
                  {d.is_replay && <span className="text-xs text-muted-foreground">(replay)</span>}
                </div>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Response: {d.response_status ?? "N/A"} · Attempt {d.attempt_count} · {new Date(d.created_at).toLocaleString()}
                </p>
              </div>
              <Button
                variant="ghost"
                size="sm"
                disabled={replayingId === d.delivery_history_uuid}
                onClick={() => { setReplayingId(d.delivery_history_uuid); replayMut.mutate(d.delivery_history_uuid) }}
              >
                <RotateCcw className="h-3 w-3 mr-1" />
                Replay
              </Button>
            </CardContent>
          </Card>
        ))
      )}
    </div>
  )
}
