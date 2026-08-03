import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Shield, Calendar, Plus, Trash2, Eye } from "lucide-react"
import { safeFormat } from "@/lib/formatDate"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton, StatusBadge } from "@/components/details"
import { DataTablePagination, RowActions, usePaginationTable, type RowActionItem } from "@/components/data-table"
import { useClientRoles, useRemoveClientRole } from "@/hooks/useClients"
import { useToast } from "@/hooks/useToast"
import type { Role } from "@/services/api/roles/types"
import { type PaginationState } from "@tanstack/react-table"
import { AssignClientRolesDialog } from "./AssignClientRolesDialog"

interface ClientRolesProps {
  clientId: string
}

export function ClientRoles({ clientId }: ClientRolesProps) {
  const navigate = useNavigate()
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const [assignDialogOpen, setAssignDialogOpen] = useState(false)

  const { showSuccess, showError } = useToast()
  const removeRoleMutation = useRemoveClientRole(clientId)

  const { data, isLoading, isError } = useClientRoles(clientId)

  const clientRoles = data ?? []

  const paginatedRoles = clientRoles.slice(
    pagination.pageIndex * pagination.pageSize,
    (pagination.pageIndex + 1) * pagination.pageSize
  )

  const table = usePaginationTable({
    pagination,
    onPaginationChange: setPagination,
    pageCount: Math.ceil(clientRoles.length / pagination.pageSize) || 1,
  })

  const removeRole = async (role: Role) => {
    try {
      await removeRoleMutation.mutateAsync(role.role_id)
      showSuccess(`Role "${role.name}" removed successfully`)
    } catch (error) {
      showError(error)
    }
  }

  const roleActions = (role: Role): RowActionItem[] => [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () => navigate(`/roles/${role.role_id}`),
    },
    {
      key: "remove",
      label: "Remove from Client",
      icon: Trash2,
      destructive: true,
      separatorBefore: true,
      onSelect: () => removeRole(role),
      confirm: {
        title: "Remove Role from Client",
        description: "This removes the role from this client. The role itself isn't deleted.",
        confirmText: "Remove",
      },
    },
  ]

  const existingRoleIds = clientRoles.map(role => role.role_id)

  return (
    <>
      <InformationCard
        title="Client Roles"
        description="Roles and permissions assigned to this client"
        icon={Shield}
        action={
          <Button size="sm" className="gap-2" onClick={() => setAssignDialogOpen(true)}>
            <Plus className="h-4 w-4" />
            Assign Role
          </Button>
        }
      >
        <div className="space-y-4">
          {isLoading && <ListSkeleton />}

          {isError && (
            <p className="py-8 text-center text-sm text-destructive">Failed to load roles</p>
          )}

          {data && clientRoles.length === 0 && (
            <EmptyState
              icon={Shield}
              title="No roles assigned"
              description="This client has no roles yet. Assign a role to grant permissions."
            />
          )}

          {data && clientRoles.length > 0 && (
            <div className="space-y-3">
              {paginatedRoles.map((role) => (
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
                  data-md-listing-item
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
                            <Badge variant="secondary" className="text-xs">
                              System
                            </Badge>
                          )}
                          {role.is_default && (
                            <Badge variant="outline" className="text-xs">
                              Default
                            </Badge>
                          )}
                        </div>
                        <p className="text-sm text-muted-foreground">{role.description}</p>
                        <div className="flex items-center gap-1 text-xs text-muted-foreground">
                          <Calendar className="h-3 w-3" />
                          Added {safeFormat(role.created_at, "PPP")}
                        </div>
                      </div>
                  </div>
                  <RowActions items={roleActions(role)} />
                </div>
              ))}
            </div>
          )}

          {data && clientRoles.length > 0 && (
            <div className="pt-4 border-t">
              <DataTablePagination table={table} rowCount={clientRoles.length} />
            </div>
          )}
        </div>
      </InformationCard>

      <AssignClientRolesDialog
        open={assignDialogOpen}
        onOpenChange={setAssignDialogOpen}
        clientId={clientId}
        existingRoleIds={existingRoleIds}
      />
    </>
  )
}
