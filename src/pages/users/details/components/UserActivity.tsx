import { useState } from "react"
import { Activity, Calendar, Globe } from "lucide-react"
import { safeFormat } from "@/lib/formatDate"
import { Badge } from "@/components/ui/badge"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton } from "@/components/details"
import { DataTablePagination, usePaginationTable } from "@/components/data-table"
import { useUserActivity } from "@/hooks/useUsers"
import { cn } from "@/lib/utils"
import type { AuthEvent } from "@/services/api/users/types"
import { type PaginationState } from "@tanstack/react-table"

interface UserActivityProps {
  userId: string
}

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

function ResultBadge({ result }: { result: string }) {
  const key = result.toUpperCase()
  return (
    <span className="inline-flex items-center gap-1.5 rounded-md border bg-background px-2 py-0.5 text-xs font-medium capitalize">
      <span className={cn("size-1.5 rounded-full", RESULT_DOT[key] ?? "bg-muted-foreground")} />
      {result.toLowerCase()}
    </span>
  )
}

export function UserActivity({ userId }: UserActivityProps) {
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })

  const { data, isLoading, isError } = useUserActivity(userId, {
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    sort_by: "created_at",
    sort_order: "desc",
  })

  const table = usePaginationTable({
    pagination,
    onPaginationChange: setPagination,
    pageCount: data?.total_pages ?? 0,
  })

  return (
    <InformationCard
      title="Activity"
      description="Authentication and security events involving this user. Read-only."
      icon={Activity}
    >
      <div className="space-y-4">
        {isLoading && <ListSkeleton />}

        {isError && <p className="py-8 text-center text-sm text-destructive">Failed to load activity</p>}

        {data && data.rows.length === 0 && (
          <EmptyState
            icon={Activity}
            title="No activity"
            description="No authentication or security events have been recorded for this user yet."
          />
        )}

        {data && data.rows.length > 0 && (
          <div className="space-y-3">
            {data.rows.map((event: AuthEvent) => (
              <div data-md-listing-item key={event.auth_event_id} className="rounded-lg border p-4">
                <div className="flex items-start gap-3">
                  <div data-md-listing-icon className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <Activity className="size-4" />
                  </div>
                  <div className="min-w-0 flex-1 space-y-1.5">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="text-sm font-medium">{humanizeEventType(event.event_type)}</p>
                      <ResultBadge result={event.result} />
                      {event.severity && event.severity.toUpperCase() !== "INFO" && (
                        <Badge variant="secondary" className="font-normal capitalize">
                          {event.severity.toLowerCase()}
                        </Badge>
                      )}
                    </div>
                    {(event.description || event.error_reason) && (
                      <p className="text-sm text-muted-foreground">
                        {event.description || event.error_reason}
                      </p>
                    )}
                    <div data-md-listing-meta className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                      <span className="inline-flex items-center gap-1">
                        <Calendar className="size-3" />
                        {safeFormat(event.created_at, "PPp")}
                      </span>
                      {event.ip_address && (
                        <span className="inline-flex items-center gap-1 font-mono">
                          <Globe className="size-3" />
                          {event.ip_address}
                        </span>
                      )}
                      <span className="capitalize">{event.category}</span>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {data && data.total > 0 && (
          <div className="border-t pt-4">
            <DataTablePagination table={table} rowCount={data.total} />
          </div>
        )}
      </div>
    </InformationCard>
  )
}
