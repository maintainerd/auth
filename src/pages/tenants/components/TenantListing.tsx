import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { tenantColumns } from "./TenantColumns"
import { useTenants } from "@/hooks/useTenants"
import { useNavigate } from "react-router-dom"

const DEFAULT_SORT: SortingState = [{ id: "created_at", desc: false }]
const SEARCH_FIELDS = ["name", "display_name"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive", "suspended"] },
]

export function TenantListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const navigate = useNavigate()

  return (
    <ResourceListing
      tableInCard={tableInCard}
      columns={tenantColumns}
      defaultSort={DEFAULT_SORT}
      searchFields={SEARCH_FIELDS}
      searchPlaceholder="Search tenants by name or display name..."
      useData={useTenants}
      filterGroups={FILTER_GROUPS}
      onRowClick={(tenant) => navigate(`/tenants/${tenant.tenant_id}`)}
      onCreate={() => navigate(`/tenants/create`)}
      createLabel="New Tenant"
      emptyTitle="No tenants yet"
      emptyDescription="Add your first tenant to start managing organizations, users, and authentication."
    />
  )
}
