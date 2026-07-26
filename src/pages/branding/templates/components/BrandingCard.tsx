import { useNavigate } from "react-router-dom"
import { Card } from "@/components/ui/card"
import { StatusBadge } from "@/components/details"
import { SystemBadge } from "@/components/badges"
import { BrandingActions } from "./BrandingActions"
import { tokensFromMetadata, isHex } from "../themeTokens"
import type { Branding } from "@/services/api/branding/types"
import { authUiTemplateIdFromMetadata, getAuthUiTemplate } from "@/lib/branding/authUiTemplates"
import { BRANDING_THEMES_BACK_STATE } from "../brandingNavigation"

// The swatches shown in the card's preview band — a representative slice of the
// theme's palette.
const PREVIEW_KEYS = [
  "colors.primary",
  "colors.secondary",
  "colors.accent",
  "colors.appBackground",
  "colors.cardBackground",
  "colors.sidePanelBackground",
] as const

export function BrandingCard({ branding }: { branding: Branding }) {
  const navigate = useNavigate()

  const tokens = tokensFromMetadata(branding.metadata)
  const uiTemplate = getAuthUiTemplate(authUiTemplateIdFromMetadata(branding.metadata, branding.layout))
  const palette = PREVIEW_KEYS.map((k) => tokens[k]).filter(isHex)
  const open = () => navigate(`/branding/templates/${branding.branding_id}`, { state: BRANDING_THEMES_BACK_STATE })

  return (
    <Card
      role="button"
      tabIndex={0}
      onClick={open}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault()
          open()
        }
      }}
      className="cursor-pointer gap-0 overflow-hidden py-0 transition-colors hover:border-primary/40 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      {/* Palette preview band */}
      <div className="flex h-20 w-full border-b" aria-hidden>
        {palette.length > 0 ? (
          palette.map((color, i) => (
            <span key={i} className="flex-1" style={{ backgroundColor: color }} />
          ))
        ) : (
          <span className="flex-1 bg-muted" />
        )}
      </div>

      <div className="flex items-start justify-between gap-3 p-4">
        <div className="min-w-0 space-y-1">
          <div className="flex items-center gap-2">
            <span className="truncate font-medium">{branding.name || "Untitled"}</span>
            <SystemBadge isSystem={branding.is_system} />
          </div>
          <p className="truncate text-sm text-muted-foreground">{branding.company_name || "—"}</p>
          <p className="truncate text-xs text-muted-foreground">{uiTemplate.label}</p>
          <div className="pt-1">
            <StatusBadge status={branding.is_active ? "active" : "inactive"} />
          </div>
        </div>

        <div onClick={(e) => e.stopPropagation()}>
          <BrandingActions branding={branding} />
        </div>
      </div>
    </Card>
  )
}
