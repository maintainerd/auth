import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Edit, Trash2, MoreVertical, Shield, CalendarDays, Play, Pause } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { ConfirmationDialog, DeleteConfirmationDialog } from "@/components/dialog"
import { DetailHeaderCard, StatusBadge, type DetailAttribute } from "@/components/details"
import { useDeleteRole, useUpdateRoleStatus } from "@/hooks/useRoles"
import { useToast } from "@/hooks/useToast"
import { safeFormat } from "@/lib/formatDate"
import type { Role, RoleStatus } from "@/services/api/roles/types"

interface RoleHeaderProps {
  role: Role
  roleId: string
}

export function RoleHeader({ role, roleId }: RoleHeaderProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const deleteRoleMutation = useDeleteRole()
  const updateStatusMutation = useUpdateRoleStatus()
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [statusAction, setStatusAction] = useState<{ status: RoleStatus; title: string; description: string } | null>(null)

  const handleDelete = async () => {
    try {
      await deleteRoleMutation.mutateAsync(roleId)
      showSuccess("Role deleted successfully")
      navigate(`/roles`)
    } catch (error) {
      showError(error)
    }
  }

  const handleStatusChange = async () => {
    if (!statusAction) return
    try {
      await updateStatusMutation.mutateAsync({ roleId, data: { status: statusAction.status } })
      showSuccess(`Role status updated to ${statusAction.status}`)
    } catch (error) {
      showError(error)
    } finally {
      setStatusAction(null)
    }
  }

  // Availability mirrors the backend rules: system roles can't be edited, change
  // status, or be deleted; the default (registration) role can't be deactivated
  // or deleted. System roles hide the whole actions block — matching ApiHeader,
  // PolicyHeader, and ServiceHeader — rather than offering an Edit button that
  // only ever opens a permanently disabled form.
  const isActive = role.status === "active"
  const canActivate = !isActive
  const canDeactivate = isActive && !role.is_default
  const canDelete = !role.is_default
  const hasMenu = canActivate || canDeactivate || canDelete

  const attributes: DetailAttribute[] = [
    {
      icon: Shield,
      label: "Type",
      value: role.is_system ? "System Role" : "Custom Role",
    },
    ...(role.is_default
      ? [{ icon: Shield, label: "Default", value: "Default role assigned to new users" } satisfies DetailAttribute]
      : []),
    {
      icon: CalendarDays,
      label: "Created",
      value: safeFormat(role.created_at, "PP"),
    },
    {
      icon: CalendarDays,
      label: "Last updated",
      value: safeFormat(role.updated_at, "PP"),
    },
  ]

  return (
    <>
      <DetailHeaderCard
        leading={
          <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
            <Shield className="size-6" />
          </div>
        }
        title={role.name}
        badge={<StatusBadge status={role.status} />}
        subtitle={role.description}
        attributes={attributes}
        actions={
          !role.is_system && (
            <>
              <Button
                variant="outline"
                size="sm"
                className="h-9 gap-2"
                onClick={() =>
                  navigate(`/roles/${roleId}/edit`, {
                    state: { from: `/roles/${roleId}`, backLabel: "Back to Role Details" },
                  })
                }
              >
                <Edit className="size-4" />
                Edit
              </Button>
              {hasMenu && (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button data-md-details-menu-button variant="outline" size="sm" className="h-9 w-9 p-0">
                      <span className="sr-only">Open actions</span>
                      <MoreVertical className="size-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    {canActivate && (
                      <DropdownMenuItem
                        onClick={() =>
                          setStatusAction({
                            status: "active",
                            title: "Activate Role",
                            description: "Are you sure you want to activate this role? It can be assigned to users again.",
                          })
                        }
                      >
                        <Play className="mr-2 size-4" />
                        Activate Role
                      </DropdownMenuItem>
                    )}
                    {canDeactivate && (
                      <DropdownMenuItem
                        onClick={() =>
                          setStatusAction({
                            status: "inactive",
                            title: "Deactivate Role",
                            description: "Are you sure you want to deactivate this role? It can no longer be assigned to users.",
                          })
                        }
                        className="text-destructive focus:text-destructive"
                      >
                        <Pause className="mr-2 size-4" />
                        Deactivate Role
                      </DropdownMenuItem>
                    )}
                    {canDelete && (
                      <>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={() => setShowDeleteDialog(true)}
                          className="text-destructive focus:text-destructive"
                        >
                          <Trash2 className="mr-2 size-4" />
                          Delete Role
                        </DropdownMenuItem>
                      </>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
              )}
            </>
          )
        }
      />

      <ConfirmationDialog
        open={!!statusAction}
        onOpenChange={(open) => { if (!open) setStatusAction(null) }}
        onConfirm={handleStatusChange}
        title={statusAction?.title ?? ""}
        description={statusAction?.description ?? ""}
        variant={statusAction?.status === "inactive" ? "destructive" : "default"}
        confirmText={statusAction?.status === "active" ? "Activate" : "Deactivate"}
        isLoading={updateStatusMutation.isPending}
      />

      <DeleteConfirmationDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        onConfirm={handleDelete}
        title="Delete Role"
        description="This action cannot be undone. This will permanently delete the role and all associated data."
        itemName={role.name}
        isDeleting={deleteRoleMutation.isPending}
      />
    </>
  )
}
