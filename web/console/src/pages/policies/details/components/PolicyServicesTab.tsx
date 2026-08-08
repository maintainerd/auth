import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Server, Eye } from "lucide-react"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton } from "@/components/details"
import { SystemBadge } from "@/components/badges"
import { DataTablePagination, usePaginationTable, RowActions, type RowActionItem } from "@/components/data-table"
import { useServicesByPolicy } from "../hooks/useServicesByPolicy"
import { type PaginationState } from "@tanstack/react-table"
import type { Service } from "@/services/api/services/types"

interface PolicyServicesTabProps {
  policyId: string
}

export function PolicyServicesTab({ policyId }: PolicyServicesTabProps) {
  const navigate = useNavigate()
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data, isLoading, isError } = useServicesByPolicy({
    policyId,
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    sort_by: "name",
    sort_order: "asc",
  })

  const table = usePaginationTable({
    pagination,
    onPaginationChange: setPagination,
    pageCount: data?.total_pages ?? 0,
  })
  const services = data?.rows ?? []

  const serviceActions = (service: Service): RowActionItem[] => [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () =>
        navigate(`/services/${service.service_id}`, {
          state: { from: `/policies/${policyId}`, backLabel: "Back to Policy Details" },
        }),
    },
  ]

  return (
    <InformationCard
      title="Applied Services"
      description="Services where this policy is applied"
      icon={Server}
    >
      <div className="space-y-4">
        {isLoading && <ListSkeleton />}

        {isError && (
          <p className="py-8 text-center text-sm text-destructive">Failed to load services</p>
        )}

        {data && services.length === 0 && (
          <EmptyState
            icon={Server}
            title="No services"
            description="This policy is not applied to any services yet."
          />
        )}

        {services.length > 0 && (
          <div className="space-y-3">
            {services.map((service) => (
              <div
                key={service.service_id}
                role="button"
                tabIndex={0}
                onClick={(e) => {
                  const target = e.target as HTMLElement
                  if (!e.currentTarget.contains(target) || target.closest("button, a")) return
                  navigate(`/services/${service.service_id}`, {
                    state: { from: `/policies/${policyId}`, backLabel: "Back to Policy Details" },
                  })
                }}
                onKeyDown={(e) => {
                  if (e.target !== e.currentTarget) return
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault()
                    navigate(`/services/${service.service_id}`, {
                      state: { from: `/policies/${policyId}`, backLabel: "Back to Policy Details" },
                    })
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
                      <span className="text-sm font-medium">{service.display_name}</span>
                      <SystemBadge isSystem={service.is_system} />
                    </div>
                    <p className="text-sm text-muted-foreground">{service.description}</p>
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                      <span className="font-mono">{service.name}</span>
                    </div>
                  </div>
                </div>
                <RowActions items={serviceActions(service)} />
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
