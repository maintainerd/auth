import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Edit, Trash2, MoreVertical, Palette, CalendarDays, CheckCircle2, RotateCcw } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { DeleteConfirmationDialog, ConfirmationDialog } from "@/components/dialog"
import { DetailHeaderCard, StatusBadge, type DetailAttribute } from "@/components/details"
import { useDeleteBranding, useActivateBranding, useRestoreBranding } from "@/hooks/useBranding"
import { useToast } from "@/hooks/useToast"
import { format } from "date-fns"
import { isHex } from "../../themeTokens"
import type { Branding } from "@/services/api/branding/types"

interface BrandingHeaderProps {
  branding: Branding
  brandingId: string
}

function primaryColor(b: Branding): string {
  const colors = b.metadata?.colors as Record<string, unknown> | undefined
  const value = colors?.primary
  if (typeof value === "string" && isHex(value)) return value
  return "#e2e8f0"
}

export function BrandingHeader({ branding, brandingId }: BrandingHeaderProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const deleteMutation = useDeleteBranding()
  const activateMutation = useActivateBranding()
  const restoreMutation = useRestoreBranding()
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [showActivateDialog, setShowActivateDialog] = useState(false)
  const [showRestoreDialog, setShowRestoreDialog] = useState(false)

  const listUrl = `/branding?tab=themes`
  const detailBase = `/branding/templates`

  const handleDelete = async () => {
    try {
      await deleteMutation.mutateAsync(brandingId)
      showSuccess("Branding deleted successfully")
      navigate(listUrl)
    } catch (error) {
      showError(error)
    }
  }

  const handleActivate = async () => {
    try {
      await activateMutation.mutateAsync(brandingId)
      showSuccess(`"${branding.name}" is now the active branding`)
    } catch (error) {
      showError(error)
    }
  }

  const handleRestore = async () => {
    try {
      await restoreMutation.mutateAsync(brandingId)
      showSuccess(`"${branding.name}" restored to the system default`)
    } catch (error) {
      showError(error)
    }
  }

  const attributes: DetailAttribute[] = [
    {
      icon: Palette,
      label: "Type",
      value: branding.is_system ? "System Template" : "Custom Template",
    },
    {
      icon: CalendarDays,
      label: "Created",
      value: format(new Date(branding.created_at), "PP"),
    },
    {
      icon: CalendarDays,
      label: "Last updated",
      value: format(new Date(branding.updated_at), "PP"),
    },
  ]

  return (
    <>
      <DetailHeaderCard
        leading={
          <div
            className="flex size-14 shrink-0 items-center justify-center rounded-xl border"
            style={{ backgroundColor: primaryColor(branding) }}
            aria-hidden
          />
        }
        title={branding.name}
        badge={<StatusBadge status={branding.is_active ? "active" : "inactive"} />}
        subtitle={branding.company_name || undefined}
        attributes={attributes}
        actions={
          <>
            <Button
              data-md-details-edit-button
              variant="outline"
              size="sm"
              className="h-9 gap-2"
              onClick={() =>
                navigate(`${detailBase}/${brandingId}/edit`, {
                  state: { from: `${detailBase}/${brandingId}`, backLabel: "Back to Branding Details" },
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
                {!branding.is_active && (
                  <DropdownMenuItem onClick={() => setShowActivateDialog(true)}>
                    <CheckCircle2 className="mr-2 size-4" />
                    Set as Active
                  </DropdownMenuItem>
                )}
                {branding.is_system && (
                  <DropdownMenuItem onClick={() => setShowRestoreDialog(true)}>
                    <RotateCcw className="mr-2 size-4" />
                    Restore
                  </DropdownMenuItem>
                )}
                {!branding.is_system && (
                  <DropdownMenuItem
                    onClick={() => setShowDeleteDialog(true)}
                    className="text-destructive focus:text-destructive"
                  >
                    <Trash2 className="mr-2 size-4" />
                    Delete Branding
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </>
        }
      />

      <ConfirmationDialog
        open={showActivateDialog}
        onOpenChange={setShowActivateDialog}
        onConfirm={handleActivate}
        title="Set as Active Branding"
        description={`Make "${branding.name}" the active branding? It becomes the loaded style and the current active template is deactivated.`}
        confirmText="Set as Active"
        isLoading={activateMutation.isPending}
      />

      <ConfirmationDialog
        open={showRestoreDialog}
        onOpenChange={setShowRestoreDialog}
        onConfirm={handleRestore}
        title="Restore System Theme"
        description={`Restore "${branding.name}" to the seeded system defaults? This replaces edited theme values but keeps the current active theme selection.`}
        confirmText="Restore Defaults"
        isLoading={restoreMutation.isPending}
      />

      <DeleteConfirmationDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        onConfirm={handleDelete}
        title="Delete Branding"
        description="This action cannot be undone. This will permanently delete the branding template."
        itemName={branding.name}
        isDeleting={deleteMutation.isPending}
      />
    </>
  )
}
