import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Server, Eye } from "lucide-react"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton } from "@/components/details"
import { DataTablePagination, usePaginationTable, RowActions, type RowActionItem } from "@/components/data-table"
import { useServiceApis } from "../hooks/useServiceApis"
import { type PaginationState } from "@tanstack/react-table"
import type { Api } from "@/services/api/api/types"

interface ServiceApisTabProps {
  serviceId: string
}

const navState = (serviceId: string) => ({
  from: `/services/${serviceId}`,
  backLabel: "Back to Service Details",
})

export function ServiceApisTab({ serviceId }: ServiceApisTabProps) {
  const navigate = useNavigate()
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data, isLoading, isError } = useServiceApis({
    serviceId,
    limit: pagination.pageSize,
    page: pagination.pageIndex + 1,
    sortBy: "name",
    sortOrder: "asc",
  })

  const table = usePaginationTable({
    pagination,
    onPaginationChange: setPagination,
    pageCount: data?.total_pages ?? 0,
  })

  const apiActions = (api: Api): RowActionItem[] => [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () => navigate(`/apis/${api.api_id}`, { state: navState(serviceId) }),
    },
  ]

  return (
    <InformationCard
      title="Service APIs"
      description="APIs registered under this service"
      icon={Server}
    >
      <div className="space-y-4">
        {isLoading && <ListSkeleton />}

        {isError && (
          <p className="py-8 text-center text-sm text-destructive">Failed to load APIs</p>
        )}

        {data && data.rows.length === 0 && (
          <EmptyState
            icon={Server}
            title="No APIs"
            description="This service has no APIs registered yet."
          />
        )}

        {data && data.rows.length > 0 && (
          <div className="space-y-3">
            {data.rows.map((api) => (
              <div
                key={api.api_id}
                role="button"
                tabIndex={0}
                onClick={(e) => {
                  const target = e.target as HTMLElement
                  if (!e.currentTarget.contains(target) || target.closest("button, a")) return
                  navigate(`/apis/${api.api_id}`, { state: navState(serviceId) })
                }}
                onKeyDown={(e) => {
                  if (e.target !== e.currentTarget) return
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault()
                  navigate(`/apis/${api.api_id}`, { state: navState(serviceId) })
                  }
                }}
                data-md-listing-item
                className="flex cursor-pointer items-start justify-between gap-3 rounded-lg border p-4 transition-colors hover:bg-accent/50"
              >
                <div className="flex min-w-0 items-start gap-3">
                  <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <Server className="size-4" />
                  </div>
                  <div className="min-w-0 space-y-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-sm font-medium">{api.display_name}</span>
                    </div>
                    <p className="text-sm text-muted-foreground">{api.description}</p>
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                      <span className="font-mono">{api.name}</span>
                      {api.identifier && <span className="font-mono">{api.identifier}</span>}
                    </div>
                  </div>
                </div>
                <RowActions items={apiActions(api)} />
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
