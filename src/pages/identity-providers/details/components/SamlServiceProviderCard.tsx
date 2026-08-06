import { Copy, Download, ExternalLink, Network, TriangleAlert } from "lucide-react"
import { InformationCard } from "@/components/card"
import { CopyableCode } from "@/components/inputs"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { useToast } from "@/hooks/useToast"
import { useSamlServiceProviderMetadata } from "@/hooks/useIdentityProviders"

interface SamlServiceProviderCardProps {
  /** The provider's `identifier` — the value the metadata route is keyed on. */
  identifier: string
}

const METADATA_FILENAME_PREFIX = "maintainerd-sp-metadata"

function Field({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="space-y-1">
      <p className="text-sm font-medium text-muted-foreground">{label}</p>
      <CopyableCode value={value} label={label} variant="block" />
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  )
}

/**
 * The half of a SAML integration that Maintainerd owns.
 *
 * A SAML connection needs values flowing in BOTH directions: the admin enters
 * the IdP's SSO URL, entity ID and certificate on the form, and the IdP needs
 * Maintainerd's entity ID and ACS URL in return. Only the inbound half was ever
 * shown, so the connection could not be completed from the console at all — the
 * admin had to know the backend's route layout and public hostname by heart.
 *
 * Every value here is read out of the metadata document the backend publishes,
 * never composed in the browser: the console does not know the authorization
 * server's public hostname, and a URL that differs by one host or path segment
 * is an IdP that posts assertions nowhere.
 */
export function SamlServiceProviderCard({ identifier }: SamlServiceProviderCardProps) {
  const { showSuccess, showError } = useToast()
  const { data, isLoading, isError, error, refetch, isFetching } = useSamlServiceProviderMetadata(
    identifier,
    true,
  )

  const copyXml = async () => {
    if (!data) return
    try {
      await navigator.clipboard.writeText(data.xml)
      showSuccess("Service provider metadata copied to clipboard")
    } catch (clipboardError) {
      // Clipboard access can be denied (insecure context, permissions policy).
      // Surface it rather than silently appearing to succeed.
      showError(clipboardError)
    }
  }

  // Most IdPs (Okta, Entra ID, Google Workspace) import an SP by FILE, so the
  // XML has to be obtainable even when the metadata URL is unreachable from the
  // admin's network.
  const canDownload = typeof URL !== "undefined" && typeof URL.createObjectURL === "function"

  const downloadXml = () => {
    if (!data || !canDownload) return
    const blob = new Blob([data.xml], { type: "application/samlmetadata+xml" })
    const href = URL.createObjectURL(blob)
    const anchor = document.createElement("a")
    anchor.href = href
    anchor.download = `${METADATA_FILENAME_PREFIX}-${identifier}.xml`
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
    // The object URL pins the blob in memory for the life of the document.
    URL.revokeObjectURL(href)
  }

  return (
    <InformationCard
      title="Service Provider Details"
      description="Maintainerd is the service provider in this connection. Give these values to your SAML identity provider so it can send assertions back."
      icon={Network}
      action={
        data ? (
          <div className="flex gap-2">
            <Button type="button" variant="outline" size="sm" className="h-9 gap-2" onClick={copyXml}>
              <Copy className="size-4" />
              Copy XML
            </Button>
            {canDownload && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-9 gap-2"
                onClick={downloadXml}
              >
                <Download className="size-4" />
                Download
              </Button>
            )}
          </div>
        ) : undefined
      }
    >
      <div className="space-y-6">
        {isLoading && (
          <div className="grid gap-4 md:grid-cols-2">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        )}

        {isError && (
          <Alert variant="destructive">
            <TriangleAlert className="size-4" />
            <AlertDescription>
              <div className="space-y-3">
                {/* Deliberately no fallback URLs here. Printing a guessed entity
                    ID or ACS URL would be pasted straight into a production IdP
                    and fail only at the end of a user's login. */}
                <p>
                  Maintainerd could not publish its service provider details for this connection
                  {error instanceof Error && error.message ? `: ${error.message}` : "."} Save the
                  SSO URL, IdP entity ID and certificate first — the metadata document is only
                  served once the SAML configuration is valid.
                </p>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => void refetch()}
                  disabled={isFetching}
                >
                  {isFetching ? "Retrying…" : "Try again"}
                </Button>
              </div>
            </AlertDescription>
          </Alert>
        )}

        {data && (
          <>
            <div className="grid gap-4 md:grid-cols-2">
              <Field
                label="SP Entity ID"
                value={data.entityId}
                hint="Also called Audience URI or Identifier."
              />
              <Field
                label="ACS URL"
                value={data.acsUrl}
                hint="Also called Reply URL, Single Sign-On URL or Assertion Consumer Service. HTTP-POST binding."
              />
              {data.metadataUrl ? (
                <Field
                  label="SP Metadata URL"
                  value={data.metadataUrl}
                  hint="Import this URL if your IdP fetches service provider metadata."
                />
              ) : (
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">SP Metadata URL</p>
                  <p className="text-sm text-muted-foreground">
                    Not published — use Copy XML or Download instead.
                  </p>
                </div>
              )}
              {data.sloUrl && (
                <Field
                  label="Single Logout URL"
                  value={data.sloUrl}
                  hint="Where the IdP sends logout requests and responses."
                />
              )}
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-1.5">
                <p className="text-sm font-medium text-muted-foreground">NameID format</p>
                {data.nameIdFormats.length > 0 ? (
                  <div className="flex flex-wrap gap-1.5">
                    {data.nameIdFormats.map((format) => (
                      <span key={format} className="rounded bg-muted px-2 py-1 font-mono text-xs">
                        {format}
                      </span>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">—</p>
                )}
              </div>

              <div className="space-y-1.5">
                <p className="text-sm font-medium text-muted-foreground">Signature requirements</p>
                <div className="flex flex-wrap gap-1.5">
                  <Badge variant={data.wantAssertionsSigned ? "default" : "secondary"}>
                    {data.wantAssertionsSigned
                      ? "Signed assertions required"
                      : "Assertion signature not required"}
                  </Badge>
                  <Badge variant={data.authnRequestsSigned ? "default" : "secondary"}>
                    {data.authnRequestsSigned
                      ? "AuthnRequests are signed"
                      : "AuthnRequests are unsigned"}
                  </Badge>
                </div>
              </div>
            </div>

            {data.metadataUrl && (
              <a
                href={data.metadataUrl}
                target="_blank"
                rel="noreferrer noopener"
                className="inline-flex w-fit items-center gap-1.5 text-sm font-medium text-primary hover:underline"
              >
                <ExternalLink className="size-3.5" />
                Open service provider metadata
              </a>
            )}
          </>
        )}
      </div>
    </InformationCard>
  )
}
