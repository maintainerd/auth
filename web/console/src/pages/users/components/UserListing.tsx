import { useNavigate } from "react-router-dom"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { userColumns } from "./UserColumns"
import { useUsers } from "@/hooks/useUsers"

const DEFAULT_SORT: SortingState = [{ id: "created_at", desc: false }]
const SEARCH_FIELDS = ["search"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive", "pending", "suspended"] },
]

export function UserListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const navigate = useNavigate()

  return (
    <ResourceListing
      tableInCard={tableInCard}
      columns={userColumns}
      defaultSort={DEFAULT_SORT}
      searchFields={SEARCH_FIELDS}
      searchPlaceholder="Search users by name, username, email, or phone..."
      useData={useUsers}
      filterGroups={FILTER_GROUPS}
      onRowClick={(user) => navigate(`/users/${user.user_id}`)}
      onCreate={() => navigate(`/users/create`)}
      createLabel="New User"
      emptyTitle="No users yet"
      emptyDescription="Add your first user to start managing accounts, authentication, and security settings."
    />
  )
}
