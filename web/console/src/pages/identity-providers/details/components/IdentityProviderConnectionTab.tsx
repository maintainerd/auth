import { Info, KeyRound } from "lucide-react"
import { InformationCard } from "@/components/card"
import { CopyableCode } from "@/components/inputs"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { getProviderConnectionSchema, PROVIDER_LABELS } from "@/components/provider-config"
import type { IdentityProviderDetail } from "@/services/api/identity-providers/types"
import { SamlServiceProviderCard } from "./SamlServiceProviderCard"

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

/** True when this provider federates over SAML 2.0 rather than OIDC/OAuth2. */
function isSamlProvider(provider: IdentityProviderDetail): boolean {
  if (provider.is_system) return false
  return provider.provider === "saml" || provider.provider_type === "saml"
}

/**
 * Read-only mirror of the provider-aware section of the form: it renders the
 * top-level provider connection fields. Client secrets are intentionally absent
 * because the backend stores them write-only.
 *
 * SAML providers have no OIDC broker connection at all, so instead of the
 * issuer/client-id grid they get the service-provider card — the values the
 * upstream IdP needs from Maintainerd to complete the connection.
 */
export function IdentityProviderConnectionTab({ provider }: IdentityProviderConnectionTabProps) {
  const isSaml = isSamlProvider(provider)
  const connectionSchema = provider.is_system || provider.provider_type === "system"
    ? undefined
    : getProviderConnectionSchema(provider.provider)
  const providerLabel = PROVIDER_LABELS[provider.provider] ?? provider.provider

  return (
    <div className="space-y-6">
      <InformationCard
        title="Connection"
        description={
          connectionSchema
            ? `${providerLabel} broker connection fields stored on the provider record.`
            : isSaml
              ? `${providerLabel} federates over signed assertions rather than an OIDC broker connection.`
              : "Built-in Maintainerd authentication does not use an upstream provider connection."
        }
        icon={KeyRound}
      >
        <div className="space-y-6">
          {connectionSchema && provider.callback_url && (
            <div className="space-y-2">
              <div className="space-y-0.5">
                <h4 className="text-sm font-semibold">Redirect / Callback URL</h4>
                <p className="text-sm text-muted-foreground">
                  Register this exact URL as an allowed callback (redirect) URL in {providerLabel}.
                  After a user authenticates there, the brokered sign-in returns them here.
                </p>
              </div>
              <CopyableCode value={provider.callback_url} label="callback URL" variant="block" />
            </div>
          )}

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

          {/* A SAML provider also has no connection schema, but calling it a
              system provider managed by Maintainerd was simply false — it is a
              tenant-configured enterprise connection whose inbound half lives
              on the Configuration tab. */}
          {!connectionSchema && isSaml && (
            <Alert>
              <Info className="h-4 w-4" />
              <AlertDescription>
                The IdP side of this connection — SSO URL, IdP entity ID and signing certificate —
                is on the Configuration tab. The values your IdP needs from Maintainerd are below.
              </AlertDescription>
            </Alert>
          )}

          {!connectionSchema && !isSaml && (
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

      {isSaml && <SamlServiceProviderCard identifier={provider.identifier} />}
    </div>
  )
}
