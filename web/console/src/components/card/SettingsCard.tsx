/**
 * Reusable Settings Card Component
 * A card wrapper for settings sections with icon, title, and description
 */

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import type { ReactNode } from "react"
import type { LucideIcon } from "lucide-react"

export interface SettingsCardProps {
  title: string
  description?: string
  icon?: LucideIcon
  children: ReactNode
  className?: string
  contentClassName?: string
}

export function SettingsCard({ 
  title, 
  description, 
  icon: Icon, 
  children,
  className,
  contentClassName
}: SettingsCardProps) {
  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          {Icon && <Icon className="h-5 w-5" />}
          {title}
        </CardTitle>
        {description && (
          <CardDescription>
            {description}
          </CardDescription>
        )}
      </CardHeader>
      <CardContent className={contentClassName}>
        {children}
      </CardContent>
    </Card>
  )
}
