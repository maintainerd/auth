import { useState } from "react"
import { useSearchParams } from "react-router-dom"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { permissionColumns } from "./PermissionColumns"
import { PermissionCreateDialog } from "./PermissionCreateDialog"
import { usePermissionsList } from "@/hooks/usePermissions"

const DEFAULT_SORT: SortingState = [{ id: "created_at", desc: true }]
const SEARCH_FIELDS = ["name", "description"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive"] },
  { key: "is_system", label: "Type", options: ["system", "regular"] },
]

/** The CreateMenu entry lands here with this flag rather than on a /create
 *  route, because creating a permission needs an API chosen in the dialog. */
const CREATE_PARAM = "create"

export function PermissionListing() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [isCreateOpen, setIsCreateOpen] = useState(() => searchParams.get(CREATE_PARAM) === "1")

  const handleCreateOpenChange = (open: boolean) => {
    setIsCreateOpen(open)
    // Drop the flag once consumed, so a refresh or a back-navigation doesn't
    // reopen the dialog on top of the listing.
    if (!open && searchParams.get(CREATE_PARAM)) {
      const next = new URLSearchParams(searchParams)
      next.delete(CREATE_PARAM)
      setSearchParams(next, { replace: true })
    }
  }

  return (
    <>
      <ResourceListing
        columns={permissionColumns}
        defaultSort={DEFAULT_SORT}
        searchFields={SEARCH_FIELDS}
        searchPlaceholder="Search permissions by name or description..."
        useData={usePermissionsList}
        filterGroups={FILTER_GROUPS}
        emptyTitle="No permissions yet"
        emptyDescription="Create permissions to define granular access controls for your APIs."
        onCreate={() => handleCreateOpenChange(true)}
        createLabel="New Permission"
      />
      {/* No onRowClick: there is no /permissions/:id route, so every row was a
          one-way trip to the 404 page. */}
      <PermissionCreateDialog open={isCreateOpen} onOpenChange={handleCreateOpenChange} />
    </>
  )
}
