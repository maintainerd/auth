import type { ReactNode } from "react"
import type { LucideIcon } from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"

export interface DetailAttribute {
  icon: LucideIcon
  label: string
  value: ReactNode
}

interface DetailHeaderCardProps {
  /** Optional leading visual — an avatar tile for users, an icon tile, or nothing. */
  leading?: ReactNode
  title: ReactNode
  /** A status pill or similar shown next to the title. */
  badge?: ReactNode
  subtitle?: ReactNode
  /** Action buttons / dropdown shown on the right. */
  actions?: ReactNode
  /** Optional key/value grid rendered below a separator. */
  attributes?: DetailAttribute[]
}

/**
 * The shared summary header for detail pages: a leading slot + title + badge +
 * subtitle + actions, with an optional attribute grid. Every entity reuses this
 * shell; only the slots differ (e.g. users pass an avatar, others pass nothing).
 */
export function DetailHeaderCard({
  leading,
  title,
  badge,
  subtitle,
  actions,
  attributes,
}: DetailHeaderCardProps) {
  return (
    <Card>
      <CardContent>
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div className="flex min-w-0 items-center gap-4">
            {leading}
            <div className="min-w-0 space-y-1">
              <div className="flex flex-wrap items-center gap-2.5">
                <h1 className="text-lg font-semibold tracking-tight">{title}</h1>
                {badge}
              </div>
              {subtitle && <div className="text-sm text-muted-foreground break-words">{subtitle}</div>}
            </div>
          </div>
          {actions && <div className="flex items-center gap-2">{actions}</div>}
        </div>

        {attributes && attributes.length > 0 && (
          <>
            <Separator className="my-5" />
            <div className="grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              {attributes.map((attr) => (
                <DetailAttributeItem key={attr.label} {...attr} />
              ))}
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}

function DetailAttributeItem({ icon: Icon, label, value }: DetailAttribute) {
  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-muted-foreground">
        <Icon className="size-3.5" />
        {label}
      </div>
      <div className="text-sm text-foreground">{value}</div>
    </div>
  )
}
