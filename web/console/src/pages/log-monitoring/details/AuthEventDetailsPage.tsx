import { useParams, useNavigate } from "react-router-dom"
import { format } from "date-fns"
import { ShieldCheck, ShieldAlert, MapPin, Clock, Activity, FileText } from "lucide-react"
import { DetailLayout, StatusBadge, type DetailAttribute } from "@/components/details"
import { DetailHeaderCard } from "@/components/details"
import { InformationCard } from "@/components/card/InformationCard"
import { useAuthEvent } from "@/hooks/useAuthEvents"

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

export default function AuthEventDetailsPage() {
  const { eventId } = useParams<{ eventId: string }>()
  const navigate = useNavigate()

  const { data: event, isLoading, isError } = useAuthEvent(eventId || "")

  const attributes: DetailAttribute[] = [
    { icon: Activity, label: "Category", value: event?.category ?? "" },
    { icon: event?.severity === "CRITICAL" ? ShieldAlert : ShieldCheck, label: "Severity", value: event?.severity ?? "" },
    { icon: ShieldCheck, label: "Result", value: event?.result ?? "" },
    { icon: MapPin, label: "IP Address", value: event?.ip_address ?? "" },
    { icon: Clock, label: "Timestamp", value: event ? format(new Date(event.created_at), "PPpp") : "" },
  ]

  return (
    <DetailLayout
      backLabel="Back to Monitoring"
      onBack={() => navigate(`/monitoring?tab=logs`)}
      isLoading={isLoading}
      isError={isError || !event}
      notFoundTitle="Event not found"
      notFoundDescription="The auth event you're looking for doesn't exist or may have been removed."
    >
      {event && (
        <>
          <DetailHeaderCard
            title={event.event_type}
            badge={<StatusBadge status={event.result === "success" ? "active" : "inactive"} />}
            subtitle={event.description ?? undefined}
            attributes={attributes}
          />

          <InformationCard
            title="Event Details"
            icon={FileText}
            description="Full information about this authentication event."
          >
            <dl className="space-y-4">
              <InfoRow label="Event ID" value={<span className="font-mono text-xs break-all">{event.auth_event_id}</span>} />
              {event.error_reason && (
                <InfoRow
                  label="Error reason"
                  value={<span className="text-destructive">{event.error_reason}</span>}
                />
              )}
              {event.user_agent && (
                <InfoRow label="User agent" value={<span className="break-all text-xs">{event.user_agent}</span>} />
              )}
              {event.trace_id && (
                <InfoRow label="Trace ID" value={<code className="text-xs">{event.trace_id}</code>} />
              )}
              {event.metadata && Object.keys(event.metadata).length > 0 && (
                <InfoRow
                  label="Metadata"
                  value={
                    <pre className="max-h-48 overflow-auto rounded bg-muted p-2 text-xs">
                      {JSON.stringify(event.metadata, null, 2)}
                    </pre>
                  }
                />
              )}
            </dl>
          </InformationCard>
        </>
      )}
    </DetailLayout>
  )
}
