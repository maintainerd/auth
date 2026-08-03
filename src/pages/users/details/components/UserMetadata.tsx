import { Braces } from "lucide-react"
import { InformationCard } from "@/components/card"
import { EmptyState } from "@/components/details"
import type { User } from "@/services/api/users/types"

interface UserMetadataProps {
  user: User
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return "—"
  if (typeof value === "boolean") return value ? "true" : "false"
  if (Array.isArray(value)) return value.join(", ")
  if (typeof value === "object") return JSON.stringify(value)
  return String(value)
}

export function UserMetadata({ user }: UserMetadataProps) {
  const metadata = user.metadata || {}
  const entries = Object.entries(metadata)

  return (
    <InformationCard
      title="Custom Metadata"
      description="Extended data attached to this user — external identifiers and any extra attributes an application needs."
      icon={Braces}
    >
      {entries.length === 0 ? (
        <EmptyState
          icon={Braces}
          title="No metadata"
          description="This user has no custom metadata. Applications can attach external identifiers or extended attributes here."
        />
      ) : (
        <div data-md-listing-nested className="divide-y rounded-lg border">
          {entries.map(([key, value]) => (
            <div
              key={key}
              className="grid grid-cols-1 gap-1 px-4 py-3 sm:grid-cols-[minmax(0,14rem)_1fr] sm:gap-4"
            >
              <div className="text-sm font-medium break-all">{key}</div>
              <div className="font-mono text-sm text-muted-foreground break-all">{formatValue(value)}</div>
            </div>
          ))}
        </div>
      )}
    </InformationCard>
  )
}
