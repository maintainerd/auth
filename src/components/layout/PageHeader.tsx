import type { LucideIcon } from "lucide-react"

interface PageHeaderProps {
  title: string
  description?: string
  /** Optional icon rendered to the left of the title. */
  icon?: LucideIcon
}

export function PageHeader({ title, description, icon: Icon }: PageHeaderProps) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        {Icon && <Icon className="size-5 shrink-0 text-foreground" />}
        <h1 className="text-xl font-semibold tracking-tight">{title}</h1>
      </div>
      {description && (
        <p className="text-sm text-muted-foreground">{description}</p>
      )}
    </div>
  )
}
