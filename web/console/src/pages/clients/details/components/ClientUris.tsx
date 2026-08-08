import { safeFormat } from "@/lib/formatDate"
import { Link2 } from "lucide-react"
import { InformationCard } from "@/components/card"
import { EmptyState } from "@/components/details"
import { Badge } from "@/components/ui/badge"
import type { ClientResponse, ClientUriType } from "@/services/api/clients/types"
import { getClientTypeCapability } from "../../form/clientTypeConfig"

interface ClientUrisProps {
  client: ClientResponse
}

const URI_LABELS: Record<ClientUriType, string> = {
  "login_uri": "Login URI",
  "redirect_uri": "Redirect URIs",
  "origin_uri": "Allowed Origins",
  "logout_uri": "Allowed Logout URLs",
  "cors_origin_uri": "CORS Origins",
}

const URI_DESCRIPTIONS: Record<ClientUriType, string> = {
  "login_uri": "Where users are directed to begin login.",
  "redirect_uri": "Where users return after authentication.",
  "origin_uri": "Browser origins allowed to call this client.",
  "logout_uri": "Where users can return after logout.",
  "cors_origin_uri": "Origins allowed for cross-origin authentication.",
}

export function ClientUris({ client }: ClientUrisProps) {
  const uris = client.uris ?? []
  const capability = getClientTypeCapability(client.client_type)
  const orderedTypes: ClientUriType[] = [
    ...(capability.showLoginUri ? ["login_uri" as ClientUriType] : []),
    ...(capability.showRedirectUris ? ["redirect_uri" as ClientUriType] : []),
    ...(capability.showAllowedOrigins ? ["origin_uri" as ClientUriType] : []),
    ...(capability.showLogoutUrls ? ["logout_uri" as ClientUriType] : []),
    ...(capability.showCors ? ["cors_origin_uri" as ClientUriType] : []),
  ]
  const remainingTypes = [...new Set(uris.map((uri) => uri.type))]
    .filter((type) => !orderedTypes.includes(type))
  const visibleTypes = [...orderedTypes, ...remainingTypes]
    .filter((type) => uris.some((uri) => uri.type === type))

  return (
    <InformationCard
      title="URIs & Origins"
      description="Application redirects, browser origins, login/logout targets, and CORS origins returned by the backend."
      icon={Link2}
    >
      {uris.length > 0 ? (
        <div className="space-y-5">
          {visibleTypes.map((type) => (
            <section key={type} className="space-y-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <h4 className="text-sm font-semibold">{URI_LABELS[type] ?? type}</h4>
                  <p className="text-sm text-muted-foreground">{URI_DESCRIPTIONS[type] ?? "Configured client URI."}</p>
                </div>
                <Badge variant="outline" className="text-xs">
                  {uris.filter((uri) => uri.type === type).length}
                </Badge>
              </div>

              <div className="space-y-2">
                {uris.filter((uri) => uri.type === type).map((uri) => (
                  <div data-md-listing-nested key={uri.uri_id} className="rounded-md border bg-background p-3">
                    <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                      <code className="break-all rounded bg-muted px-2 py-1.5 font-mono text-sm">
                        {uri.uri}
                      </code>
                      <Badge variant="secondary" className="w-fit shrink-0 text-xs">
                        {uri.type}
                      </Badge>
                    </div>
                    <div className="mt-3 grid gap-3 border-t pt-3 text-xs text-muted-foreground md:grid-cols-3">
                      <div>
                        <span className="font-medium">URI UUID</span>
                        <p className="break-all font-mono">{uri.uri_id}</p>
                      </div>
                      <div>
                        <span className="font-medium">Created</span>
                        <p>{safeFormat(uri.created_at, "PPpp")}</p>
                      </div>
                      <div>
                        <span className="font-medium">Last Updated</span>
                        <p>{safeFormat(uri.updated_at, "PPpp")}</p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </section>
          ))}
        </div>
      ) : (
        <EmptyState
          icon={Link2}
          title="No URIs or origins"
          description="This client has no application URI records configured."
        />
      )}
    </InformationCard>
  )
}
