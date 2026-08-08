import { useState } from "react"
import { useNavigate } from "react-router-dom"
import {
  Archive,
  CalendarDays,
  Edit,
  MoreVertical,
  Pause,
  Play,
  Server,
  Trash2,
  XCircle,
} from "lucide-react"
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
import { SystemBadge } from "@/components/badges"
import { useDeleteService, useUpdateServiceStatus } from "@/hooks/useServices"
import { useToast } from "@/hooks/useToast"
import { safeFormat } from "@/lib/formatDate"
import type { ServiceResponse, ServiceStatus } from "@/services/api/services/types"

interface ServiceHeaderProps {
  service: ServiceResponse
  serviceId: string
}

interface StatusAction {
  status: ServiceStatus
  label: string
  title: string
  description: string
  icon: typeof Play
}

const STATUS_ACTIONS: Record<ServiceStatus, StatusAction> = {
  active: {
    status: "active",
    label: "Activate Service",
    title: "Activate Service",
    description: "Are you sure you want to activate this service? Its APIs and policies will be available for use.",
    icon: Play,
  },
  maintenance: {
    status: "maintenance",
    label: "Set Maintenance",
    title: "Set Maintenance Mode",
    description: "Are you sure you want to put this service into maintenance mode?",
    icon: Pause,
  },
  deprecated: {
    status: "deprecated",
    label: "Deprecate Service",
    title: "Deprecate Service",
    description: "Are you sure you want to mark this service as deprecated?",
    icon: Archive,
  },
  inactive: {
    status: "inactive",
    label: "Deactivate Service",
    title: "Deactivate Service",
    description: "Are you sure you want to deactivate this service? Its APIs and policies will no longer be available.",
    icon: XCircle,
  },
}

export function ServiceHeader({ service, serviceId }: ServiceHeaderProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const deleteServiceMutation = useDeleteService()
  const updateStatusMutation = useUpdateServiceStatus()
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [statusAction, setStatusAction] = useState<StatusAction | null>(null)

  const handleDelete = async () => {
    try {
      await deleteServiceMutation.mutateAsync(serviceId)
      showSuccess("Service deleted successfully")
      navigate(`/services`)
    } catch (error) {
      showError(error)
    }
  }

  const handleStatusChange = async () => {
    if (!statusAction) return
    try {
      await updateStatusMutation.mutateAsync({ serviceId, data: { status: statusAction.status } })
      showSuccess(`Service status updated to ${statusAction.status}`)
    } catch (error) {
      showError(error)
    } finally {
      setStatusAction(null)
    }
  }

  // Availability mirrors the backend rules: system services cannot be edited,
  // change status, or be deleted.
  const statusActions = service.is_system
    ? []
    : Object.values(STATUS_ACTIONS).filter((action) => action.status !== service.status)

  const attributes: DetailAttribute[] = [
    {
      icon: Server,
      label: "Service Name",
      value: <span className="font-mono text-xs">{service.name}</span>,
    },
    {
      icon: Server,
      label: "Version",
      value: <span className="font-mono text-xs">{service.version}</span>,
    },
    {
      icon: CalendarDays,
      label: "Created",
      value: safeFormat(service.created_at, "PP"),
    },
    {
      icon: CalendarDays,
      label: "Last updated",
      value: safeFormat(service.updated_at, "PP"),
    },
    {
      icon: Server,
      label: "Type",
      value: service.is_system ? "System Service" : "Custom Service",
    },
  ]

  return (
    <>
      <DetailHeaderCard
        leading={
          <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
            <Server className="size-6" />
          </div>
        }
        title={service.display_name}
        badge={
          <div className="flex items-center gap-2">
            <StatusBadge status={service.status} />
            <SystemBadge isSystem={service.is_system} />
          </div>
        }
        subtitle={service.description}
        attributes={attributes}
        actions={
          !service.is_system && (
            <>
              <Button
                data-md-details-edit-button
                variant="outline"
                size="sm"
                className="h-9 gap-2"
                onClick={() =>
                  navigate(`/services/${serviceId}/edit`, {
                    state: { from: `/services/${serviceId}`, backLabel: "Back to Service Details" },
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
                      className={
                        action.status !== "active"
                          ? "text-destructive focus:text-destructive"
                          : undefined
                      }
                    >
                      <action.icon className="mr-2 size-4" />
                      {action.label}
                    </DropdownMenuItem>
                  ))}
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={() => setShowDeleteDialog(true)}
                    className="text-destructive focus:text-destructive"
                  >
                    <Trash2 className="mr-2 size-4" />
                    Delete Service
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
        // Everything except Activate takes the service (partly) out of use →
        // red confirm. Activate is restorative → normal confirm.
        variant={statusAction && statusAction.status !== "active" ? "destructive" : "default"}
        confirmText={statusAction?.label}
        isLoading={updateStatusMutation.isPending}
      />

      <DeleteConfirmationDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        onConfirm={handleDelete}
        title="Delete Service"
        description="This action cannot be undone. This will permanently delete the service and all associated data."
        itemName={service.name}
        isDeleting={deleteServiceMutation.isPending}
      />
    </>
  )
}
