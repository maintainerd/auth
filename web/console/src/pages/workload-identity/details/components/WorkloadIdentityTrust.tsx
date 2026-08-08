import { ShieldCheck } from "lucide-react"
import { InformationCard } from "@/components/card"
import { InfoRow } from "./InfoRow"
import type { WorkloadIdentityFederation } from "@/services/api/workload-identity/types"

interface WorkloadIdentityTrustProps {
  federation: WorkloadIdentityFederation
}

export function WorkloadIdentityTrust({ federation }: WorkloadIdentityTrustProps) {
  return (
    <InformationCard
      title="Trust"
      icon={ShieldCheck}
      description="Which external tokens this federation accepts. These fields together are the entire trust boundary."
    >
      <dl className="space-y-4">
        <InfoRow
          label="Issuer"
          value={<span className="break-all font-mono">{federation.issuer_url}</span>}
        />
        <InfoRow
          label="Audience"
          value={<span className="break-all font-mono">{federation.audience}</span>}
        />
        <InfoRow label="Subject claim" value={<code>{federation.subject_claim}</code>} />
        <InfoRow
          label="Subject pattern"
          value={<span className="break-all font-mono">{federation.subject_pattern}</span>}
        />
      </dl>
    </InformationCard>
  )
}
