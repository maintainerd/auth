import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Building2, CalendarDays, Edit, Fingerprint, Hash, MoreVertical, User, Trash2, Play, Pause, Ban } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useDeleteTenant, useUpdateTenantStatus } from "@/hooks/useTenants"
import { useToast } from "@/hooks/useToast"
import { ConfirmationDialog, DeleteConfirmationDialog } from "@/components/dialog"
import { DetailHeaderCard, StatusBadge, type DetailAttribute } from "@/components/details"
import { safeFormat } from "@/lib/formatDate"
import { SystemBadge } from "@/components/badges"
import type { TenantEntity, TenantStatus } from "@/services/api/tenants/types"

interface TenantHeaderProps {
  tenant: TenantEntity
  tenantId: string
}

interface StatusAction {
  status: TenantStatus
  label: string
  title: string
  description: string
  icon: typeof Play
}

const STATUS_ACTIONS: Record<TenantStatus, StatusAction[]> = {
  inactive: [
    { status: "active", label: "Activate Tenant", title: "Activate Tenant", description: "Are you sure you want to activate this tenant? Users will be able to sign in.", icon: Play },
  ],
  active: [
    { status: "inactive", label: "Deactivate Tenant", title: "Deactivate Tenant", description: "Are you sure you want to deactivate this tenant? Users will no longer be able to sign in.", icon: Pause },
    { status: "suspended", label: "Suspend Tenant", title: "Suspend Tenant", description: "Are you sure you want to suspend this tenant? All active sessions will be terminated.", icon: Ban },
  ],
  suspended: [
    { status: "active", label: "Activate Tenant", title: "Activate Tenant", description: "Are you sure you want to reactivate this suspended tenant?", icon: Play },
  ],
}

export function TenantHeader({ tenant, tenantId }: TenantHeaderProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const deleteTenantMutation = useDeleteTenant()
  const updateStatusMutation = useUpdateTenantStatus()
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [statusAction, setStatusAction] = useState<StatusAction | null>(null)

  const handleDelete = async () => {
    try {
      await deleteTenantMutation.mutateAsync(tenantId)
      showSuccess("Tenant deleted successfully")
      navigate(`/tenants`)
    } catch (error) {
      showError(error)
    }
  }

  const handleStatusChange = async () => {
    if (!statusAction) return
    try {
      await updateStatusMutation.mutateAsync({ tenantId, status: statusAction.status })
      showSuccess(`Tenant status updated to ${statusAction.status}`)
    } catch (error) {
      showError(error)
    } finally {
      setStatusAction(null)
    }
  }

  const statusActions = tenant.is_system
    ? []
    : STATUS_ACTIONS[tenant.status]

  const attributes: DetailAttribute[] = [
    { icon: Fingerprint, label: "Tenant ID", value: <span className="font-mono text-xs">{tenant.tenant_id}</span> },
    { icon: Hash, label: "Name", value: <span className="font-mono text-xs">{tenant.name}</span> },
    { icon: User, label: "Description", value: tenant.description || <span className="text-muted-foreground">—</span> },
    { icon: CalendarDays, label: "Created", value: safeFormat(tenant.created_at, "PP") },
    { icon: CalendarDays, label: "Last updated", value: safeFormat(tenant.updated_at, "PP") },
  ]

  return (
    <>
      <DetailHeaderCard
        leading={
          <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
            <Building2 className="size-6" />
          </div>
        }
        title={tenant.display_name}
        badge={
          <div className="flex items-center gap-2">
            <StatusBadge status={tenant.status} />
            {tenant.is_system && <SystemBadge isSystem />}
            {tenant.is_default && <Badge variant="outline" className="text-xs">Default</Badge>}
          </div>
        }
        subtitle={<span className="font-mono text-xs text-muted-foreground">{tenant.name}</span>}
        attributes={attributes}
        actions={
          !tenant.is_system && (
            <>
              <Button
                data-md-details-edit-button
                variant="outline"
                size="sm"
                className="h-9 gap-2"
                onClick={() =>
                  navigate(`/tenants/${tenantId}/edit`, {
                    state: { from: `/tenants/${tenantId}`, backLabel: "Back to Tenant Details" },
                  })
                }
              >
                <Edit className="size-4" />
                Edit
              </Button>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button data-md-details-menu-button variant="outline" size="sm" className="h-9 w-9 p-0">
                    <span className="sr-only">Open actions</span>
                    <MoreVertical className="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  {statusActions.map((action) => (
                    <DropdownMenuItem
                      key={action.status}
                      onClick={() => setStatusAction(action)}
                      className={action.status !== "active" ? "text-destructive focus:text-destructive" : undefined}
                    >
                      <action.icon className="mr-2 size-4" />
                      {action.label}
                    </DropdownMenuItem>
                  ))}
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={() => setShowDeleteDialog(true)} className="text-destructive focus:text-destructive">
                    <Trash2 className="mr-2 size-4" />
                    Delete Tenant
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
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
        variant={statusAction && statusAction.status !== "active" ? "destructive" : "default"}
        confirmText={statusAction?.label}
        isLoading={updateStatusMutation.isPending}
      />

      <DeleteConfirmationDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        onConfirm={handleDelete}
        title="Delete Tenant"
        description="This action cannot be undone. This will permanently delete the tenant and all associated users, roles, and configuration."
        itemName={tenant.display_name}
        isDeleting={deleteTenantMutation.isPending}
      />
    </>
  )
}
