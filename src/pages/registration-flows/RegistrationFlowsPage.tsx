import { Link } from "react-router-dom"
import { RegistrationFlowListing } from "./components/RegistrationFlowListing"
import { PageHeader } from "@/components/layout"

/**
 * Titled "Registration Flows", not "Registration": the tenant-wide registration
 * policy is a separate feature (Security → Registration). A flow's
 * `verification_required` switch overrides that tenant-wide
 * `require_email_verification`, so the relationship is cross-linked here rather
 * than left for the operator to infer from two similarly named screens.
 */
export default function RegistrationFlowsPage() {
  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Registration Flows"
        description="Define how users onboard into your applications, with automatic role assignment per flow. Each flow has its own registration link."
      />
      <p className="text-sm text-muted-foreground">
        These are per-application flows. The tenant-wide registration defaults they override live in{" "}
        <Link to="/security?tab=registration" className="font-medium underline underline-offset-4">
          Security → Registration
        </Link>
        .
      </p>
      <RegistrationFlowListing tableInCard />
    </div>
  )
}
