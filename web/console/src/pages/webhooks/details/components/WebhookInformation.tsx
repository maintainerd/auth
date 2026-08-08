import { Activity, RefreshCw, ShieldCheck, Timer } from "lucide-react"
import { InformationCard } from "@/components/card/InformationCard"
import { Badge } from "@/components/ui/badge"
import type { Webhook } from "@/services/api/webhooks/types"

interface WebhookInformationProps {
  webhook: Webhook
}

/** A small key/value row used inside the information card. */
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

export function WebhookInformation({ webhook }: WebhookInformationProps) {
  return (
    <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(20rem,24rem)]">
      <InformationCard
        title="Endpoint"
        icon={Activity}
        description="Delivery destination and subscription scope for this webhook."
      >
        <dl className="space-y-4">
          <InfoRow
            label="Payload URL"
            value={<span className="break-all font-mono">{webhook.url}</span>}
          />
          <InfoRow
            label="Event subscription"
            value={
              <Badge variant="secondary" className="font-normal">
                {webhook.subscribe_all ? "All event types" : "Selected event types"}
              </Badge>
            }
          />
          <InfoRow
            label="Description"
            value={webhook.description || <span className="text-muted-foreground">No description</span>}
          />
        </dl>
      </InformationCard>

      <div className="space-y-4">
        <InformationCard
          title="Delivery Settings"
          icon={Timer}
          description="Retry and timeout controls used for each delivery attempt."
        >
          <dl className="space-y-4">
            <InfoRow
              label="Max retries"
              value={
                <span className="inline-flex items-center gap-2">
                  <RefreshCw className="size-3.5 text-muted-foreground" />
                  {webhook.max_retries} attempt(s)
                </span>
              }
            />
            <InfoRow label="Request timeout" value={`${webhook.timeout_seconds} second(s)`} />
          </dl>
        </InformationCard>

        <InformationCard
          title="Signature"
          icon={ShieldCheck}
          description="Every delivery is signed so receivers can verify its origin."
        >
          <p className="text-sm text-muted-foreground">
            The signing secret is shown once when created. Rotate it from the edit form when the
            receiver needs a fresh verification key.
          </p>
        </InformationCard>
      </div>
    </div>
  )
}
