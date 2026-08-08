import { Braces } from "lucide-react"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton } from "@/components/details"
import { useClientConfig } from "@/hooks/useClients"
import { formatClientConfigValue, getClientMetadataEntries } from "../../clientConfig"

interface ClientMetadataProps {
  clientId: string
}

export function ClientMetadata({ clientId }: ClientMetadataProps) {
  const { data: configData, isLoading, isError } = useClientConfig(clientId)

  const metadataEntries = getClientMetadataEntries(configData ?? {})

  return (
    <InformationCard
      title="Client Metadata"
      description="Provider-specific and application-specific fields beyond common client configuration."
      icon={Braces}
    >
      {isLoading && <ListSkeleton />}

      {isError && (
        <p className="py-8 text-center text-sm text-destructive">Failed to load metadata</p>
      )}

      {configData && metadataEntries.length > 0 && (
        <div className="grid gap-4 md:grid-cols-2">
          {metadataEntries.map(([key, value]) => (
            <div key={key} className="space-y-1">
              <p className="font-mono text-sm font-medium text-muted-foreground">{key}</p>
              <p className="break-all rounded bg-muted px-2 py-1.5 font-mono text-sm">
                {formatClientConfigValue(value) || "—"}
              </p>
            </div>
          ))}
        </div>
      )}

      {configData && metadataEntries.length === 0 && (
        <EmptyState
          icon={Braces}
          title="No metadata"
          description="This client has no non-common provider or application metadata."
        />
      )}
    </InformationCard>
  )
}
