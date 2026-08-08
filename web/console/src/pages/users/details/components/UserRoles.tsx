import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Shield, Calendar, Plus, Trash2, Eye } from "lucide-react"
import { safeFormat } from "@/lib/formatDate"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton, StatusBadge } from "@/components/details"
import { DataTablePagination, RowActions, usePaginationTable, type RowActionItem } from "@/components/data-table"
import { useUserRoles, useRemoveUserRole } from "@/hooks/useUsers"
import { useToast } from "@/hooks/useToast"
import type { UserRole } from "@/services/api/users/types"
import { type PaginationState } from "@tanstack/react-table"
import { AssignUserRolesDialog } from "./AssignUserRolesDialog"

interface UserRolesProps {
  userId: string
}

export function UserRoles({ userId }: UserRolesProps) {
  const navigate = useNavigate()
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const [assignDialogOpen, setAssignDialogOpen] = useState(false)

  const { showSuccess, showError } = useToast()
  const removeRoleMutation = useRemoveUserRole()

  const { data, isLoading, isError } = useUserRoles(userId, {
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    sort_by: 'created_at',
    sort_order: 'desc',
  })

  const table = usePaginationTable({
    pagination,
    onPaginationChange: setPagination,
    pageCount: data?.total_pages ?? 0,
  })

  const removeRole = async (role: UserRole) => {
    try {
      await removeRoleMutation.mutateAsync({ userId, roleId: role.role_id })
      showSuccess(`Role "${role.name}" removed successfully`)
    } catch (error) {
      showError(error)
    }
  }

  const roleActions = (role: UserRole): RowActionItem[] => [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () => navigate(`/roles/${role.role_id}`),
    },
    {
      key: "remove",
      label: "Remove from User",
      icon: Trash2,
      destructive: true,
      separatorBefore: true,
      onSelect: () => removeRole(role),
      confirm: {
        title: "Remove Role from User",
        description: "This removes the role from this user. The role itself isn't deleted.",
        confirmText: "Remove",
      },
    },
  ]

  // Get existing role IDs for the assign dialog
  const existingRoleIds = data?.rows.map(role => role.role_id) || []

  return (
    <>
      <InformationCard
        title="User Roles"
        description="Roles and permissions assigned to this user"
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

        {data && data.rows.length === 0 && (
          <EmptyState
            icon={Shield}
            title="No roles assigned"
            description="This user has no roles yet. Assign a role to grant permissions."
          />
        )}

        {data && data.rows.length > 0 && (
          <div className="space-y-3">
            {data.rows.map((role: UserRole) => (
              <div
                data-md-listing-item
                key={role.role_id}
                role="button"
                tabIndex={0}
                onClick={(e) => {
                  // Ignore clicks coming from the actions menu (its button and
                  // portaled items), so only the row body navigates.
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
                  <div data-md-listing-icon className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
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
                    <div data-md-listing-meta className="flex items-center gap-1 text-xs text-muted-foreground">
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

        {/* Pagination controls */}
        {data && data.total > 0 && (
          <div className="pt-4 border-t">
            <DataTablePagination table={table} rowCount={data.total} />
          </div>
        )}
      </div>
    </InformationCard>

      {/* Assign Roles Dialog */}
      <AssignUserRolesDialog
        open={assignDialogOpen}
        onOpenChange={setAssignDialogOpen}
        userId={userId}
        existingRoleIds={existingRoleIds}
      />
    </>
  )
}
