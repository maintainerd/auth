import { useState } from "react"
import { Link, useNavigate } from "react-router-dom"
import { Edit, Trash2, MoreVertical, Workflow, Box, CalendarDays, Play, Pause } from "lucide-react"
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
import { useDeleteRegistrationFlow, useUpdateRegistrationFlowStatus } from "@/hooks/useRegistrationFlows"
import { useToast } from "@/hooks/useToast"
import { safeFormat } from "@/lib/formatDate"
import type { RegistrationFlowDetail, RegistrationFlowStatus } from "@/services/api/registration-flows/types"

interface RegistrationFlowHeaderProps {
  /** The detail projection — the header renders the resolved nested client. */
  registrationFlow: RegistrationFlowDetail
  registrationFlowId: string
}

export function RegistrationFlowHeader({ registrationFlow, registrationFlowId }: RegistrationFlowHeaderProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const deleteMutation = useDeleteRegistrationFlow()
  const updateStatusMutation = useUpdateRegistrationFlowStatus()
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [statusAction, setStatusAction] = useState<{ status: RegistrationFlowStatus; title: string; description: string } | null>(null)

  const handleDelete = async () => {
    try {
      await deleteMutation.mutateAsync(registrationFlowId)
      showSuccess("Registration flow deleted successfully")
      navigate(`/registration-flows`)
    } catch (error) {
      showError(error)
    }
  }

  const handleStatusChange = async () => {
    if (!statusAction) return
    try {
      await updateStatusMutation.mutateAsync({ registrationFlowId, data: { status: statusAction.status } })
      showSuccess(`Registration flow ${statusAction.status === "active" ? "activated" : "deactivated"} successfully`)
    } catch (error) {
      showError(error)
    } finally {
      setStatusAction(null)
    }
  }

  // Availability mirrors the backend rules: system flows can't change status or be deleted.
  const isActive = registrationFlow.status === "active"
  const canActivate = !registrationFlow.is_system && !isActive
  const canDeactivate = !registrationFlow.is_system && isActive
  const canDelete = !registrationFlow.is_system
  const hasMenu = canActivate || canDeactivate || canDelete

  // Show the client by its human name (display_name preferred) with the OAuth
  // identifier as mono secondary text — a bare UUID told the operator nothing,
  // and the identifier is what actually appears in a registration link.
  const client = registrationFlow.client
  const clientLabel = client ? client.display_name || client.name : null

  const attributes: DetailAttribute[] = [
    {
      icon: Box,
      label: "Client",
      value: client ? (
        <div className="flex min-w-0 flex-col gap-0.5">
          <Link
            to={`/clients/${client.client_id}`}
            className="truncate font-medium underline underline-offset-4"
          >
            {clientLabel}
          </Link>
          <span className="break-all font-mono text-xs text-muted-foreground">{client.identifier}</span>
        </div>
      ) : (
        <span className="text-muted-foreground">—</span>
      ),
    },
    { icon: CalendarDays, label: "Created", value: safeFormat(registrationFlow.created_at, "PP") },
    { icon: CalendarDays, label: "Last updated", value: safeFormat(registrationFlow.updated_at, "PP") },
  ]

  return (
    <>
      <DetailHeaderCard
        leading={
          <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
            <Workflow className="size-6" />
          </div>
        }
        title={registrationFlow.name}
        badge={
          <div className="flex items-center gap-2">
            <StatusBadge status={registrationFlow.status} />
            <SystemBadge isSystem={registrationFlow.is_system} />
          </div>
        }
        subtitle={registrationFlow.description}
        attributes={attributes}
        actions={
          <>
            <Button
              data-md-details-edit-button
              variant="outline"
              size="sm"
              className="h-9 gap-2"
              onClick={() =>
                navigate(`/registration-flows/${registrationFlowId}/edit`, {
                  state: { from: `/registration-flows/${registrationFlowId}`, backLabel: "Back to Registration Flow Details" },
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
                          title: "Activate Registration Flow",
                          description: "Are you sure you want to activate this registration flow?",
                        })
                      }
                    >
                      <Play className="mr-2 size-4" />
                      Activate Registration Flow
                    </DropdownMenuItem>
                  )}
                  {canDeactivate && (
                    <DropdownMenuItem
                      onClick={() =>
                        setStatusAction({
                          status: "inactive",
                          title: "Deactivate Registration Flow",
                          description: "Are you sure you want to deactivate this registration flow?",
                        })
                      }
                      className="text-destructive focus:text-destructive"
                    >
                      <Pause className="mr-2 size-4" />
                      Deactivate Registration Flow
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
                        Delete Registration Flow
                      </DropdownMenuItem>
                    </>
                  )}
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </>
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
        title="Delete Registration Flow"
        description="This action cannot be undone. This will permanently delete the registration flow and all associated data."
        itemName={registrationFlow.name}
        isDeleting={deleteMutation.isPending}
      />
    </>
  )
}
