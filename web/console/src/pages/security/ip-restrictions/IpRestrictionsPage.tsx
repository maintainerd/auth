import { PageHeader } from "@/components/layout"
import { IpRestrictionListing } from "./components/IpRestrictionListing"

export default function IpRestrictionsPage({ standalone = true }: { standalone?: boolean }) {
  if (!standalone) return <IpRestrictionListing tableInCard />

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="IP Restrictions"
        description="Manage IP allow/deny rules to control access to your authentication system."
      />
      <IpRestrictionListing tableInCard />
    </div>
  )
}
