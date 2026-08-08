import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

export interface SettingsCardProps {
  title: string
  description?: string
  icon?: LucideIcon
  action?: ReactNode
  children: ReactNode
  className?: string
  contentClassName?: string
}

export function SettingsCard({
  title,
  description,
  icon: Icon,
  action,
  children,
  className,
  contentClassName,
}: SettingsCardProps) {
  return (
    <Card className={className}>
      <CardHeader>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0 space-y-1.5">
            <CardTitle className="flex items-center gap-2 text-base">
              {Icon && <Icon className="size-4 shrink-0" />}
              <span className="min-w-0 truncate">{title}</span>
            </CardTitle>
            {description && <CardDescription>{description}</CardDescription>}
          </div>
          {action && <div className="shrink-0 sm:pt-0.5">{action}</div>}
        </div>
      </CardHeader>
      <CardContent className={contentClassName}>
        {children}
      </CardContent>
    </Card>
  )
}
