import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Edit, Trash2, MoreVertical, KeyRound, Building2, CalendarDays, Globe2, Play, Pause } from "lucide-react"
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
import { ProviderLogo } from "@/components/provider-config"
import { useDeleteIdentityProvider, useUpdateIdentityProviderStatus } from "@/hooks/useIdentityProviders"
import { useToast } from "@/hooks/useToast"
import { safeFormat } from "@/lib/formatDate"
import { getProviderDisplayName } from "../utils"
import type { IdentityProviderDetail, IdentityProviderStatus } from "@/services/api/identity-providers/types"

interface IdentityProviderHeaderProps {
  provider: IdentityProviderDetail
  providerId: string
}

export function IdentityProviderHeader({ provider, providerId }: IdentityProviderHeaderProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const deleteProviderMutation = useDeleteIdentityProvider()
  const updateStatusMutation = useUpdateIdentityProviderStatus()
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [statusAction, setStatusAction] = useState<{ status: IdentityProviderStatus; title: string; description: string } | null>(null)

  const handleDelete = async () => {
    try {
      await deleteProviderMutation.mutateAsync(providerId)
      showSuccess("Identity provider deleted successfully")
      navigate(`/providers/identity`)
    } catch (error) {
      showError(error)
    }
  }

  const handleStatusChange = async () => {
    if (!statusAction) return
    try {
      await updateStatusMutation.mutateAsync({ identityProviderId: providerId, data: { status: statusAction.status } })
      showSuccess(`Identity provider status updated to ${statusAction.status}`)
    } catch (error) {
      showError(error)
    } finally {
      setStatusAction(null)
    }
  }

  // Availability mirrors the backend rules and the listing (IdentityProviderActions):
  // system providers can't change status or be deleted; the default provider also
  // can't be deactivated or deleted.
  const isActive = provider.status === "active"
  const canActivate = !provider.is_system && !isActive
  const canDeactivate = !provider.is_system && isActive && !provider.is_default
  const canDelete = !provider.is_system && !provider.is_default
  const hasMenu = canActivate || canDeactivate || canDelete

  const providerLabel = getProviderDisplayName(provider.provider, provider.is_system)

  const attributes: DetailAttribute[] = [
    {
      icon: KeyRound,
      label: "System Name",
      value: <span className="font-mono text-xs">{provider.name}</span>,
    },
    {
      icon: KeyRound,
      label: "Identifier",
      value: <span className="font-mono text-xs">{provider.identifier}</span>,
    },
    {
      icon: Building2,
      label: "Provider Type",
      value: providerLabel,
    },
    {
      icon: Globe2,
      label: "Issuer",
      value: provider.issuer ? (
        <span className="break-all font-mono text-xs">{provider.issuer}</span>
      ) : (
        <span className="text-muted-foreground">Built-in</span>
      ),
    },
    {
      icon: CalendarDays,
      label: "Created",
      value: safeFormat(provider.created_at, "PP"),
    },
    {
      icon: CalendarDays,
      label: "Last updated",
      value: safeFormat(provider.updated_at, "PP"),
    },
  ]

  return (
    <>
      <DetailHeaderCard
        leading={<ProviderLogo provider={provider.provider} className="size-14" iconClassName="size-6" />}
        title={provider.display_name}
        badge={
          <div className="flex items-center gap-2">
            <StatusBadge status={provider.status} />
            <SystemBadge isSystem={provider.is_system} />
          </div>
        }
        subtitle={<span className="text-sm font-medium text-muted-foreground">{providerLabel}</span>}
        attributes={attributes}
        actions={
          <>
            <Button
              data-md-details-edit-button
              variant="outline"
              size="sm"
              className="h-9 gap-2"
              onClick={() =>
                navigate(`/providers/identity/${providerId}/edit`, {
                  state: { from: `/providers/identity/${providerId}`, backLabel: "Back to Provider Details" },
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
                          title: "Activate Identity Provider",
                          description: "Are you sure you want to activate this identity provider? Users will be able to authenticate through it.",
                        })
                      }
                    >
                      <Play className="mr-2 size-4" />
                      Activate Provider
                    </DropdownMenuItem>
                  )}
                  {canDeactivate && (
                    <DropdownMenuItem
                      onClick={() =>
                        setStatusAction({
                          status: "inactive",
                          title: "Deactivate Identity Provider",
                          description: "Are you sure you want to deactivate this identity provider? Users will not be able to authenticate through it.",
                        })
                      }
                      className="text-destructive focus:text-destructive"
                    >
                      <Pause className="mr-2 size-4" />
                      Deactivate Provider
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
                        Delete Provider
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
        title="Delete Identity Provider"
        description="This action cannot be undone. This will permanently delete the identity provider and all associated data."
        itemName={provider.name}
        isDeleting={deleteProviderMutation.isPending}
      />
    </>
  )
}
