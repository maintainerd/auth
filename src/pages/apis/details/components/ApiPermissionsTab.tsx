import { useState } from "react"
import { Key, Plus, Edit, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { InformationCard } from "@/components/card"
import { SystemBadge } from "@/components/badges"
import { EmptyState, ListSkeleton, StatusBadge } from "@/components/details"
import { DataTablePagination, usePaginationTable, RowActions, type RowActionItem } from "@/components/data-table"
import { PermissionFormDialog } from "./PermissionFormDialog"
import { useApiPermissions } from "../hooks/useApiPermissions"
import { useDeletePermission } from "@/hooks/usePermissions"
import { useToast } from "@/hooks/useToast"
import { type PaginationState } from "@tanstack/react-table"
import type { PermissionEntity } from "@/services/api/permissions/types"

interface ApiPermissionsTabProps {
  apiId: string
}

export function ApiPermissionsTab({ apiId }: ApiPermissionsTabProps) {
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [editingPermission, setEditingPermission] = useState<PermissionEntity | null>(null)

  const { data, isLoading, isError } = useApiPermissions({
    apiId,
    limit: pagination.pageSize,
    page: pagination.pageIndex + 1,
  })

  const deletePermissionMutation = useDeletePermission()
  const { showSuccess, showError } = useToast()

  const table = usePaginationTable({
    pagination,
    onPaginationChange: setPagination,
    pageCount: data?.total_pages ?? 0,
  })

  const removePermission = async (permissionId: string) => {
    try {
      await deletePermissionMutation.mutateAsync(permissionId)
      showSuccess("Permission deleted successfully")
    } catch (error) {
      showError(error)
    }
  }

  return (
    <>
      <InformationCard
        title="Permissions"
        description="Permissions assigned to this API"
        icon={Key}
        action={
          <Button size="sm" className="gap-2" onClick={() => { setEditingPermission(null); setIsDialogOpen(true) }}>
            <Plus className="h-4 w-4" />
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
              description="This API has no permissions yet. Add some to control what clients and roles can access."
            />
          )}

          {data && data.rows.length > 0 && (
            <div className="space-y-3">
              {data.rows.map((permission) => {
                const actions: RowActionItem[] = []

                if (!permission.is_system) {
                  actions.push({
                    key: "edit",
                    label: "Edit Permission",
                    icon: Edit,
                    onSelect: () => {
                      setEditingPermission(permission)
                      setIsDialogOpen(true)
                    },
                  })
                }

                if (!permission.is_system) {
                  actions.push({
                    key: "remove",
                    label: "Delete Permission",
                    icon: Trash2,
                    destructive: true,
                    separatorBefore: actions.length > 0,
                    onSelect: () => removePermission(permission.permission_id),
                    // Permanent deletion (not a detach) → type-to-confirm.
                    confirm: {
                      title: "Delete Permission",
                      description: `This will permanently delete the permission "${permission.name}" and remove it from all roles and policies.`,
                      destructive: true,
                      itemName: permission.name,
                    },
                  })
                }

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
                          <StatusBadge status={permission.status} />
                          <SystemBadge isSystem={permission.is_system} />
                        </div>
                        <p className="text-sm text-muted-foreground">{permission.description}</p>
                      </div>
                    </div>
                    {actions.length > 0 && <RowActions items={actions} />}
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

      <PermissionFormDialog
        open={isDialogOpen}
        onOpenChange={(open) => {
          setIsDialogOpen(open)
          if (!open) setEditingPermission(null)
        }}
        apiId={apiId}
        permission={editingPermission || undefined}
      />
    </>
  )
}
