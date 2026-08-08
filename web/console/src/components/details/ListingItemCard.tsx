import type { ComponentProps, ComponentPropsWithoutRef, ReactNode } from "react"
import type { LucideIcon } from "lucide-react"
import { cn } from "@/lib/utils"

type ListingItemBaseProps = {
  icon?: LucideIcon
  action?: ReactNode
  actionClassName?: string
  children: ReactNode
  contentClassName?: string
  iconClassName?: string
}

type ListingItemCardProps =
  | ({ as?: "div" } & Omit<ComponentPropsWithoutRef<"div">, keyof ListingItemBaseProps> & ListingItemBaseProps)
  | ({ as: "button" } & Omit<ComponentPropsWithoutRef<"button">, keyof ListingItemBaseProps> & ListingItemBaseProps)

function ListingItemContent({
  icon: Icon,
  action,
  actionClassName,
  children,
  contentClassName,
  iconClassName,
}: ListingItemBaseProps) {
  return (
    <>
      <div className={cn("flex min-w-0 flex-1 items-start gap-3", contentClassName)}>
        {Icon && (
          <ListingItemIcon className={iconClassName}>
            <Icon className="size-4" />
          </ListingItemIcon>
        )}
        <div className="min-w-0 flex-1">{children}</div>
      </div>
      {action && <div className={cn("shrink-0", actionClassName)}>{action}</div>}
    </>
  )
}

export function ListingItemCard({
  as = "div",
  icon: Icon,
  action,
  actionClassName,
  children,
  className,
  contentClassName,
  iconClassName,
  ...props
}: ListingItemCardProps) {
  const itemClassName = cn(
    "flex items-start justify-between gap-3 rounded-lg border p-4",
    as === "button" &&
      "w-full text-left transition-colors hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50",
    className,
  )
  const content = (
    <ListingItemContent
      icon={Icon}
      action={action}
      actionClassName={actionClassName}
      contentClassName={contentClassName}
      iconClassName={iconClassName}
    >
      {children}
    </ListingItemContent>
  )

  if (as === "button") {
    return (
      <button
        type="button"
        data-md-listing-item
        className={itemClassName}
        {...(props as ComponentPropsWithoutRef<"button">)}
      >
        {content}
      </button>
    )
  }

  return (
    <div
      data-md-listing-item
      className={itemClassName}
      {...(props as ComponentPropsWithoutRef<"div">)}
    >
      {content}
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
