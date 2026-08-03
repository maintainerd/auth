import { useState, useMemo } from "react"
import { AlertCircle, Users, Search, Plus } from "lucide-react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { InformationCard } from "@/components/card"
import { DataTablePagination } from "@/components/data-table"
import { DeleteConfirmationDialog } from "@/components/dialog"
import { EmptyState, ListSkeleton } from "@/components/details"
import { MemberItem } from "./MemberItem"
import { AddMemberDialog } from "./AddMemberDialog"
import { TransferOwnershipDialog } from "./TransferOwnershipDialog"
import { useTenantMembers, useDeleteTenantMember } from "@/hooks/useTenantMembers"
import { useToast } from "@/hooks/useToast"
import { useAppSelector } from '@/store/hooks'
import type { TenantMember } from "@/services/api/tenants/members"
import {
  getCoreRowModel,
  useReactTable,
  type PaginationState,
} from "@tanstack/react-table"

interface MembersSettingsProps {
  tenantId?: string
  isSystemTenant?: boolean
}

export function MembersSettings({ tenantId: propTenantId, isSystemTenant }: MembersSettingsProps = {}) {
  const currentTenant = useAppSelector((state) => state.tenant.currentTenant)
  const tenantId = propTenantId || currentTenant?.tenant_id || ''
  
  const [searchQuery, setSearchQuery] = useState("")
  const [addMemberDialogOpen, setAddMemberDialogOpen] = useState(false)
  const [transferOwnershipDialog, setTransferOwnershipDialog] = useState<{
    open: boolean
    owner: TenantMember | null
  }>({
    open: false,
    owner: null
  })
  const [deleteMemberDialog, setDeleteMemberDialog] = useState<{
    open: boolean
    memberId: string | null
    memberName: string | null
    username: string | null
  }>({
    open: false,
    memberId: null,
    memberName: null,
    username: null
  })
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data, isLoading, error } = useTenantMembers(tenantId, {
    limit: pagination.pageSize,
    page: pagination.pageIndex + 1,
    sort_by: 'created_at',
    sort_order: 'desc'
  })
  const deleteMemberMutation = useDeleteTenantMember(tenantId)
  const { showSuccess, showError } = useToast()

  // Create a simple table instance for pagination
  const columns = useMemo(() => [], [])
  const tableData = data?.data?.rows || []

  const table = useReactTable({
    data: tableData,
    columns,
    pageCount: Math.ceil((tableData.length) / pagination.pageSize),
    state: {
      pagination,
    },
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
  })

  // Filter data based on search query (client-side filtering)
  const filteredData = useMemo(() => {
    const rows = data?.data?.rows
    if (!searchQuery || !rows) return rows || []
    return rows.filter((member: TenantMember) =>
      member.user.fullname.toLowerCase().includes(searchQuery.toLowerCase()) ||
      member.user.email.toLowerCase().includes(searchQuery.toLowerCase()) ||
      member.user.username.toLowerCase().includes(searchQuery.toLowerCase())
    )
  }, [data?.data?.rows, searchQuery])

  // Handle delete member
  const handleDeleteMember = async () => {
    if (!deleteMemberDialog.memberId) return

    try {
      await deleteMemberMutation.mutateAsync(deleteMemberDialog.memberId)
      showSuccess("Member removed successfully")
      setDeleteMemberDialog({ open: false, memberId: null, memberName: null, username: null })
    } catch (error) {
      showError(error)
    }
  }

  return (
    <>
      <InformationCard
        title="Tenant Members"
        description="Manage users who have access to this tenant"
        icon={Users}
      >
        <div className="space-y-4">
          {/* Search filter and Add button */}
          <div className="flex items-center justify-between gap-4">
            <div className="relative w-full max-w-sm">
              <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search members..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-8"
              />
            </div>
            <Button size="sm" className="gap-2" onClick={() => setAddMemberDialogOpen(true)}>
              <Plus className="h-4 w-4" />
              Add Member
            </Button>
          </div>

          {/* Horizontal line */}
          <div className="border-t" />

          {/* Scrollable content area */}
          <div className="max-h-[600px] overflow-y-auto pr-2">
            {isLoading && <ListSkeleton />}

            {error && (
              <EmptyState
                icon={AlertCircle}
                title="Failed to load members"
                description="Refresh the page or try again in a moment."
              />
            )}

            {filteredData && filteredData.length === 0 && (
              <EmptyState
                icon={Users}
                title={searchQuery ? "No members found" : "No members yet"}
                description={searchQuery ? "No members match your search." : "Add members to give users access to this tenant."}
              />
            )}

            {filteredData && filteredData.length > 0 && (
              <div className="space-y-2">
                {filteredData.map((member: TenantMember) => {
                  const isOwner = member.role === 'owner'
                  const isProtectedOwner = isSystemTenant && isOwner
                  return (
                    <MemberItem
                      key={member.tenant_member_id}
                      member={member}
                      onTransferOwnership={isOwner && !isProtectedOwner ? () =>
                        setTransferOwnershipDialog({
                          open: true,
                          owner: member
                        })
                      : undefined}
                      onDelete={isOwner ? undefined : (memberId, memberName) =>
                        setDeleteMemberDialog({
                          open: true,
                          memberId,
                          memberName,
                          username: member.user.username
                        })
                      }
                    />
                  )
                })}
              </div>
            )}
          </div>

          {/* Pagination controls */}
          {data && data.data && data.data.rows && data.data.rows.length > 0 && !searchQuery && (
            <div className="pt-4 border-t">
              <DataTablePagination table={table} rowCount={data.data.rows.length} />
            </div>
          )}
        </div>
      </InformationCard>

      {/* Add Member Dialog */}
      <AddMemberDialog
        open={addMemberDialogOpen}
        onOpenChange={setAddMemberDialogOpen}
        tenantId={tenantId}
      />

      {/* Transfer Ownership Dialog */}
      {transferOwnershipDialog.owner && (
        <TransferOwnershipDialog
          open={transferOwnershipDialog.open}
          onOpenChange={(open) => {
            if (!open) {
              setTransferOwnershipDialog({ open: false, owner: null })
            }
          }}
          tenantId={tenantId}
          currentOwner={transferOwnershipDialog.owner}
        />
      )}

      {/* Delete Member Dialog */}
      <DeleteConfirmationDialog
        open={deleteMemberDialog.open}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteMemberDialog({ open: false, memberId: null, memberName: null, username: null })
          }
        }}
        onConfirm={handleDeleteMember}
        title="Remove Member"
        description={
          deleteMemberDialog.memberName
            ? `Are you sure you want to remove "${deleteMemberDialog.memberName}" from this tenant?`
            : "Are you sure you want to remove this member from the tenant?"
        }
        confirmationText="This will permanently remove this member from the tenant and revoke their access. This action cannot be undone."
        itemName={deleteMemberDialog.username || ""}
        isDeleting={deleteMemberMutation.isPending}
      />
    </>
  )
}
