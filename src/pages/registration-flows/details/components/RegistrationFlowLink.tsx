import { ExternalLink, Link2, ShieldAlert, TriangleAlert } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { InformationCard } from "@/components/card"
import { CopyableCode } from "@/components/inputs"
import { useAppSelector } from "@/store/hooks"
import { API_CONFIG } from "@/services/api/config"
import { buildRegistrationUrl } from "../utils"
import type { RegistrationFlowDetail } from "@/services/api/registration-flows/types"

interface RegistrationFlowLinkProps {
  flow: RegistrationFlowDetail
}

/**
 * The registration link is the point of a flow: an external app redirects users
 * here and whoever completes registration receives the flow's roles. It is
 * surfaced with copy + open actions, plus the two facts an operator needs —
 * renaming the flow changes this link, and deactivating the flow is the kill
 * switch.
 */
export function RegistrationFlowLink({ flow }: RegistrationFlowLinkProps) {
  // The real identity host is per-tenant and comes from the tenant bootstrap;
  // the env-derived origin is only the last-resort fallback.
  const tenantIdentityUrl = useAppSelector((state) => state.tenant.identityUrl)
  const identityUrl = tenantIdentityUrl || API_CONFIG.IDENTITY_BASE_URL

  const clientIdentifier = flow.client?.identifier
  // System flows are invite-only: the backend refuses a self-service link for
  // them unconditionally. Composing one anyway would hand the operator a URL
  // that always fails, so there is deliberately no link to copy or open.
  const registrationUrl =
    clientIdentifier && !flow.is_system
      ? buildRegistrationUrl(identityUrl, clientIdentifier, flow.name)
      : null


  return (
    <InformationCard
      title="Registration Link"
      description="Send users here to register through this flow. Everyone who completes it receives the roles assigned below, plus the default registered role."
      icon={Link2}
      action={
        registrationUrl ? (
          <Button variant="outline" size="sm" className="h-9 gap-2" asChild>
            <a href={registrationUrl} target="_blank" rel="noopener noreferrer">
              <ExternalLink className="size-4" />
              Open
            </a>
          </Button>
        ) : undefined
      }
    >
      <div className="space-y-5">
        {registrationUrl ? (
          <div className="space-y-1">
            <p className="text-sm font-medium text-muted-foreground">Link</p>
            {/* Copy sits with the value rather than in the card header, so it is
                unambiguous which text it puts on the clipboard. */}
            <CopyableCode value={registrationUrl} label="Registration link" variant="block" />
          </div>
        ) : flow.is_system ? (
          <Alert>
            <ShieldAlert className="size-4" />
            <AlertDescription>
              This is a system flow, so it has no registration link. System flows grant privileged
              roles and are redeemable only through an invite — a self-service link would be refused.
            </AlertDescription>
          </Alert>
        ) : (
          <Alert>
            <TriangleAlert className="size-4" />
            <AlertDescription>
              This flow has no client attached, so no registration link can be composed. Assign a
              client to the flow to publish a link.
            </AlertDescription>
          </Alert>
        )}

        {registrationUrl && (
          <Alert>
            <ShieldAlert className="size-4" />
            <AlertDescription>
              This link is built from the flow name, so <strong>renaming the flow breaks it</strong> —
              anyone who has already published it must be told to update it.{" "}
              {flow.status === "active"
                ? "To revoke the link, deactivate this flow: registration is refused while it is inactive."
                : "This flow is inactive, so the link currently refuses all registrations."}
            </AlertDescription>
          </Alert>
        )}
      </div>
    </InformationCard>
  )
}
