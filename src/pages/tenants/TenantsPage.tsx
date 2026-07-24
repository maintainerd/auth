import { TenantListing } from "./components/TenantListing"
import { PageHeader } from "@/components/layout"

export default function TenantsPage() {
  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Tenants"
        description="Manage your tenant organizations, their status, and authentication settings."
      />
      <TenantListing tableInCard />
    </div>
  )
}
