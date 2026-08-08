import { useState, useEffect } from "react"
import { Plus, Search } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useRoles } from "@/hooks/useRoles"
import { useAssignRegistrationFlowRoles } from "@/hooks/useRegistrationFlows"
import { useToast } from "@/hooks/useToast"
import { SelectableOptionRow } from "@/components/inputs"

interface AssignRegistrationFlowRolesDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  registrationFlowId: string
  existingRoleIds: string[]
}

export function AssignRegistrationFlowRolesDialog({
  open,
  onOpenChange,
  registrationFlowId,
  existingRoleIds,
}: AssignRegistrationFlowRolesDialogProps) {
  const [selectedRoles, setSelectedRoles] = useState<string[]>([])
  const [searchQuery, setSearchQuery] = useState("")

  const { showSuccess, showError } = useToast()
  const assignRolesMutation = useAssignRegistrationFlowRoles()

  // Fetch all roles — only while the dialog is open, so opening the details page
  // doesn't pull 100 roles nobody asked for (mirrors AssignUserRolesDialog).
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
    if (!rolesData?.rows) return
    
    const availableRoles = rolesData.rows.filter(
      role => !existingRoleIds.includes(role.role_id)
    )
    const availableRoleIds = availableRoles.map(r => r.role_id)
    
    if (selectedRoles.length === availableRoleIds.length) {
      setSelectedRoles([])
    } else {
      setSelectedRoles(availableRoleIds)
    }
  }

  const handleSubmit = async () => {
    if (selectedRoles.length === 0) {
      showError("Please select at least one role")
      return
    }

    try {
      await assignRolesMutation.mutateAsync({
        registrationFlowId,
        data: { role_uuids: selectedRoles }
      })

      showSuccess(`${selectedRoles.length} role${selectedRoles.length !== 1 ? 's' : ''} assigned successfully`)
      onOpenChange(false)
    } catch (error) {
      showError(error)
    }
  }

  const isLoading = assignRolesMutation.isPending

  // A registration flow is redeemed from a public link, so the backend refuses
  // system roles, inactive roles, and any role carrying an administrative
  // permission. System and inactive are visible here, so filter them out rather
  // than letting the operator pick something that will be rejected on save. The
  // administrative-permission case needs each role's permission set, so it stays
  // a server-side refusal whose message names the offending permission.
  const assignableRoles = rolesData?.rows.filter(
    role => !existingRoleIds.includes(role.role_id) && !role.is_system && role.status === "active"
  ) || []

  const filteredRoles = assignableRoles.filter(role =>
    role.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    role.description.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const availableRolesCount = assignableRoles.length

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Assign Roles to Registration Flow</DialogTitle>
          <DialogDescription>
            Anyone who registers through this flow&apos;s public link receives these roles, so only
            non-administrative roles can be assigned. To grant a privileged role, send an invite
            instead. Already assigned, system and inactive roles are not shown.
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
          <div className="border rounded-md divide-y max-h-[400px] overflow-y-auto">
            {isLoadingRoles ? (
              <div className="p-8 text-center text-muted-foreground">
                Loading roles...
              </div>
            ) : filteredRoles.length === 0 ? (
              <div className="p-8 text-center text-muted-foreground">
                {searchQuery
                  ? "No roles found matching your search"
                  : availableRolesCount === 0
                  ? "All available roles are already assigned"
                  : "No roles available"}
              </div>
            ) : (
              filteredRoles.map((role) => (
                <SelectableOptionRow
                  key={role.role_id}
                  selected={selectedRoles.includes(role.role_id)}
                  onToggle={() => handleRoleToggle(role.role_id)}
                  title={role.name}
                  description={role.description}
                />
              ))
            )}
          </div>

          {/* Selected Count */}
          {selectedRoles.length > 0 && (
            <div className="text-sm text-muted-foreground">
              {selectedRoles.length} role{selectedRoles.length !== 1 ? 's' : ''} selected
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={isLoading || selectedRoles.length === 0}
          >
            {isLoading ? (
              <>
                <Plus className="mr-2 h-4 w-4 animate-spin" />
                Assigning...
              </>
            ) : (
              <>
                <Plus className="mr-2 h-4 w-4" />
                Assign Roles
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
