import { useNavigate } from "react-router-dom"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { policyColumns } from "./PolicyColumns"
import { usePoliciesList } from "@/hooks/usePolicies"

const DEFAULT_SORT: SortingState = [{ id: "created_at", desc: false }]
// Backend list filters the free-text search maps onto (ILIKE per field).
const SEARCH_FIELDS = ["name", "description", "version"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive"] },
  { key: "is_system", label: "Type", options: ["system", "regular"] },
]

export function PolicyListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const navigate = useNavigate()

  return (
    <ResourceListing
      tableInCard={tableInCard}
      columns={policyColumns}
      defaultSort={DEFAULT_SORT}
      searchFields={SEARCH_FIELDS}
      searchPlaceholder="Search policies by name, description, or version..."
      useData={usePoliciesList}
      filterGroups={FILTER_GROUPS}
      onRowClick={(policy) => navigate(`/policies/${policy.policy_id}`)}
      onCreate={() => navigate(`/policies/create`)}
      createLabel="New Policy"
      emptyTitle="No policies yet"
      emptyDescription="Create your first policy to control service access with allow and deny statements."
    />
  )
}
