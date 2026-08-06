import { useState, useEffect } from "react"
import { Search, Plus, User } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { FormSelectField } from "@/components/form"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useAddTenantMember, useTenantMembers } from "@/hooks/useTenantMembers"
import { useMembershipCandidates } from "@/hooks/useUsers"
import { useToast } from "@/hooks/useToast"
import { useAppSelector } from '@/store/hooks'

interface AddMemberDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  tenantId?: string
}

export function AddMemberDialog({ open, onOpenChange, tenantId: propTenantId }: AddMemberDialogProps) {
  const currentTenant = useAppSelector((state) => state.tenant.currentTenant)
  const tenantId = propTenantId || currentTenant?.tenant_id || ''
  const { showSuccess, showError } = useToast()
  const [selectedUserId, setSelectedUserId] = useState<string>("")
  const [role, setRole] = useState<'owner' | 'admin' | 'member'>('member')
  const [userSearchQuery, setUserSearchQuery] = useState("")
  
  const addMemberMutation = useAddTenantMember(tenantId)

  // Tenant members are only ever sourced from the SYSTEM tenant's shared user
  // pool: the backend rejects any user whose home tenant is not the system
  // tenant with a 403 (maintainerd-auth internal/tenant/service_member.go:
  // 210-220). GET /users, meanwhile, is hard-scoped to the caller's own tenant
  // — the handler ignores any tenant filter and always passes its own
  // tenant.CreateByUserUUID accepts ONLY system-tenant users, and the ordinary
  // user list is pinned to the caller's own tenant — so this dialog used to
  // offer choices that were guaranteed to 403 outside the system tenant.
  // /users/membership-candidates is the endpoint that returns the set the
  // backend will actually accept; the tenant is resolved server-side.
  const { data: candidatesData, isLoading: isLoadingUsers } = useMembershipCandidates(
    { search: userSearchQuery || undefined, page: 1, limit: 100 },
    { enabled: open },
  )

  // Fetch existing members to filter them out
  const { data: membersData } = useTenantMembers(tenantId, {
    page: 1,
    limit: 1000 // Get all members to filter
  })

  // Reset form when dialog opens/closes
  useEffect(() => {
    if (!open) {
      setSelectedUserId("")
      setRole('member')
      setUserSearchQuery("")
    }
  }, [open])

  const handleSubmit = async () => {
    if (!selectedUserId) {
      showError("Please select a user")
      return
    }

    try {
      await addMemberMutation.mutateAsync({
        user_id: selectedUserId,
        role
      })
      showSuccess("Member added successfully")
      onOpenChange(false)
    } catch (error) {
      showError(error)
    }
  }

  const isLoading = addMemberMutation.isPending

  const users = candidatesData?.rows ?? []
  const existingMemberUserIds = membersData?.data?.rows?.map(m => m.user.user_id) ?? []
  const hasOwner = membersData?.data?.rows?.some(m => m.role === 'owner') ?? false

  useEffect(() => {
    if (hasOwner && role === 'owner') {
      setRole('member')
    }
  }, [hasOwner, role])

  // Filter out users who are already members
  const availableUsers = users.filter(
    user => !existingMemberUserIds.includes(user.user_id)
  )

  // No client-side search filter: the endpoint already applied `search`, and
  // re-filtering here would drop rows the server matched on a field this
  // projection does not carry.
  const filteredUsers = availableUsers

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Add Member to Tenant</DialogTitle>
          <DialogDescription>
            Select a user and assign them a role for this tenant. Members are
            drawn from the system tenant's shared user pool.
          </DialogDescription>
        </DialogHeader>


        <div className="space-y-6 py-4">
          {/* Role Selection — only shown when tenant has no owner yet */}
          {!hasOwner && (
            <div className="space-y-2">
              <FormSelectField
                id="role"
                label="Member Role"
                options={[
                  { value: "member", label: "Member" },
                  { value: "admin", label: "Admin" },
                  { value: "owner", label: "Owner" },
                ]}
                value={role}
                onValueChange={(value) => setRole(value as 'owner' | 'admin' | 'member')}
                required
              />
              <p className="text-xs text-muted-foreground">
                Owners have full administrative access to the tenant. Only one owner is allowed.
              </p>
            </div>
          )}

          {/* User Selection */}
          <div className="space-y-3">
            <Label>
              Select User <span className="text-destructive">*</span>
            </Label>

            {/* Search Users */}
            <div className="relative">
              <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search users by name or email..."
                value={userSearchQuery}
                onChange={(e) => setUserSearchQuery(e.target.value)}
                className="pl-8"
              />
            </div>

            {/* Users List */}
            <div data-md-checkbox-sub-container className="border rounded-lg max-h-[300px] overflow-y-auto">
              {isLoadingUsers && (
                <div className="text-center py-8 text-muted-foreground text-sm">
                  Loading users...
                </div>
              )}

              {!isLoadingUsers && availableUsers.length === 0 && (
                <div className="text-center py-8 text-muted-foreground text-sm">
                  All active users have already been added as members
                </div>
              )}

              {!isLoadingUsers && availableUsers.length > 0 && filteredUsers.length === 0 && (
                <div className="text-center py-8 text-muted-foreground text-sm">
                  No users found matching your search
                </div>
              )}

              {!isLoadingUsers && filteredUsers.length > 0 && (
                <div className="divide-y">
                  {filteredUsers.map((user) => (
                    <div
                      key={user.user_id}
                      className={`flex items-start gap-3 p-3 hover:bg-accent/50 transition-colors cursor-pointer ${
                        selectedUserId === user.user_id ? 'bg-accent' : ''
                      }`}
                      onClick={() => setSelectedUserId(user.user_id)}
                    >
                      <Checkbox
                        id={user.user_id}
                        checked={selectedUserId === user.user_id}
                        onCheckedChange={(checked) => {
                          setSelectedUserId(checked ? user.user_id : "")
                        }}
                        disabled={isLoading}
                      />
                      <label
                        htmlFor={user.user_id}
                        className="flex-1 cursor-pointer"
                      >
                        <div className="flex items-center gap-2 mb-1">
                          <User className="h-4 w-4 text-muted-foreground" />
                          <span className="font-medium">{user.fullname}</span>
                        </div>
                        <div className="text-sm text-muted-foreground">{user.email}</div>
                        {user.username && user.username !== user.email && (
                          <div className="text-xs text-muted-foreground mt-0.5">
                            @{user.username}
                          </div>
                        )}
                      </label>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {selectedUserId && (
              <p className="text-sm text-muted-foreground">
                1 user selected
              </p>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleSubmit}
            disabled={!selectedUserId || isLoading}
            className="gap-2"
          >
            {isLoading ? (
              <>Adding...</>
            ) : (
              <>
                <Plus className="h-4 w-4" />
                Add Member
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
