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
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { useApis } from "@/hooks/useApis"
import { usePermissions } from "@/hooks/usePermissions"
import { useAddClientApis, useAddClientApiPermissions } from "@/hooks/useClients"
import { useToast } from "@/hooks/useToast"
import type { Api } from "@/services/api/api/types"

interface AddClientApiDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  clientId: string
}

export function AddClientApiDialog({ open, onOpenChange, clientId }: AddClientApiDialogProps) {
  const [apiSearchOpen, setApiSearchOpen] = useState(false)
  const [selectedApi, setSelectedApi] = useState<Api | null>(null)
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([])
  const [permissionSearchQuery, setPermissionSearchQuery] = useState("")

  const { showSuccess, showError } = useToast()
  const addClientApisMutation = useAddClientApis()
  const addClientApiPermissionsMutation = useAddClientApiPermissions()

  // Fetch all APIs
  const { data: apisData, isLoading: isLoadingApis } = useApis({
    page: 1,
    limit: 100,
    sort_by: 'display_name',
    sort_order: 'asc'
  })

  // Fetch permissions for selected API
  const { data: permissionsData, isLoading: isLoadingPermissions } = usePermissions({
    api_id: selectedApi?.api_id,
    page: 1,
    limit: 100,
    sort_by: 'name',
    sort_order: 'asc'
  })

  // Reset form when dialog opens/closes
  useEffect(() => {
    if (!open) {
      setSelectedApi(null)
      setSelectedPermissions([])
      setPermissionSearchQuery("")
    }
  }, [open])

  const handleApiSelect = (api: Api) => {
    setSelectedApi(api)
    setSelectedPermissions([])
    setApiSearchOpen(false)
  }

  const handlePermissionToggle = (permissionId: string) => {
    setSelectedPermissions(prev =>
      prev.includes(permissionId)
        ? prev.filter(id => id !== permissionId)
        : [...prev, permissionId]
    )
  }

  const handleSelectAllPermissions = () => {
    if (!permissionsData?.rows) return
    
    const allPermissionIds = permissionsData.rows.map(p => p.permission_id)
    if (selectedPermissions.length === allPermissionIds.length) {
      setSelectedPermissions([])
    } else {
      setSelectedPermissions(allPermissionIds)
    }
  }

  const handleSubmit = async () => {
    if (!selectedApi) {
      showError("Please select an API")
      return
    }

    try {
      // Add the API to the client
      await addClientApisMutation.mutateAsync({
        clientId,
        data: {
          api_uuids: [selectedApi.api_id]
        }
      })

      // If permissions are selected, add them to the client API
      if (selectedPermissions.length > 0) {
        await addClientApiPermissionsMutation.mutateAsync({
          clientId,
          apiId: selectedApi.api_id,
          data: {
            permission_uuids: selectedPermissions
          }
        })
      }

      const message = selectedPermissions.length > 0
        ? `API "${selectedApi.display_name}" with ${selectedPermissions.length} permission${selectedPermissions.length !== 1 ? 's' : ''} added successfully`
        : `API "${selectedApi.display_name}" added successfully`

      showSuccess(message)
      onOpenChange(false)
    } catch (error) {
      showError(error)
    }
  }

  const isLoading = addClientApisMutation.isPending || addClientApiPermissionsMutation.isPending

  // Filter permissions based on search query
  const filteredPermissions = permissionsData?.rows.filter(permission =>
    permission.name.toLowerCase().includes(permissionSearchQuery.toLowerCase()) ||
    permission.description.toLowerCase().includes(permissionSearchQuery.toLowerCase())
  ) || []

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Add API to Client</DialogTitle>
          <DialogDescription>
            Select an API and choose which permissions to grant to this client.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {/* API Selection */}
          <div className="space-y-2">
            <Label>
              Select API <span className="text-destructive">*</span>
            </Label>
            <Popover open={apiSearchOpen} onOpenChange={setApiSearchOpen}>
              <PopoverTrigger asChild>
                <Button
                  variant="outline"
                  role="combobox"
                  aria-expanded={apiSearchOpen}
                  className="w-full justify-between"
                  disabled={isLoading}
                >
                  <span className={selectedApi ? "" : "text-muted-foreground"}>
                    {selectedApi ? selectedApi.display_name : "Select an API"}
                  </span>
                  <Search className="h-4 w-4 opacity-50" />
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-full p-0" align="start">
                <Command>
                  <CommandInput placeholder="Search APIs..." />
                  <CommandList>
                    <CommandEmpty>
                      {isLoadingApis ? "Loading APIs..." : "No APIs found."}
                    </CommandEmpty>
                    <CommandGroup>
                      {apisData?.rows?.map((api) => (
                        <CommandItem
                          key={api.api_id}
                          value={api.api_id}
                          onSelect={() => handleApiSelect(api)}
                        >
                          <div className="flex flex-col">
                            <span className="font-medium">{api.display_name}</span>
                            <span className="text-xs text-muted-foreground">{api.name}</span>
                          </div>
                        </CommandItem>
                      ))}
                    </CommandGroup>
                  </CommandList>
                </Command>
              </PopoverContent>
            </Popover>
          </div>

          {/* Permissions Selection */}
          {selectedApi && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <Label>Select Permissions</Label>
                {permissionsData && permissionsData.rows.length > 0 && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={handleSelectAllPermissions}
                    className="h-7 text-xs"
                  >
                    {selectedPermissions.length === permissionsData.rows.length
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
              <div className="border rounded-lg max-h-[300px] overflow-y-auto">
                {isLoadingPermissions && (
                  <div className="text-center py-8 text-muted-foreground text-sm">
                    Loading permissions...
                  </div>
                )}

                {!isLoadingPermissions && filteredPermissions.length === 0 && (
                  <div className="text-center py-8 text-muted-foreground text-sm">
                    {permissionSearchQuery
                      ? "No permissions found matching your search"
                      : "No permissions available for this API"}
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
          )}
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
            disabled={!selectedApi || isLoading}
            className="gap-2"
          >
            {isLoading ? (
              <>Adding...</>
            ) : (
              <>
                <Plus className="h-4 w-4" />
                Add API
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}


