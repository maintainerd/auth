import { WorkloadIdentityListing } from "./components/WorkloadIdentityListing"
import { PageHeader } from "@/components/layout"

export default function WorkloadIdentityPage({ standalone = true }: { standalone?: boolean }) {
  if (!standalone) return <WorkloadIdentityListing tableInCard />

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Workload Identity"
        description="Let CI jobs, Kubernetes pods and other workloads exchange their own OIDC token for an access token — no stored secret to leak or rotate."
      />
      <WorkloadIdentityListing tableInCard />
    </div>
  )
}
