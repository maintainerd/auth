import { Info, KeyRound } from "lucide-react"
import { InformationCard } from "@/components/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { getProviderConnectionSchema, PROVIDER_LABELS } from "@/components/provider-config"
import type { IdentityProviderDetail } from "@/services/api/identity-providers/types"

interface IdentityProviderConnectionTabProps {
  provider: IdentityProviderDetail
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined || value === "") return ""
  if (Array.isArray(value)) return value.join(", ")
  if (typeof value === "object") return JSON.stringify(value)
  return String(value)
}

function connectionValue(provider: IdentityProviderDetail, key: string): unknown {
  switch (key) {
    case "issuer":
      return provider.issuer
    case "provider_client_id":
      return provider.provider_client_id
    case "allow_jit_provisioning":
      return provider.allow_jit_provisioning ? "Enabled" : "Off"
    case "email_domains":
      return provider.email_domains
    default:
      return ""
  }
}

/**
 * Read-only mirror of the provider-aware section of the form: it renders the
 * top-level provider connection fields. Client secrets are intentionally absent
 * because the backend stores them write-only.
 */
export function IdentityProviderConnectionTab({ provider }: IdentityProviderConnectionTabProps) {
  const connectionSchema = provider.is_system || provider.provider_type === "system"
    ? undefined
    : getProviderConnectionSchema(provider.provider)
  const providerLabel = PROVIDER_LABELS[provider.provider] ?? provider.provider

  return (
    <InformationCard
      title="Connection"
      description={
        connectionSchema
          ? `${providerLabel} broker connection fields stored on the provider record.`
          : "Built-in Maintainerd authentication does not use an upstream provider connection."
      }
      icon={KeyRound}
    >
      <div className="space-y-6">
        {connectionSchema && (
          <div className="space-y-4">
            <div className="space-y-0.5">
              <h4 className="text-sm font-semibold">Broker Connection</h4>
              <p className="text-sm text-muted-foreground">{connectionSchema.summary}</p>
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              {connectionSchema.fields
                .filter((field) => field.key !== "provider_client_secret")
                .map((field) => {
                  const value = formatValue(connectionValue(provider, field.key))

                  return (
                    <div key={field.key} className="space-y-1">
                      <p className="text-sm font-medium text-muted-foreground">{field.label}</p>
                      {value ? (
                        <p className="break-all rounded bg-muted px-2 py-1.5 font-mono text-sm">
                          {value}
                        </p>
                      ) : (
                        <p className="text-sm text-muted-foreground">—</p>
                      )}
                    </div>
                  )
                })}
            </div>
          </div>
        )}

        {!connectionSchema && (
          <Alert>
            <Info className="h-4 w-4" />
            <AlertDescription>
              This system provider is managed by Maintainerd and has no issuer, upstream client ID,
              client secret, or email-domain routing.
            </AlertDescription>
          </Alert>
        )}
      </div>
    </InformationCard>
  )
}
