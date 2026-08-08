import { InformationCard } from "@/components/card"
import { ShieldCheck } from "lucide-react"
import type { ClientResponse } from "@/services/api/clients/types"

interface ClientInformationProps {
  client: ClientResponse
}

const INHERIT_LABEL = "Inherits tenant default"

const ACR_LABELS: Record<string, string> = {
  "1": "Password / single factor (ACR 1)",
  "2": "Step-up — MFA (ACR 2)",
}

function formatAcr(acr?: string | null): string {
  if (!acr) return INHERIT_LABEL
  return ACR_LABELS[acr] ?? acr
}

function formatPkce(requirePkce?: boolean | null): string {
  if (requirePkce == null) return INHERIT_LABEL
  return requirePkce ? "Required" : "Disabled"
}

// Render a seconds value as "1800s · 30 min" with the coarsest exact unit.
function formatSeconds(value?: number | null): string {
  if (value == null) return INHERIT_LABEL
  if (value % 86400 === 0) return `${value}s · ${value / 86400} day${value / 86400 === 1 ? "" : "s"}`
  if (value % 3600 === 0) return `${value}s · ${value / 3600} hour${value / 3600 === 1 ? "" : "s"}`
  if (value % 60 === 0) return `${value}s · ${value / 60} min`
  return `${value}s`
}

export function ClientInformation({ client }: ClientInformationProps) {
  return (
    <div className="space-y-4">
      <InformationCard
        title="Security & Sessions"
        description="Authentication assurance and session-lifetime overrides. Unset values inherit the tenant security policy."
        icon={ShieldCheck}
      >
        <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
          <DetailField label="Require PKCE" value={formatPkce(client.require_pkce)} />
          <DetailField label="Required Auth Level (ACR)" value={formatAcr(client.required_acr)} />
          <DetailField label="Session Idle Timeout" value={formatSeconds(client.session_idle_timeout)} />
          <DetailField label="Session Absolute Timeout" value={formatSeconds(client.session_absolute_timeout)} />
        </div>
      </InformationCard>
    </div>
  )
}

function DetailField({
  label,
  value,
  mono,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="min-w-0 space-y-1">
      <p className="text-sm font-medium text-muted-foreground">{label}</p>
      <p className={mono ? "break-all font-mono text-sm" : "break-words text-sm"}>{value}</p>
    </div>
  )
}
