import { useNavigate } from "react-router-dom"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { roleColumns } from "./RoleColumns"
import { useRoles } from "@/hooks/useRoles"

const DEFAULT_SORT: SortingState = [{ id: "name", desc: false }]
const SEARCH_FIELDS = ["search"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive"] },
]

export function RoleListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const navigate = useNavigate()

  return (
    <ResourceListing
      tableInCard={tableInCard}
      columns={roleColumns}
      defaultSort={DEFAULT_SORT}
      searchFields={SEARCH_FIELDS}
      searchPlaceholder="Search roles by name or description..."
      useData={useRoles}
      filterGroups={FILTER_GROUPS}
      onRowClick={(role) => navigate(`/roles/${role.role_id}`)}
      onCreate={() => navigate(`/roles/create`)}
      createLabel="New Role"
      emptyTitle="No roles yet"
      emptyDescription="Add your first role to define permission groups and control what users can access."
    />
  )
}
