import { useNavigate } from "react-router-dom"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { apiColumns } from "./ApiColumns"
import { useApisList } from "@/hooks/useApis"

const DEFAULT_SORT: SortingState = [{ id: "created_at", desc: false }]
// Backend list filters the free-text search maps onto (ILIKE per field).
const SEARCH_FIELDS = ["name", "display_name", "identifier"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive"] },
  { key: "is_system", label: "Type", options: ["system", "regular"] },
]

export function ApiListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const navigate = useNavigate()

  return (
    <ResourceListing
      tableInCard={tableInCard}
      columns={apiColumns}
      defaultSort={DEFAULT_SORT}
      searchFields={SEARCH_FIELDS}
      searchPlaceholder="Search APIs by name, display name, or identifier..."
      useData={useApisList}
      filterGroups={FILTER_GROUPS}
      onRowClick={(api) => navigate(`/apis/${api.api_id}`)}
      onCreate={() => navigate(`/apis/create`)}
      createLabel="New API"
      emptyTitle="No APIs yet"
      emptyDescription="Register your first API to start grouping endpoints and managing their permissions."
    />
  )
}
