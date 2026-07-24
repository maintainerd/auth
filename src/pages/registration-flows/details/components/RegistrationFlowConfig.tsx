import { Link } from "react-router-dom"
import { Settings } from "lucide-react"
import { InformationCard } from "@/components/card"
import type { RegistrationFlowDetail } from "@/services/api/registration-flows/types"

interface RegistrationFlowConfigProps {
  flow: RegistrationFlowDetail
}

const FIELD_LABELS: Record<string, string> = {
  fullname: "Full name",
  email: "Email",
  phone: "Phone",
}

/**
 * Takes the already-loaded flow as a prop instead of re-fetching it: the details
 * page's DetailLayout only renders its children once the flow has resolved
 * successfully, so re-implementing loading/error states here would duplicate a
 * guarantee that already holds (and issue a second request for the same record).
 */
export function RegistrationFlowConfig({ flow }: RegistrationFlowConfigProps) {
  const requiredFields = flow.required_fields ?? []

  return (
    <InformationCard
      title="Flow behaviour"
      description="Applies to this flow only, and may tighten the tenant-wide policy but never loosen it"
      icon={Settings}
    >
      <div className="space-y-4">
        <div className="space-y-1">
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            Email verification
          </p>
          <p className="text-sm">{flow.verification_required ? "Required" : "Not required"}</p>
          <p className="text-xs text-muted-foreground">
            This overrides the tenant-wide policy in{" "}
            <Link to="/security?tab=registration" className="underline underline-offset-4">
              Security → Registration
            </Link>
            .
          </p>
        </div>
        <div className="space-y-1">
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            Required fields
          </p>
          <p className="text-sm">
            {requiredFields.length > 0
              ? requiredFields.map((field) => FIELD_LABELS[field] ?? field).join(", ")
              : "Username and password only"}
          </p>
        </div>
      </div>
    </InformationCard>
  )
}
