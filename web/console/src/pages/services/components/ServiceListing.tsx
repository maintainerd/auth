import { useNavigate } from "react-router-dom"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { serviceColumns } from "./ServiceColumns"
import { useServicesList } from "@/hooks/useServices"

const DEFAULT_SORT: SortingState = [{ id: "created_at", desc: false }]
// Backend list filters the free-text search maps onto (ILIKE per field).
const SEARCH_FIELDS = ["name", "display_name", "description", "version"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "maintenance", "deprecated", "inactive"] },
  { key: "is_system", label: "Type", options: ["system", "regular"] },
]

export function ServiceListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const navigate = useNavigate()

  return (
    <ResourceListing
      tableInCard={tableInCard}
      columns={serviceColumns}
      defaultSort={DEFAULT_SORT}
      searchFields={SEARCH_FIELDS}
      searchPlaceholder="Search services by name, display name, description, or version..."
      useData={useServicesList}
      filterGroups={FILTER_GROUPS}
      onRowClick={(service) => navigate(`/services/${service.service_id}`)}
      onCreate={() => navigate(`/services/create`)}
      createLabel="New Service"
      emptyTitle="No services yet"
      emptyDescription="Register your first service to start managing its APIs, permissions, and policies."
    />
  )
}
