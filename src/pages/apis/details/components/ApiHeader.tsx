import { useState } from "react"
import { useNavigate } from "react-router-dom"
import {
  CalendarDays,
  Edit,
  KeyRound,
  MoreVertical,
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
import { useDeleteApi, useUpdateApiStatus } from "@/hooks/useApis"
import { useToast } from "@/hooks/useToast"
import { safeFormat } from "@/lib/formatDate"
import type { Api, ApiStatus } from "@/services/api/api/types"

interface ApiHeaderProps {
  api: Api
  apiId: string
}

interface StatusAction {
  status: ApiStatus
  label: string
  title: string
  description: string
  icon: typeof Play
}

const STATUS_ACTIONS: Record<ApiStatus, StatusAction> = {
  active: {
    status: "active",
    label: "Activate API",
    title: "Activate API",
    description: "Are you sure you want to activate this API? Its permissions will be available for use.",
    icon: Play,
  },
  inactive: {
    status: "inactive",
    label: "Deactivate API",
    title: "Deactivate API",
    description: "Are you sure you want to deactivate this API? Its permissions will no longer be available.",
    icon: XCircle,
  },
}

export function ApiHeader({ api, apiId }: ApiHeaderProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const deleteApiMutation = useDeleteApi()
  const updateStatusMutation = useUpdateApiStatus()
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [statusAction, setStatusAction] = useState<StatusAction | null>(null)

  const handleDelete = async () => {
    try {
      await deleteApiMutation.mutateAsync(apiId)
      showSuccess("API deleted successfully")
      navigate(`/apis`)
    } catch (error) {
      showError(error)
    }
  }

  const handleStatusChange = async () => {
    if (!statusAction) return
    try {
      await updateStatusMutation.mutateAsync({ apiId, data: { status: statusAction.status } })
      showSuccess(`API status updated to ${statusAction.status}`)
    } catch (error) {
      showError(error)
    } finally {
      setStatusAction(null)
    }
  }

  // Availability mirrors the backend rules: system APIs cannot be edited,
  // change status, or be deleted.
  const statusActions = api.is_system
    ? []
    : Object.values(STATUS_ACTIONS).filter((action) => action.status !== api.status)

  const attributes: DetailAttribute[] = [
    {
      icon: Server,
      label: "API Name",
      value: <span className="font-mono text-xs">{api.name}</span>,
    },
    {
      icon: KeyRound,
      label: "Identifier",
      value: <span className="font-mono text-xs">{api.identifier}</span>,
    },
    {
      icon: Server,
      label: "Service",
      value: api.service?.display_name ?? <span className="text-muted-foreground">&mdash;</span>,
    },
    {
      icon: CalendarDays,
      label: "Created",
      value: safeFormat(api.created_at, "PP"),
    },
    {
      icon: CalendarDays,
      label: "Last updated",
      value: safeFormat(api.updated_at, "PP"),
    },
    {
      icon: Server,
      label: "Type",
      value: api.is_system ? "System API" : "Custom API",
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
        title={api.display_name}
        badge={
          <div className="flex items-center gap-2">
            <StatusBadge status={api.status} />
            <SystemBadge isSystem={api.is_system} />
          </div>
        }
        subtitle={api.description}
        attributes={attributes}
        actions={
          !api.is_system && (
            <>
              <Button
                data-md-details-edit-button
                variant="outline"
                size="sm"
                className="h-9 gap-2"
                onClick={() =>
                  navigate(`/apis/${apiId}/edit`, {
                    state: { from: `/apis/${apiId}`, backLabel: "Back to API Details" },
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
                    Delete API
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
        // Deactivate takes the API out of use → red confirm. Activate is
        // restorative → normal confirm.
        variant={statusAction && statusAction.status !== "active" ? "destructive" : "default"}
        confirmText={statusAction?.label}
        isLoading={updateStatusMutation.isPending}
      />

      <DeleteConfirmationDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        onConfirm={handleDelete}
        title="Delete API"
        description="This action cannot be undone. This will permanently delete the API and all associated data."
        itemName={api.name}
        isDeleting={deleteApiMutation.isPending}
      />
    </>
  )
}
