import { Badge } from "@/components/ui/badge"
import { Shield } from "lucide-react"

interface SystemBadgeProps {
  isSystem: boolean
  className?: string
}

export function SystemBadge({ isSystem, className }: SystemBadgeProps) {
  if (!isSystem) return null

  return (
    <Badge variant="secondary" className={`text-xs ${className || ""}`}>
      <Shield className="h-3 w-3 mr-1" />
      System
    </Badge>
  )
}

