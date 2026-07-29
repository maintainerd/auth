import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { FileText, Plus, Eye, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton } from "@/components/details"
import { SystemBadge } from "@/components/badges"
import { DataTablePagination, usePaginationTable, RowActions, type RowActionItem } from "@/components/data-table"
import { PolicyAssignDialog } from "./PolicyAssignDialog"
import { useServicePolicies } from "../hooks/useServicePolicies"
import { useServicePolicyMutations } from "../hooks/useServicePolicyMutations"
import { useToast } from "@/hooks/useToast"
import { type PaginationState } from "@tanstack/react-table"
import type { Policy } from "@/services/api/policies/types"

interface ServicePoliciesTabProps {
  serviceId: string
}

export function ServicePoliciesTab({ serviceId }: ServicePoliciesTabProps) {
  const navigate = useNavigate()
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const [isDialogOpen, setIsDialogOpen] = useState(false)

  const { data, isLoading, isError } = useServicePolicies({
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

  const { removePolicy: removePolicyMutation } = useServicePolicyMutations(serviceId)
  const { showSuccess, showError } = useToast()

  const removePolicy = async (policy: Policy) => {
    try {
      await removePolicyMutation.mutateAsync(policy.policy_id)
      showSuccess(`Policy "${policy.name}" removed successfully`)
    } catch (error) {
      showError(error)
    }
  }

  const navState = { from: `/services/${serviceId}`, backLabel: "Back to Service Details" }
  const existingPolicyIds = data?.rows.map((p) => p.policy_id) ?? []

  const policyActions = (policy: Policy): RowActionItem[] => {
    const items: RowActionItem[] = [
      {
        key: "view",
        label: "View Details",
        icon: Eye,
        onSelect: () => navigate(`/policies/${policy.policy_id}`, { state: navState }),
      },
    ]

    items.push({
      key: "remove",
      label: "Remove from Service",
      icon: Trash2,
      destructive: true,
      separatorBefore: true,
      onSelect: () => removePolicy(policy),
      confirm: {
        title: "Remove Policy from Service",
        description: "This removes the policy from this service. The policy itself will not be deleted.",
        confirmText: "Remove",
      },
    })

    return items
  }

  return (
    <>
      <InformationCard
        title="Applied Policies"
        description="Policies applied to this service for access control and permissions management"
        icon={FileText}
        action={
          <Button size="sm" className="gap-2" onClick={() => setIsDialogOpen(true)}>
            <Plus className="h-4 w-4" />
            Add Policy
          </Button>
        }
      >
        <div className="space-y-4">
          {isLoading && <ListSkeleton />}

          {isError && (
            <p className="py-8 text-center text-sm text-destructive">Failed to load policies</p>
          )}

          {data && data.rows.length === 0 && (
            <EmptyState
              icon={FileText}
              title="No policies"
              description="This service has no policies assigned yet. Add a policy to control access and permissions."
            />
          )}

          {data && data.rows.length > 0 && (
            <div className="space-y-3">
              {data.rows.map((policy) => (
                <div
                  key={policy.policy_id}
                  role="button"
                  tabIndex={0}
                  onClick={(e) => {
                    const target = e.target as HTMLElement
                    if (!e.currentTarget.contains(target) || target.closest("button, a")) return
                    navigate(`/policies/${policy.policy_id}`, { state: navState })
                  }}
                  onKeyDown={(e) => {
                    if (e.target !== e.currentTarget) return
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault()
                      navigate(`/policies/${policy.policy_id}`, { state: navState })
                    }
                  }}
                  className="flex cursor-pointer items-start justify-between gap-3 rounded-lg border p-4 transition-colors hover:bg-accent/50"
                >
                  <div className="flex min-w-0 items-start gap-3">
                    <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                      <FileText className="size-4" />
                    </div>
                    <div className="min-w-0 space-y-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-sm font-medium">{policy.name}</span>
                        <SystemBadge isSystem={policy.is_system} />
                      </div>
                      <p className="text-sm text-muted-foreground">{policy.description}</p>
                    </div>
                  </div>
                  <RowActions items={policyActions(policy)} />
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

      <PolicyAssignDialog
        open={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        serviceId={serviceId}
        existingPolicyIds={existingPolicyIds}
      />
    </>
  )
}
