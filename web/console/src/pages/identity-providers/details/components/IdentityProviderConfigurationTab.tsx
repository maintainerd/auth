import { ExternalLink, Settings2, ShieldCheck } from "lucide-react"
import { InformationCard } from "@/components/card"
import { EmptyState } from "@/components/details"
import { Badge } from "@/components/ui/badge"
import {
  getPromotedProviderFieldKeys,
  getProviderConfigSchema,
  getProviderFieldKeys,
} from "@/components/provider-config"
import type { IdentityProviderDetail } from "@/services/api/identity-providers/types"

interface IdentityProviderConfigurationTabProps {
  provider: IdentityProviderDetail
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined || value === "") return ""
  if (typeof value === "object") return JSON.stringify(value)
  return String(value)
}

/**
 * Shows the provider-specific config JSON. Top-level broker connection fields
 * are rendered on the Connection tab, while this tab owns scopes, endpoint
 * overrides, attribute mapping, and extra provider-specific config keys.
 */
export function IdentityProviderConfigurationTab({ provider }: IdentityProviderConfigurationTabProps) {
  const schema = provider.is_system || provider.provider_type === "system"
    ? undefined
    : getProviderConfigSchema(provider.provider)
  const config = provider.config || {}
  const allowedAudiences = provider.allowed_audiences ?? []
  const knownKeys = [
    ...getProviderFieldKeys(provider.provider),
    ...getPromotedProviderFieldKeys(),
  ]
  const entries = Object.entries(config).filter(([key]) => !knownKeys.includes(key))
  const hasConfigGroups = Boolean(schema && schema.groups.length > 0)

  return (
    <div className="space-y-6">
    <InformationCard
      title="Registration & Federation"
      description="Self-service registration and token federation policies for this provider."
      icon={ShieldCheck}
    >
      <div className="space-y-4">
        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-1">
            <p className="text-sm font-medium text-muted-foreground">Allow registration</p>
            <Badge variant={provider.allow_registration ? "default" : "secondary"}>
              {provider.allow_registration ? "Enabled" : "Disabled"}
            </Badge>
            <p className="text-xs text-muted-foreground">
              {provider.allow_registration
                ? "Users can self-register through this provider."
                : "Self-registration is blocked for this provider."}
            </p>
          </div>

          <div className="space-y-1">
            <p className="text-sm font-medium text-muted-foreground">Token federation (Mode B)</p>
            <Badge variant={provider.allow_token_federation ? "default" : "secondary"}>
              {provider.allow_token_federation ? "Enabled" : "Disabled"}
            </Badge>
            <p className="text-xs text-muted-foreground">
              {provider.allow_token_federation
                ? "Accepts foreign OIDC ID tokens from this issuer."
                : "Foreign token acceptance is off."}
            </p>
          </div>
        </div>

        {provider.allow_token_federation && allowedAudiences.length > 0 && (
          <div className="space-y-2">
            <p className="text-sm font-medium text-muted-foreground">Allowed audiences</p>
            <div className="flex flex-wrap gap-1.5">
              {allowedAudiences.map((aud) => (
                <span key={aud} className="rounded bg-muted px-2 py-1 font-mono text-xs">
                  {aud}
                </span>
              ))}
            </div>
          </div>
        )}

        {provider.allow_token_federation && allowedAudiences.length === 0 && (
          <div className="space-y-2">
            <p className="text-sm font-medium text-muted-foreground">Allowed audiences</p>
            <p className="text-xs text-muted-foreground">—</p>
          </div>
        )}
      </div>
    </InformationCard>

    <InformationCard
      title="Configuration"
      description={schema?.summary ?? "Provider-specific configuration stored as JSON."}
      icon={Settings2}
    >
      <div className="space-y-6">
        {schema?.docsUrl && (
          <a
            href={schema.docsUrl}
            target="_blank"
            rel="noreferrer noopener"
            className="inline-flex w-fit items-center gap-1.5 text-sm font-medium text-primary hover:underline"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            {schema.docsLabel ?? "Setup documentation"}
          </a>
        )}

        {hasConfigGroups && schema!.groups.map((group, groupIndex) => (
          <div key={group.title} className="space-y-4">
            {groupIndex > 0 && <div className="border-t" />}
            <div className="space-y-0.5">
              <h4 className="text-sm font-semibold">{group.title}</h4>
              {group.description && (
                <p className="text-sm text-muted-foreground">{group.description}</p>
              )}
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              {group.fields.map((field) => {
                const value = formatValue(config[field.key])
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
        ))}

        {entries.length > 0 && (
          <div className="space-y-4">
            {hasConfigGroups && <div className="border-t" />}
            <div className="space-y-0.5">
              <h4 className="text-sm font-semibold">Additional Configuration</h4>
              <p className="text-sm text-muted-foreground">
                Extra config keys not managed by the provider-specific form fields.
              </p>
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              {entries.map(([key, value]) => (
                <div key={key} className="space-y-1">
                  <p className="font-mono text-sm font-medium text-muted-foreground">{key}</p>
                  <p className="break-all rounded bg-muted px-2 py-1.5 font-mono text-sm">
                    {formatValue(value) || "—"}
                  </p>
                </div>
              ))}
            </div>
          </div>
        )}

        {!hasConfigGroups && entries.length === 0 && (
          <EmptyState
            icon={Settings2}
            title="No configuration"
            description="This provider has no provider-specific config JSON."
          />
        )}
      </div>
    </InformationCard>
    </div>
  )
}
