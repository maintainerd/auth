import { useState, useEffect, type FormEvent } from "react"
import { Plus, Search } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useRoles } from "@/hooks/useRoles"
import { useAssignUserRoles } from "@/hooks/useUsers"
import { useToast } from "@/hooks/useToast"

interface AssignUserRolesDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  userId: string
  existingRoleIds: string[]
}

export function AssignUserRolesDialog({
  open,
  onOpenChange,
  userId,
  existingRoleIds,
}: AssignUserRolesDialogProps) {
  const [selectedRoles, setSelectedRoles] = useState<string[]>([])
  const [searchQuery, setSearchQuery] = useState("")

  const { showSuccess, showError } = useToast()
  const assignRolesMutation = useAssignUserRoles()

  // Fetch all roles — only while the dialog is open.
  const { data: rolesData, isLoading: isLoadingRoles } = useRoles(
    {
      page: 1,
      limit: 100,
      sort_by: 'name',
      sort_order: 'asc'
    },
    { enabled: open }
  )

  // Reset form when dialog opens/closes
  useEffect(() => {
    if (!open) {
      setSelectedRoles([])
      setSearchQuery("")
    }
  }, [open])

  const handleRoleToggle = (roleId: string) => {
    setSelectedRoles(prev =>
      prev.includes(roleId)
        ? prev.filter(id => id !== roleId)
        : [...prev, roleId]
    )
  }

  const handleSelectAll = () => {
    const availableRoles = (rolesData?.rows ?? []).filter(
      role => !existingRoleIds.includes(role.role_id)
    )
    const availableRoleIds = availableRoles.map(r => r.role_id)
    
    if (selectedRoles.length === availableRoleIds.length) {
      setSelectedRoles([])
    } else {
      setSelectedRoles(availableRoleIds)
    }
  }

  const handleSubmit = async (e?: FormEvent) => {
    e?.preventDefault()
    if (selectedRoles.length === 0) {
      showError("Please select at least one role")
      return
    }

    try {
      await assignRolesMutation.mutateAsync({
        userId,
        data: { role_ids: selectedRoles }
      })

      showSuccess(`${selectedRoles.length} role${selectedRoles.length !== 1 ? 's' : ''} assigned successfully`)
      onOpenChange(false)
    } catch (error) {
      showError(error)
    }
  }

  const isLoading = assignRolesMutation.isPending

  // Filter roles based on search query and exclude already assigned roles
  const filteredRoles = rolesData?.rows?.filter(role =>
    !existingRoleIds.includes(role.role_id) &&
    (role.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    (role.description ?? "").toLowerCase().includes(searchQuery.toLowerCase()))
  ) ?? []

  const availableRolesCount = rolesData?.rows?.filter(
    role => !existingRoleIds.includes(role.role_id)
  ).length ?? 0

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Assign Roles to User</DialogTitle>
          <DialogDescription>
            Select roles to assign to this user. Already assigned roles are not shown.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Search and Select All */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label>Select Roles</Label>
              {availableRolesCount > 0 && (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={handleSelectAll}
                  className="h-7 text-xs"
                >
                  {selectedRoles.length === availableRolesCount
                    ? "Deselect All"
                    : "Select All"}
                </Button>
              )}
            </div>

            {/* Search */}
            <div className="relative">
              <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search roles..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-8"
              />
            </div>
          </div>

          {/* Roles List */}
          <div className="border rounded-lg max-h-[400px] overflow-y-auto">
            {isLoadingRoles && (
              <div className="text-center py-8 text-muted-foreground text-sm">
                Loading roles...
              </div>
            )}

            {!isLoadingRoles && filteredRoles.length === 0 && (
              <div className="text-center py-8 text-muted-foreground text-sm">
                {searchQuery
                  ? "No roles found matching your search"
                  : "All available roles are already assigned"}
              </div>
            )}

            {!isLoadingRoles && filteredRoles.length > 0 && (
              <div className="divide-y">
                {filteredRoles.map((role) => (
                  <div
                    key={role.role_id}
                    className="flex items-start gap-3 p-3 hover:bg-accent/50 transition-colors"
                  >
                    <Checkbox
                      id={role.role_id}
                      checked={selectedRoles.includes(role.role_id)}
                      onCheckedChange={() => handleRoleToggle(role.role_id)}
                      disabled={isLoading}
                    />
                    <label
                      htmlFor={role.role_id}
                      className="flex-1 cursor-pointer"
                    >
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{role.name}</span>
                        {role.is_system && (
                          <span className="text-xs px-1.5 py-0.5 rounded bg-secondary text-secondary-foreground">
                            System
                          </span>
                        )}
                        {role.is_default && (
                          <span className="text-xs px-1.5 py-0.5 rounded border">
                            Default
                          </span>
                        )}
                      </div>
                      <div className="text-xs text-muted-foreground mt-0.5">
                        {role.description}
                      </div>
                    </label>
                  </div>
                ))}
              </div>
            )}
          </div>

          {selectedRoles.length > 0 && (
            <p className="text-sm text-muted-foreground">
              {selectedRoles.length} role{selectedRoles.length !== 1 ? 's' : ''} selected
            </p>
          )}
        </div>

        <DialogFooter>
          <form onSubmit={handleSubmit} className="contents">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={selectedRoles.length === 0 || isLoading}
              className="gap-2"
            >
              {isLoading ? (
                <>Assigning...</>
              ) : (
                <>
                  <Plus className="h-4 w-4" />
                  Assign Roles
                </>
              )}
            </Button>
          </form>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
