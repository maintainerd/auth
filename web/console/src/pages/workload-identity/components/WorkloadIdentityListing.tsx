import { useNavigate } from "react-router-dom"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { workloadIdentityColumns } from "./WorkloadIdentityColumns"
import { useWorkloadIdentitiesList } from "@/hooks/useWorkloadIdentity"

// Sort id is the API column (backend allow-list), not the display column id —
// "Created" would be rejected by the backend sanitizer and silently fall back.
const DEFAULT_SORT: SortingState = [{ id: "created_at", desc: true }]

const SEARCH_FIELDS = ["name"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  // Human option values, matching the is_system ("system"/"regular") convention in
  // the other listings — the chips read as words, not booleans.
  { key: "is_active", label: "Status", options: ["active", "inactive"] },
]

export function WorkloadIdentityListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const navigate = useNavigate()

  return (
    <ResourceListing
      tableInCard={tableInCard}
      columns={workloadIdentityColumns}
      defaultSort={DEFAULT_SORT}
      searchFields={SEARCH_FIELDS}
      searchPlaceholder="Search federations by name..."
      useData={useWorkloadIdentitiesList}
      filterGroups={FILTER_GROUPS}
      onRowClick={(federation) =>
        navigate(`/workload-identity/${federation.workload_identity_federation_uuid}`)
      }
      onCreate={() => navigate(`/workload-identity/create`)}
      createLabel="New Federation"
      emptyTitle="No workload identity federations yet"
      emptyDescription="Let a CI job, Kubernetes pod or other workload exchange its own OIDC token for an access token — no stored secret to leak or rotate."
    />
  )
}
