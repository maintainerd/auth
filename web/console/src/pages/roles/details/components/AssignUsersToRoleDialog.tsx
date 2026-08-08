import { useState, useEffect, useMemo } from "react"
import { Search, Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog, DialogContent, DialogDescription, DialogFooter,
  DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import { useUsers } from "@/hooks/useUsers"
import { useAssignUserRoles } from "@/hooks/useUsers"
import { useToast } from "@/hooks/useToast"

interface AssignUsersToRoleDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  roleId: string
  existingUserIds: string[]
}

export function AssignUsersToRoleDialog({
  open, onOpenChange, roleId, existingUserIds,
}: AssignUsersToRoleDialogProps) {
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([])

  const { showSuccess, showError } = useToast()
  const assignRolesMutation = useAssignUserRoles()

  // Fetch users not already assigned to this role
  const { data: usersData, isLoading } = useUsers(
    { page: 1, limit: 100, sort_by: "username", sort_order: "asc" },
  )

  useEffect(() => {
    if (!open) {
      setSelectedUserIds([])
      setSearchQuery("")
    }
  }, [open])

  const availableUsers = useMemo(() => {
    if (!usersData?.rows) return []
    return usersData.rows
      .filter((u) => !existingUserIds.includes(u.user_id))
      .filter((u) => {
        if (!searchQuery) return true
        const q = searchQuery.toLowerCase()
        return (
          u.username.toLowerCase().includes(q) ||
          u.email.toLowerCase().includes(q) ||
          (u.fullname || "").toLowerCase().includes(q)
        )
      })
  }, [usersData, existingUserIds, searchQuery])

  const handleToggle = (userId: string) => {
    setSelectedUserIds((prev) =>
      prev.includes(userId) ? prev.filter((id) => id !== userId) : [...prev, userId],
    )
  }

  const handleSelectAll = () => {
    const allIds = availableUsers.map((u) => u.user_id)
    setSelectedUserIds((prev) =>
      prev.length === allIds.length ? [] : allIds,
    )
  }

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault()
    if (selectedUserIds.length === 0) {
      showError("Please select at least one user")
      return
    }

    try {
      // Assign the role to each selected user
      await Promise.all(
        selectedUserIds.map((userId) =>
          assignRolesMutation.mutateAsync({ userId, data: { role_ids: [roleId] } }),
        ),
      )
      showSuccess(
        `${selectedUserIds.length} user${selectedUserIds.length !== 1 ? "s" : ""} assigned to role`,
      )
      onOpenChange(false)
    } catch (error) {
      showError(error)
    }
  }

  const isLoadingMutation = assignRolesMutation.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Assign Users to Role</DialogTitle>
          <DialogDescription>
            Select users to assign this role to. Users already assigned are not shown.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="flex items-center justify-between">
            <Label>Select Users</Label>
            {availableUsers.length > 0 && (
              <Button type="button" variant="ghost" size="sm" onClick={handleSelectAll} className="h-7 text-xs">
                {selectedUserIds.length === availableUsers.length ? "Deselect All" : "Select All"}
              </Button>
            )}
          </div>

          <div className="relative">
            <Search className="absolute left-2 top-2.5 size-4 text-muted-foreground" />
            <Input
              placeholder="Search users..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-8"
            />
          </div>

          <div className="border rounded-lg max-h-[300px] overflow-y-auto">
            {isLoading && (
              <div className="text-center py-8 text-muted-foreground text-sm">Loading users...</div>
            )}

            {!isLoading && availableUsers.length === 0 && (
              <div className="text-center py-8 text-muted-foreground text-sm">
                {searchQuery ? "No users found matching your search" : "All users are already assigned to this role"}
              </div>
            )}

            {!isLoading && availableUsers.map((user) => (
              <div
                key={user.user_id}
                className="flex items-start gap-3 p-3 hover:bg-accent/50 transition-colors"
              >
                <Checkbox
                  id={user.user_id}
                  checked={selectedUserIds.includes(user.user_id)}
                  onCheckedChange={() => handleToggle(user.user_id)}
                  disabled={isLoadingMutation}
                />
                <label htmlFor={user.user_id} className="flex-1 cursor-pointer min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-sm">{user.fullname || user.username}</span>
                    <span className="text-xs text-muted-foreground font-mono">@{user.username}</span>
                  </div>
                  <div className="text-xs text-muted-foreground mt-0.5">{user.email}</div>
                </label>
              </div>
            ))}
          </div>

          {selectedUserIds.length > 0 && (
            <p className="text-sm text-muted-foreground">
              {selectedUserIds.length} user{selectedUserIds.length !== 1 ? "s" : ""} selected
            </p>
          )}
        </div>

        <DialogFooter>
          <form onSubmit={handleSubmit} className="contents">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={isLoadingMutation}>
              Cancel
            </Button>
            <Button type="submit" disabled={selectedUserIds.length === 0 || isLoadingMutation} className="gap-2">
              {isLoadingMutation ? "Assigning..." : <><Plus className="size-4" /> Assign</>}
            </Button>
          </form>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
