import { useState } from "react"
import { Key, Plus, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { safeFormat } from "@/lib/formatDate"
import { InformationCard } from "@/components/card"
import { SystemBadge } from "@/components/badges"
import { EmptyState, ListSkeleton } from "@/components/details"
import { DataTablePagination, usePaginationTable, RowActions, type RowActionItem } from "@/components/data-table"
import { AddRolePermissionsDialog } from "./AddRolePermissionsDialog"
import { useRolePermissions, useRemoveRolePermission } from "@/hooks/useRoles"
import { useToast } from "@/hooks/useToast"
import { type PaginationState } from "@tanstack/react-table"

interface RolePermissionsTabProps {
  roleId: string
}

export function RolePermissionsTab({ roleId }: RolePermissionsTabProps) {
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [isAddOpen, setIsAddOpen] = useState(false)

  const { data, isLoading, isError } = useRolePermissions(roleId, {
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    status: "active",
  })
  const removePermissionMutation = useRemoveRolePermission()
  const { showSuccess, showError } = useToast()

  const table = usePaginationTable({
    pagination,
    onPaginationChange: setPagination,
    pageCount: data?.total_pages ?? 0,
  })

  const existingPermissionIds = data?.rows?.map((p) => p.permission_id) ?? []

  const removePermission = async (permissionId: string) => {
    try {
      await removePermissionMutation.mutateAsync({ roleId, permissionId })
      showSuccess("Permission removed from role successfully")
    } catch (error) {
      showError(error)
    }
  }

  return (
    <>
      <InformationCard
        title="Permissions"
        description="Permissions assigned to this role"
        icon={Key}
        action={
          <Button onClick={() => setIsAddOpen(true)} size="sm">
            <Plus className="h-4 w-4 mr-2" />
            Add Permission
          </Button>
        }
      >
        <div className="space-y-4">
          {isLoading && <ListSkeleton />}

          {isError && (
            <p className="py-8 text-center text-sm text-destructive">Failed to load permissions</p>
          )}

          {data && data.rows.length === 0 && (
            <EmptyState
              icon={Key}
              title="No permissions"
              description="This role has no permissions assigned yet. Add some to control what users with this role can access."
            />
          )}

          {data && data.rows.length > 0 && (
            <div className="space-y-3">
              {data.rows.map((permission) => {
                const actions: RowActionItem[] = [
                  {
                    key: "remove",
                    label: "Remove Permission",
                    icon: Trash2,
                    destructive: true,
                    onSelect: () => removePermission(permission.permission_id),
                    confirm: {
                      title: "Remove Permission",
                      description: `This will permanently remove the permission "${permission.name}" from this role.`,
                      confirmText: "Remove",
                    },
                  },
                ]

                return (
                  <div
                    key={permission.permission_id}
                    data-md-listing-item
                    className="flex items-start justify-between gap-3 rounded-lg border p-4"
                  >
                    <div className="flex min-w-0 items-start gap-3">
                      <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                        <Key className="size-4" />
                      </div>
                      <div className="min-w-0 space-y-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-mono text-sm font-medium">{permission.name}</span>
                          <SystemBadge isSystem={permission.is_system} />
                        </div>
                        <p className="text-sm text-muted-foreground">{permission.description}</p>
                        <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                          {permission.api && (
                            <span>API: {permission.api.display_name}</span>
                          )}
                          <span>Created {safeFormat(permission.created_at, "MMM dd, yyyy")}</span>
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

      <AddRolePermissionsDialog
        open={isAddOpen}
        onOpenChange={setIsAddOpen}
        roleId={roleId}
        existingPermissionIds={existingPermissionIds}
      />
    </>
  )
}
