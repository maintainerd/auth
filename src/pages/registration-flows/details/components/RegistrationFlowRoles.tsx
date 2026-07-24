import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Badge } from "@/components/ui/badge"
import { Shield, Plus, Trash2, Eye, Calendar } from "lucide-react"
import { safeFormat } from "@/lib/formatDate"
import { Button } from "@/components/ui/button"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton, StatusBadge } from "@/components/details"
import { DataTablePagination, usePaginationTable, RowActions, type RowActionItem } from "@/components/data-table"
import { useRegistrationFlowRoles, useRemoveRegistrationFlowRole } from "@/hooks/useRegistrationFlows"
import { useToast } from "@/hooks/useToast"
import { AssignRegistrationFlowRolesDialog } from "./AssignRegistrationFlowRolesDialog"
import { type PaginationState } from "@tanstack/react-table"

interface RegistrationFlowRolesProps {
  registrationFlowId: string
}

export function RegistrationFlowRoles({ registrationFlowId }: RegistrationFlowRolesProps) {
  const navigate = useNavigate()
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [isAddOpen, setIsAddOpen] = useState(false)

  const { showSuccess, showError } = useToast()
  const removeRoleMutation = useRemoveRegistrationFlowRole()

  const { data, isLoading, isError } = useRegistrationFlowRoles(registrationFlowId, {
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

  const existingRoleIds = data?.rows.map((r) => r.role_id) ?? []

  const removeRole = async (roleId: string) => {
    try {
      await removeRoleMutation.mutateAsync({ registrationFlowId, roleId })
      showSuccess("Role removed successfully")
    } catch (error) {
      showError(error)
    }
  }

  return (
    <>
      <InformationCard
        title="Roles"
        description="Roles automatically assigned to users who complete this registration flow"
        icon={Shield}
        action={
          <Button onClick={() => setIsAddOpen(true)} size="sm">
            <Plus className="h-4 w-4 mr-2" />
            Assign Role
          </Button>
        }
      >
        <div className="space-y-4">
          {isLoading && <ListSkeleton />}

          {isError && (
            <p className="py-8 text-center text-sm text-destructive">Failed to load roles</p>
          )}

          {data && data.rows.length === 0 && (
            <EmptyState
              icon={Shield}
              title="No roles assigned"
              description="Users who complete this flow still receive the default registered role. Assign roles here to grant more on top of it."
            />
          )}

          {data && data.rows.length > 0 && (
            <div className="space-y-3">
              {data.rows.map((role) => {
                const actions: RowActionItem[] = [
                  {
                    key: "view",
                    label: "View Details",
                    icon: Eye,
                    onSelect: () => navigate(`/roles/${role.role_id}`),
                  },
                  {
                    key: "remove",
                    label: "Remove Role",
                    icon: Trash2,
                    destructive: true,
                    separatorBefore: true,
                    onSelect: () => removeRole(role.role_id),
                    confirm: {
                      title: "Remove Role",
                      description: `This will remove the role "${role.name}" from this registration flow. The role itself is not deleted.`,
                      confirmText: "Remove",
                    },
                  },
                ]

                return (
                  <div
                    key={role.role_id}
                    role="button"
                    tabIndex={0}
                    onClick={(e) => {
                      const target = e.target as HTMLElement
                      if (!e.currentTarget.contains(target) || target.closest("button, a")) return
                      navigate(`/roles/${role.role_id}`)
                    }}
                    onKeyDown={(e) => {
                      if (e.target !== e.currentTarget) return
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault()
                        navigate(`/roles/${role.role_id}`)
                      }
                    }}
                    className="flex cursor-pointer items-start justify-between gap-3 rounded-lg border p-4 transition-colors hover:bg-accent/50"
                  >
                    <div className="flex items-start gap-3">
                      <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                        <Shield className="size-4" />
                      </div>
                      <div className="space-y-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <p className="text-sm font-medium">{role.name}</p>
                          {role.status && <StatusBadge status={role.status} />}
                          {role.is_system && (
                            <Badge variant="secondary" className="text-xs">System</Badge>
                          )}
                          {role.is_default && (
                            <Badge variant="outline" className="text-xs">Default</Badge>
                          )}
                        </div>
                        <p className="text-sm text-muted-foreground">{role.description}</p>
                        <div className="flex items-center gap-1 text-xs text-muted-foreground">
                          <Calendar className="h-3 w-3" />
                          Added {safeFormat(role.created_at, "PPP")}
                        </div>
                      </div>
                    </div>
                    <RowActions items={actions} />
                  </div>
                )
              })}
            </div>
          )}

          {data && data.total > 0 && (
            <div className="border-t pt-4">
              <DataTablePagination table={table} rowCount={data.total} />
            </div>
          )}
        </div>
      </InformationCard>

      <AssignRegistrationFlowRolesDialog
        open={isAddOpen}
        onOpenChange={setIsAddOpen}
        registrationFlowId={registrationFlowId}
        existingRoleIds={existingRoleIds}
      />
    </>
  )
}
