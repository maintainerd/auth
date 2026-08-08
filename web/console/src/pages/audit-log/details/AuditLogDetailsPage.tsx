import { useParams, useNavigate } from "react-router-dom"
import { format } from "date-fns"
import { ShieldCheck, ShieldAlert, MapPin, Clock, FileText, Activity } from "lucide-react"
import { DetailLayout, type DetailAttribute } from "@/components/details"
import { DetailHeaderCard } from "@/components/details"
import { InformationCard } from "@/components/card/InformationCard"
import { Badge } from "@/components/ui/badge"
import { useAuditLogEntry } from "@/hooks/useAuditLog"
import { formatAuditActor } from "../formatActor"

export default function AuditLogDetailsPage() {
  const { uuid } = useParams<{ uuid: string }>()
  const navigate = useNavigate()

  const { data: entry, isLoading, isError } = useAuditLogEntry(uuid || "")
  const actor = entry ? formatAuditActor(entry) : null

  const attributes: DetailAttribute[] = entry
    ? [
        { icon: Activity, label: "Action", value: entry.action },
        {
          icon: entry.outcome === "success" ? ShieldCheck : ShieldAlert,
          label: "Outcome",
          value: <OutcomeBadge outcome={entry.outcome} />,
        },
        { icon: MapPin, label: "IP Address", value: entry.ip_address || "—" },
        { icon: Clock, label: "Timestamp", value: format(new Date(entry.created_at), "PPpp") },
      ]
    : []

  return (
    <DetailLayout
      backLabel="Back to Audit Log"
      onBack={() => navigate(`/monitoring?tab=audit`)}
      isLoading={isLoading}
      isError={isError || !entry}
      notFoundTitle="Audit entry not found"
      notFoundDescription="The audit log entry you're looking for doesn't exist or may have been removed."
    >
      {entry && (
        <>
          <DetailHeaderCard
            leading={
              <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
                <FileText className="size-6" />
              </div>
            }
            title={entry.action}
            badge={<OutcomeBadge outcome={entry.outcome} />}
            subtitle={<span className="font-mono text-xs text-muted-foreground">{entry.uuid}</span>}
            attributes={attributes}
          />

          <InformationCard title="Details" icon={FileText}>
            <dl className="space-y-4">
              <InfoRow label="Resource Type" value={entry.resource_type} />
              <InfoRow
                label="Resource ID"
                value={<span className="font-mono text-xs break-all">{entry.resource_id || "—"}</span>}
              />
              <InfoRow
                label="Actor"
                value={
                  <span>
                    {actor?.label}
                    {actor?.context && actor.context !== "No actor" && (
                      <span className="ml-2 text-xs text-muted-foreground">{actor.context}</span>
                    )}
                  </span>
                }
              />
            </dl>
          </InformationCard>

          {entry.changes && (
            <InformationCard title="Changes" icon={FileText}>
              <pre className="overflow-auto rounded-md bg-muted p-3 text-xs">
                {formatChangesJson(entry.changes)}
              </pre>
            </InformationCard>
          )}
        </>
      )}
    </DetailLayout>
  )
}

function formatChangesJson(changes: unknown): string {
  const parsed = parseJsonStrings(changes)
  const formatted = JSON.stringify(parsed, null, 2)
  return formatted ?? String(parsed)
}

function parseJsonStrings(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(parseJsonStrings)

  if (isPlainObject(value)) {
    return Object.fromEntries(
      Object.entries(value).map(([key, nestedValue]) => [key, parseJsonStrings(nestedValue)]),
    )
  }

  if (typeof value !== "string") return value

  const trimmed = value.trim()
  if (!trimmed || !["{", "["].includes(trimmed[0])) return value

  try {
    return parseJsonStrings(JSON.parse(trimmed))
  } catch {
    return value
  }
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

function OutcomeBadge({ outcome }: { outcome: string }) {
  return (
    <Badge
      variant={outcome === "success" ? "default" : outcome === "failure" ? "destructive" : "secondary"}
      className="text-xs font-normal capitalize"
    >
      {outcome}
    </Badge>
  )
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1 sm:flex-row sm:items-baseline sm:gap-4">
      <dt className="w-44 shrink-0 text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </dt>
      <dd className="min-w-0 text-sm">{value}</dd>
    </div>
  )
}
