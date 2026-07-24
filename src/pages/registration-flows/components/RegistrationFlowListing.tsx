import { useNavigate } from "react-router-dom"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { registrationFlowColumns } from "./RegistrationFlowColumns"
import { useRegistrationFlowsList } from "@/hooks/useRegistrationFlows"

// `created_at` is the backend column name. A display label like "Created" is not
// in the backend's sort allow-list (SanitizeOrder), so it was silently discarded
// and the server fell back to its own default ordering.
const DEFAULT_SORT: SortingState = [{ id: "created_at", desc: true }]
// The backend exposes one free-text `search` filter spanning name + identifier,
// so a single field covers both — matching identity-providers and roles.
const SEARCH_FIELDS = ["search"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive"] },
  { key: "is_system", label: "Type", options: ["system", "regular"] },
]

export function RegistrationFlowListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const navigate = useNavigate()

  return (
    <ResourceListing
      tableInCard={tableInCard}
      columns={registrationFlowColumns}
      defaultSort={DEFAULT_SORT}
      searchFields={SEARCH_FIELDS}
      searchPlaceholder="Search registration flows by name or description..."
      useData={useRegistrationFlowsList}
      filterGroups={FILTER_GROUPS}
      onRowClick={(flow) => navigate(`/registration-flows/${flow.registration_flow_id}`)}
      onCreate={() => navigate(`/registration-flows/create`)}
      createLabel="New Registration Flow"
      emptyTitle="No registration flows yet"
      emptyDescription="Create your first registration flow to define how users onboard into your applications and which roles they receive."
    />
  )
}
