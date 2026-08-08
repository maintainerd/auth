import { Activity, ChevronRight, Globe } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { EmptyState } from "@/components/details"
import { useRecentActivity } from "@/hooks/useUsers"
import { cn } from "@/lib/utils"
import type { AuthEvent } from "@/services/api/users/types"

// Mirror the result dot/badge styling used in the per-user activity tab so the
// dashboard feed reads consistently with the full audit view.
const RESULT_DOT: Record<string, string> = {
  SUCCESS: "bg-emerald-500",
  FAILURE: "bg-red-500",
  FAILED: "bg-red-500",
  ERROR: "bg-red-500",
}

// "user.login_success" → "User login success"
const humanizeEventType = (value: string) => {
  const text = value.replace(/[._]/g, " ").trim()
  return text.charAt(0).toUpperCase() + text.slice(1)
}

function ActivityRow({ event }: { event: AuthEvent }) {
  return (
    <div data-md-listing-item className="flex items-start gap-3 rounded-lg border p-3">
      <div data-md-listing-icon className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
        <Activity className="size-4" />
      </div>
      <div className="min-w-0 flex-1 space-y-0.5">
        <div className="flex items-center gap-2">
          <span
            className={cn("size-1.5 shrink-0 rounded-full", RESULT_DOT[event.result?.toUpperCase()] ?? "bg-muted-foreground")}
          />
          <p className="truncate font-medium">{humanizeEventType(event.event_type)}</p>
        </div>
        <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
          <span>{formatDistanceToNow(new Date(event.created_at), { addSuffix: true })}</span>
          {event.ip_address && (
            <span className="inline-flex items-center gap-1 font-mono">
              <Globe className="size-3" />
              {event.ip_address}
            </span>
          )}
        </div>
      </div>
    </div>
  )
}

export function RecentActivityCard({ onViewAll }: { onViewAll: () => void }) {
  // Latest 5 events, newest first — the backend scopes to the caller's session.
  const { data, isLoading, isError } = useRecentActivity({
    page: 1,
    limit: 5,
    sort_by: "created_at",
    sort_order: "desc",
  })

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-2 space-y-0">
        <div className="space-y-1.5">
          <CardTitle className="flex items-center gap-2 text-base">
            <Activity className="h-4 w-4" />
            Recent Activity
          </CardTitle>
          <CardDescription>Latest authentication and security events</CardDescription>
        </div>
        <Button variant="ghost" size="sm" className="shrink-0" onClick={onViewAll}>
          View all
          <ChevronRight className="ml-1 size-4" />
        </Button>
      </CardHeader>
      <CardContent>
        {isLoading && (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-14 w-full rounded-lg" />
            ))}
          </div>
        )}

        {isError && (
          <p className="py-8 text-center text-sm text-destructive">Failed to load recent activity</p>
        )}

        {data && data.rows.length === 0 && (
          <EmptyState
            icon={Activity}
            title="No activity yet"
            description="Authentication and security events will appear here as they happen."
          />
        )}

        {data && data.rows.length > 0 && (
          <div className="space-y-3">
            {data.rows.map((event) => (
              <ActivityRow key={event.auth_event_id} event={event} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
