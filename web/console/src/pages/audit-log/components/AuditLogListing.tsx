import { useNavigate } from "react-router-dom"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { useAuditLogList } from "@/hooks/useAuditLog"
import { auditLogColumns } from "./AuditLogColumns"

const DEFAULT_SORT: SortingState = [{ id: "created_at", desc: false }]
const SEARCH_FIELDS = ["resource_type", "action"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  {
    key: "resource_type",
    label: "Resource",
    options: [
      "user",
      "client",
      "role",
      "policy",
      "tenant",
      "identity_provider",
      "registration_flow",
      "branding",
    ],
  },
  { key: "outcome", label: "Outcome", options: ["success", "failure", "partial"] },
]

export function AuditLogListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const navigate = useNavigate()

  return (
    <ResourceListing
      tableInCard={tableInCard}
      columns={auditLogColumns}
      defaultSort={DEFAULT_SORT}
      searchFields={SEARCH_FIELDS}
      searchPlaceholder="Search audit entries by action or resource type..."
      useData={useAuditLogList}
      filterGroups={FILTER_GROUPS}
      emptyTitle="No audit entries yet"
      emptyDescription="Administrative actions will appear here as changes are made."
      onRowClick={(entry) => navigate(`/audit-log/${entry.uuid}`)}
    />
  )
}
