import { useMemo } from "react"
import { Radio } from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { InformationCard } from "@/components/card/InformationCard"
import { EmptyState } from "@/components/details"
import { useEventTypes } from "@/hooks/useEventTypes"
import { useWebhookSubscriptions, useAddWebhookSubscription, useRemoveWebhookSubscription } from "@/hooks/useWebhooks"
import { useToast } from "@/hooks/useToast"
import { cn } from "@/lib/utils"
import type { EventType } from "@/services/api/event-types/types"

interface WebhookEventsProps {
  webhookId: string
}

function groupByCategory(types: EventType[]): [string, EventType[]][] {
  const groups = new Map<string, EventType[]>()
  for (const t of types) {
    const key = t.category || "OTHER"
    const bucket = groups.get(key)
    if (bucket) bucket.push(t)
    else groups.set(key, [t])
  }
  return [...groups.entries()].sort(([a], [b]) => a.localeCompare(b))
}

export function WebhookEvents({ webhookId }: WebhookEventsProps) {
  const { data: eventTypes, isLoading, isError } = useEventTypes()
  const { data: subscriptions } = useWebhookSubscriptions(webhookId)
  const addSubscription = useAddWebhookSubscription()
  const removeSubscription = useRemoveWebhookSubscription()
  const { showSuccess, showError } = useToast()

  const subscribedUuids = useMemo(() => {
    const set = new Set<string>()
    for (const s of subscriptions ?? []) set.add(s.event_type_uuid)
    return set
  }, [subscriptions])

  const isSubscribed = (uuid: string) => subscribedUuids.has(uuid)

  const handleToggle = (eventType: EventType, enabled: boolean) => {
    const data = { event_type_uuid: eventType.uuid }
    if (enabled) {
      addSubscription.mutate(
        { webhookId, data },
        {
          onSuccess: () => showSuccess(`Subscribed to ${eventType.key}`),
          onError: (error) => showError(error, "Failed to subscribe"),
        },
      )
    } else {
      removeSubscription.mutate(
        { webhookId, data },
        {
          onSuccess: () => showSuccess(`Unsubscribed from ${eventType.key}`),
          onError: (error) => showError(error, "Failed to unsubscribe"),
        },
      )
    }
  }

  const groups = useMemo(() => {
    const all = eventTypes ?? []
    return groupByCategory(all)
  }, [eventTypes])

  return (
    <div className="space-y-4">
      {isLoading && (
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Card key={i}>
              <CardContent className="space-y-3">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-3 w-full max-w-md" />
                <Skeleton className="h-3 w-full max-w-sm" />
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {isError && (
        <p className="py-8 text-center text-sm text-destructive">Failed to load event types</p>
      )}

      {eventTypes && groups.length === 0 && (
        <EmptyState
          icon={Radio}
          title="No event types"
          description="No event types are available for this tenant."
        />
      )}

      {groups.length > 0 && (
        <div className="space-y-4">
          {groups.map(([category, types]) => (
            <InformationCard
              key={category}
              title={category}
              description={`${types.length} event type${types.length === 1 ? "" : "s"}`}
            >
              <div className="divide-y divide-border/60">
                {types.map((t) => {
                  const enabled = isSubscribed(t.uuid)
                  return (
                    <div
                      key={t.uuid}
                      className="flex items-center justify-between gap-4 py-3 first:pt-0 last:pb-0"
                    >
                      <div className={cn("min-w-0 space-y-1", !enabled && "opacity-50")}>
                        <div className="flex items-center gap-2">
                          <code className="rounded bg-muted px-2 py-0.5 font-mono text-sm font-medium">
                            {t.key}
                          </code>
                          {t.version > 1 && (
                            <span className="text-xs text-muted-foreground">v{t.version}</span>
                          )}
                        </div>
                        {t.description && (
                          <p className="text-sm text-muted-foreground">{t.description}</p>
                        )}
                      </div>
                      <Switch
                        checked={enabled}
                        onCheckedChange={(checked) => handleToggle(t, checked)}
                        aria-label={`${enabled ? "Unsubscribe from" : "Subscribe to"} ${t.key}`}
                      />
                    </div>
                  )
                })}
              </div>
            </InformationCard>
          ))}
        </div>
      )}
    </div>
  )
}
