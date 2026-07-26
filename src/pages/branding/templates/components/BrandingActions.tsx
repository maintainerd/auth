import { useNavigate } from "react-router-dom"
import { Eye, Edit, Trash2, CheckCircle2 } from "lucide-react"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useActivateBranding, useDeleteBranding } from "@/hooks/useBranding"
import { useToast } from "@/hooks/useToast"
import type { Branding } from "@/services/api/branding/types"
import { BRANDING_THEMES_BACK_STATE } from "../brandingNavigation"

interface BrandingActionsProps {
  branding: Branding
}

export function BrandingActions({ branding }: BrandingActionsProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const activateMutation = useActivateBranding()
  const deleteMutation = useDeleteBranding()

  const items: RowActionItem[] = [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () => navigate(`/branding/templates/${branding.branding_id}`, { state: BRANDING_THEMES_BACK_STATE }),
    },
    {
      key: "edit",
      label: "Edit Branding",
      icon: Edit,
      onSelect: () =>
        navigate(`/branding/templates/${branding.branding_id}/edit`, { state: BRANDING_THEMES_BACK_STATE }),
    },
    // System themes can be edited and activated but never deleted.
    ...(!branding.is_active
      ? [
          {
            key: "activate",
            label: "Set as Active",
            icon: CheckCircle2,
            onSelect: async () => {
              try {
                await activateMutation.mutateAsync(branding.branding_id)
                showSuccess(`"${branding.name}" is now the active branding`)
              } catch (error) {
                showError(error)
              }
            },
            confirm: {
              title: "Set as Active Branding",
              description: `Make "${branding.name}" the active branding? The current active template will be deactivated.`,
              confirmText: "Set as Active",
            },
          } satisfies RowActionItem,
        ]
      : []),
    ...(!branding.is_system
      ? [
          {
            key: "delete",
            label: "Delete Branding",
            icon: Trash2,
            destructive: true,
            separatorBefore: true,
            onSelect: async () => {
              try {
                await deleteMutation.mutateAsync(branding.branding_id)
                showSuccess("Branding deleted successfully")
              } catch (error) {
                showError(error)
              }
            },
            confirm: {
              title: "Delete Branding",
              description:
                "This will permanently delete this branding template. This action cannot be undone.",
              destructive: true,
              itemName: branding.name,
            },
          } satisfies RowActionItem,
        ]
      : []),
  ]

  return <RowActions items={items} />
}
