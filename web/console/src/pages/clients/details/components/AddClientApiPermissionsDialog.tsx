import { useState, useEffect } from "react"
import { Search, Plus } from "lucide-react"
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
import { usePermissions } from "@/hooks/usePermissions"
import { useAddClientApiPermissions } from "@/hooks/useClients"
import { useToast } from "@/hooks/useToast"

interface AddClientApiPermissionsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  clientId: string
  apiId: string
  apiName: string
  existingPermissionIds: string[]
}

export function AddClientApiPermissionsDialog({
  open,
  onOpenChange,
  clientId,
  apiId,
  apiName,
  existingPermissionIds
}: AddClientApiPermissionsDialogProps) {
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([])
  const [permissionSearchQuery, setPermissionSearchQuery] = useState("")

  const { showSuccess, showError } = useToast()
  const addClientApiPermissionsMutation = useAddClientApiPermissions()

  // Fetch permissions for the API
  const { data: permissionsData, isLoading: isLoadingPermissions } = usePermissions({
    api_id: apiId,
    page: 1,
    limit: 100,
    sort_by: 'name',
    sort_order: 'asc'
  })

  // Reset form when dialog opens/closes
  useEffect(() => {
    if (!open) {
      setSelectedPermissions([])
      setPermissionSearchQuery("")
    }
  }, [open])

  const handlePermissionToggle = (permissionId: string) => {
    setSelectedPermissions(prev =>
      prev.includes(permissionId)
        ? prev.filter(id => id !== permissionId)
        : [...prev, permissionId]
    )
  }

  const handleSelectAllPermissions = () => {
    if (!availablePermissions) return
    
    const allPermissionIds = availablePermissions.map(p => p.permission_id)
    if (selectedPermissions.length === allPermissionIds.length) {
      setSelectedPermissions([])
    } else {
      setSelectedPermissions(allPermissionIds)
    }
  }

  const handleSubmit = async () => {
    if (selectedPermissions.length === 0) {
      showError("Please select at least one permission")
      return
    }

    try {
      await addClientApiPermissionsMutation.mutateAsync({
        clientId,
        apiId,
        data: {
          permission_uuids: selectedPermissions
        }
      })

      showSuccess(`${selectedPermissions.length} permission${selectedPermissions.length !== 1 ? 's' : ''} added successfully`)
      onOpenChange(false)
    } catch (error) {
      showError(error)
    }
  }

  const isLoading = addClientApiPermissionsMutation.isPending

  // Filter out permissions that are already added to the client API
  const availablePermissions = permissionsData?.rows.filter(
    permission => !existingPermissionIds.includes(permission.permission_id)
  ) || []

  // Filter permissions based on search query
  const filteredPermissions = availablePermissions.filter(permission =>
    permission.name.toLowerCase().includes(permissionSearchQuery.toLowerCase()) ||
    permission.description.toLowerCase().includes(permissionSearchQuery.toLowerCase())
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Add Permissions to {apiName}</DialogTitle>
          <DialogDescription>
            Select permissions to grant to this client for the selected API.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label>Select Permissions</Label>
              {availablePermissions.length > 0 && (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={handleSelectAllPermissions}
                  className="h-7 text-xs"
                >
                  {selectedPermissions.length === availablePermissions.length
                    ? "Deselect All"
                    : "Select All"}
                </Button>
              )}
            </div>

            {/* Search Permissions */}
            <div className="relative">
              <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search permissions..."
                value={permissionSearchQuery}
                onChange={(e) => setPermissionSearchQuery(e.target.value)}
                className="pl-8"
              />
            </div>

            {/* Permissions List */}
            <div className="border rounded-lg max-h-[400px] overflow-y-auto">
              {isLoadingPermissions && (
                <div className="text-center py-8 text-muted-foreground text-sm">
                  Loading permissions...
                </div>
              )}

              {!isLoadingPermissions && availablePermissions.length === 0 && (
                <div className="text-center py-8 text-muted-foreground text-sm">
                  All permissions for this API have already been added to the client
                </div>
              )}

              {!isLoadingPermissions && availablePermissions.length > 0 && filteredPermissions.length === 0 && (
                <div className="text-center py-8 text-muted-foreground text-sm">
                  No permissions found matching your search
                </div>
              )}

              {!isLoadingPermissions && filteredPermissions.length > 0 && (
                <div className="divide-y">
                  {filteredPermissions.map((permission) => (
                    <div
                      key={permission.permission_id}
                      className="flex items-start gap-3 p-3 hover:bg-accent/50 transition-colors"
                    >
                      <Checkbox
                        id={permission.permission_id}
                        checked={selectedPermissions.includes(permission.permission_id)}
                        onCheckedChange={() => handlePermissionToggle(permission.permission_id)}
                        disabled={isLoading}
                      />
                      <label
                        htmlFor={permission.permission_id}
                        className="flex-1 cursor-pointer"
                      >
                        <div className="font-medium font-mono text-sm">{permission.name}</div>
                        <div className="text-xs text-muted-foreground mt-0.5">
                          {permission.description}
                        </div>
                      </label>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {selectedPermissions.length > 0 && (
              <p className="text-sm text-muted-foreground">
                {selectedPermissions.length} permission{selectedPermissions.length !== 1 ? 's' : ''} selected
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
            disabled={selectedPermissions.length === 0 || isLoading}
            className="gap-2"
          >
            {isLoading ? (
              <>Adding...</>
            ) : (
              <>
                <Plus className="h-4 w-4" />
                Add Permissions
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}


