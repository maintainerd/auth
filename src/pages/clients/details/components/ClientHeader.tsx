import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { AppWindow, CalendarDays, Edit, Globe, KeyRound, MoreVertical, Palette, ShieldCheck, Trash2, Play, Pause } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
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
import { useDeleteClient, useUpdateClientStatus } from "@/hooks/useClients"
import { useBrandings } from "@/hooks/useBranding"
import { useToast } from "@/hooks/useToast"
import { safeFormat } from "@/lib/formatDate"
import type { ClientResponse, ClientStatus } from "@/services/api/clients/types"

interface ClientHeaderProps {
  client: ClientResponse
  clientId: string
}

const CLIENT_TYPE_LABELS: Record<string, string> = {
  traditional: "Traditional Web",
  mobile: "Native Mobile",
  spa: "Single Page App",
  m2m: "Machine to Machine",
}

export function ClientHeader({ client, clientId }: ClientHeaderProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const deleteClientMutation = useDeleteClient()
  const updateStatusMutation = useUpdateClientStatus()
  const { data: brandings } = useBrandings()
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [statusAction, setStatusAction] = useState<{ status: ClientStatus; title: string; description: string } | null>(null)

  const branding = client.branding_id
    ? brandings?.find((b) => b.branding_id === client.branding_id)
    : undefined

  const handleDelete = async () => {
    try {
      await deleteClientMutation.mutateAsync(clientId)
      showSuccess("Client deleted successfully")
      navigate(`/clients`)
    } catch (error) {
      showError(error)
    }
  }

  const handleStatusChange = async () => {
    if (!statusAction) return
    try {
      await updateStatusMutation.mutateAsync({ clientId, data: { status: statusAction.status } })
      showSuccess(`Client status updated to ${statusAction.status}`)
    } catch (error) {
      showError(error)
    } finally {
      setStatusAction(null)
    }
  }

  // Availability mirrors the backend rules: system clients can't change status or
  // be deleted; the default client also can't be deactivated or deleted.
  const isActive = client.status === "active"
  const canActivate = !client.is_system && !isActive
  const canDeactivate = !client.is_system && isActive && !client.is_default
  const canDelete = !client.is_system && !client.is_default
  const hasMenu = canActivate || canDeactivate || canDelete

  const attributes: DetailAttribute[] = [
    {
      icon: KeyRound,
      label: "Client UUID",
      value: <span className="font-mono text-xs">{client.client_id}</span>,
    },
    {
      icon: AppWindow,
      label: "Client Type",
      value: CLIENT_TYPE_LABELS[client.client_type] ?? client.client_type,
    },
    {
      icon: Globe,
      label: "Domain",
      value: client.domain ? (
        <span className="font-mono text-xs break-all">{client.domain}</span>
      ) : (
        <span className="text-muted-foreground">—</span>
      ),
    },
    {
      icon: Palette,
      label: "Branding",
      value: branding?.name ?? (client.branding_id ? "—" : "Tenant's active branding"),
    },
    {
      icon: ShieldCheck,
      label: "Allow registration",
      value: client.allow_registration !== false ? "Enabled" : "Disabled",
    },
    {
      icon: CalendarDays,
      label: "Created",
      value: safeFormat(client.created_at, "PP"),
    },
    {
      icon: CalendarDays,
      label: "Last updated",
      value: safeFormat(client.updated_at, "PP"),
    },
  ]

  return (
    <>
      <DetailHeaderCard
        leading={
          <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
            <AppWindow className="size-6" />
          </div>
        }
        title={client.display_name}
        badge={
          <div className="flex items-center gap-2">
            <StatusBadge status={client.status} />
            <SystemBadge isSystem={client.is_system} />
            {client.is_default && (
              <Badge variant="outline" className="text-xs">
                Default
              </Badge>
            )}
          </div>
        }
        subtitle={<span className="font-mono text-xs text-muted-foreground">{client.name}</span>}
        attributes={attributes}
        actions={
          <>
            <Button
              data-md-details-edit-button
              variant="outline"
              size="sm"
              className="h-9 gap-2"
              onClick={() =>
                navigate(`/clients/${clientId}/edit`, {
                  state: { from: `/clients/${clientId}`, backLabel: "Back to Client Details" },
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
                          title: "Activate Client",
                          description: "Are you sure you want to activate this client? Users will be able to authenticate using this client.",
                        })
                      }
                    >
                      <Play className="mr-2 size-4" />
                      Activate Client
                    </DropdownMenuItem>
                  )}
                  {canDeactivate && (
                    <DropdownMenuItem
                      onClick={() =>
                        setStatusAction({
                          status: "inactive",
                          title: "Deactivate Client",
                          description: "Are you sure you want to deactivate this client? Users will not be able to authenticate using this client.",
                        })
                      }
                      className="text-destructive focus:text-destructive"
                    >
                      <Pause className="mr-2 size-4" />
                      Deactivate Client
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
                        Delete Client
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
        title="Delete Client"
        description="This action cannot be undone. This will permanently delete the client and all associated data."
        itemName={client.name}
        isDeleting={deleteClientMutation.isPending}
      />
    </>
  )
}
