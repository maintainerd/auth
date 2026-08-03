import { useMemo, useState } from "react"
import { Info, Radio, Search } from "lucide-react"
import { Input } from "@/components/ui/input"
import { Card, CardContent } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { PageContainer, PageHeader } from "@/components/layout"
import { DetailsContainer } from "@/components/container"
import { EmptyState } from "@/components/details"
import { InformationCard } from "@/components/card/InformationCard"
import { useEventTypes, useTenantEventTypes, useSetTenantEventType } from "@/hooks/useEventTypes"
import { useToast } from "@/hooks/useToast"
import { cn } from "@/lib/utils"
import type { EventType } from "@/services/api/event-types/types"

/** Group event types by category, preserving a stable alphabetical order. */
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

export default function EventTypesPage({ standalone = true }: { standalone?: boolean }) {
  const { data, isLoading, isError } = useEventTypes()
  const { data: tenantConfigs } = useTenantEventTypes()
  const setTenantEventType = useSetTenantEventType()
  const { showSuccess, showError } = useToast()
  const [search, setSearch] = useState("")

  // Overrides only contain events with an explicit on/off; absence = enabled.
  const overrides = useMemo(() => {
    const map = new Map<string, boolean>()
    for (const c of tenantConfigs ?? []) map.set(c.event_type_uuid, c.enabled)
    return map
  }, [tenantConfigs])

  const isEnabled = (eventTypeUuid: string) => overrides.get(eventTypeUuid) ?? true

  const handleToggle = (eventType: EventType, enabled: boolean) => {
    setTenantEventType.mutate(
      { event_type_uuid: eventType.uuid, enabled },
      {
        onSuccess: () => showSuccess(`${eventType.key} ${enabled ? "enabled" : "disabled"}`),
        onError: (error) => showError(error, "Failed to update event"),
      },
    )
  }

  const groups = useMemo(() => {
    const all = data ?? []
    const term = search.trim().toLowerCase()
    const filtered = term
      ? all.filter(
          (t) =>
            t.key.toLowerCase().includes(term) ||
            t.category.toLowerCase().includes(term) ||
            t.description.toLowerCase().includes(term),
        )
      : all
    return groupByCategory(filtered)
  }, [data, search])

  const content = (
    <div className="flex flex-col gap-4">
      <Alert>
        <Info className="mt-0.5 size-4 shrink-0" />
        <AlertDescription>
          This is a tenant-wide master switch. Disabling an event stops it at the source — it
          won't reach any webhook or the message broker, regardless of individual subscriptions.
          Events are enabled by default.
        </AlertDescription>
      </Alert>

      <div className="relative max-w-sm">
        <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search event types by key, category, or description..."
          className="pl-9"
        />
      </div>

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

      {data && groups.length === 0 && (
        <EmptyState
          icon={Radio}
          title={search ? "No matching event types" : "No event types"}
          description={
            search
              ? "No event types match your search. Try a different key or category."
              : "No event types are available for this tenant."
          }
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
                  const enabled = isEnabled(t.uuid)
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
                        aria-label={`${enabled ? "Disable" : "Enable"} ${t.key}`}
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

  if (!standalone) return content

  return (
    <DetailsContainer>
      <PageContainer>
        <PageHeader
          title="Event Types"
          description="The catalog of events the system can emit. Toggle an event off to stop it from being delivered to any webhook or message broker for this tenant."
        />
        {content}
      </PageContainer>
    </DetailsContainer>
  )
}
