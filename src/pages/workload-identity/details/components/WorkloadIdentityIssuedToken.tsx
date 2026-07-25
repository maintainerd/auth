import { KeyRound } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { InformationCard } from "@/components/card"
import { InfoRow } from "./InfoRow"
import type { WorkloadIdentityFederation } from "@/services/api/workload-identity/types"

interface WorkloadIdentityIssuedTokenProps {
  federation: WorkloadIdentityFederation
}

export function WorkloadIdentityIssuedToken({ federation }: WorkloadIdentityIssuedTokenProps) {
  const mappingEntries = Object.entries(federation.attribute_mapping ?? {})
  const scopes = federation.allowed_scopes ?? []

  return (
    <InformationCard
      title="Issued token"
      icon={KeyRound}
      description="What a matching workload receives once its token is accepted."
    >
      <dl className="space-y-4">
        <InfoRow
          label="Allowed scopes"
          value={
            scopes.length === 0 ? (
              <span className="text-muted-foreground">
                None — the issued token carries no scopes.
              </span>
            ) : (
              <div className="flex flex-wrap gap-1">
                {scopes.map((scope) => (
                  <Badge key={scope} variant="secondary" className="text-xs">
                    {scope}
                  </Badge>
                ))}
              </div>
            )
          }
        />
        <InfoRow
          label="Attribute mapping"
          value={
            mappingEntries.length === 0 ? (
              <span className="text-muted-foreground">
                No claims are copied from the workload&apos;s token.
              </span>
            ) : (
              <div className="space-y-1">
                {mappingEntries.map(([externalClaim, tokenClaim]) => (
                  <div key={externalClaim} className="font-mono text-xs">
                    {externalClaim} <span aria-hidden="true">&rarr;</span> {tokenClaim}
                  </div>
                ))}
              </div>
            )
          }
        />
      </dl>
    </InformationCard>
  )
}
