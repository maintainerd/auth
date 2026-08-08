import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { ReactNode } from "react"
import type { LucideIcon } from "lucide-react"

interface InformationCardProps {
  title: string
  description?: string
  icon?: LucideIcon
  action?: ReactNode
  children: ReactNode
}

export function InformationCard({ title, description, icon: Icon, action, children }: InformationCardProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="min-w-0 flex-1">
            <CardTitle className={Icon ? "flex items-center gap-2 text-base" : "text-base"}>
              {Icon && <Icon className="size-4" />}
              {title}
            </CardTitle>
            {description && (
              <p className="text-sm text-muted-foreground mt-1.5">{description}</p>
            )}
          </div>
          {action && <div>{action}</div>}
        </div>
      </CardHeader>
      <CardContent>
        {children}
      </CardContent>
    </Card>
  )
}

