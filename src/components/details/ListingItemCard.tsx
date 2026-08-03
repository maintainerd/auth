import type { ComponentProps, ReactNode } from "react"
import type { LucideIcon } from "lucide-react"
import { cn } from "@/lib/utils"

type ListingItemCardProps = ComponentProps<"div"> & {
  icon?: LucideIcon
  action?: ReactNode
  children: ReactNode
}

export function ListingItemCard({
  icon: Icon,
  action,
  children,
  className,
  ...props
}: ListingItemCardProps) {
  return (
    <div
      data-md-listing-item
      className={cn("flex items-start justify-between gap-3 rounded-lg border p-4", className)}
      {...props}
    >
      <div className="flex min-w-0 flex-1 items-start gap-3">
        {Icon && (
          <div data-md-listing-icon className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
            <Icon className="size-4" />
          </div>
        )}
        <div className="min-w-0 flex-1">{children}</div>
      </div>
      {action}
    </div>
  )
}

export function ListingItemIcon({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      data-md-listing-icon
      className={cn("flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground", className)}
      {...props}
    />
  )
}

export function ListingItemMeta({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      data-md-listing-meta
      className={cn("flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground", className)}
      {...props}
    />
  )
}

export function ListingItemNested({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      data-md-listing-nested
      className={cn("space-y-2 rounded-md border bg-muted/30 p-3", className)}
      {...props}
    />
  )
}
