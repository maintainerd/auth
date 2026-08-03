import { cn } from "@/lib/utils"

// Canonical status pill for the whole app. Common status keywords → dot colour;
// any entity's status string maps through here and unknown values fall back to a
// neutral dot. Colour is carried by a small dot (not a pastel block), which
// reads enterprise rather than "vibecoded". This is the single StatusBadge — the
// pastel/icon variant was retired so every listing, header, and tab matches.
const STATUS_DOT: Record<string, string> = {
  active: "bg-emerald-500",
  enabled: "bg-emerald-500",
  verified: "bg-emerald-500",
  accepted: "bg-emerald-500",
  allow: "bg-emerald-500",
  pending: "bg-amber-500",
  draft: "bg-amber-500",
  configuring: "bg-amber-500",
  maintenance: "bg-amber-500",
  inactive: "bg-muted-foreground",
  disabled: "bg-muted-foreground",
  archived: "bg-muted-foreground",
  expired: "bg-muted-foreground",
  deny: "bg-red-500",
  suspended: "bg-red-500",
  blocked: "bg-red-500",
  revoked: "bg-red-500",
  quarantined: "bg-red-500",
  deprecated: "bg-red-500",
}

/** A restrained, entity-agnostic status pill: a small coloured dot + label. The
 *  `status` keyword selects the tone (theme badge config); `label` overrides the
 *  displayed text when it differs from the keyword. */
export function StatusBadge({
  status,
  label,
  className,
}: {
  status: string
  label?: string
  className?: string
}) {
  const normalizedStatus = status?.toLowerCase() ?? ""
  const dot = STATUS_DOT[normalizedStatus] ?? "bg-muted-foreground"
  return (
    <span
      data-md-status-badge
      data-status={normalizedStatus}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md border bg-background px-2 py-0.5 text-xs font-medium capitalize",
        className,
      )}
    >
      <span data-md-status-dot className={cn("size-1.5 rounded-full", dot)} />
      {label ?? status}
    </span>
  )
}
