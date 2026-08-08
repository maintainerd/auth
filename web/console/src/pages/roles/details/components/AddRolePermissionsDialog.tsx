import { useEffect, useState } from "react"
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
import { useDebouncedSearch } from "@/hooks/useDebouncedSearch"
import { usePermissions } from "@/hooks/usePermissions"
import { useAddRolePermissions } from "@/hooks/useRoles"
import { useToast } from "@/hooks/useToast"
import type { Api } from "@/services/api/api/types"

/**
 * One page is all a picker fetches. Both the API and the permission search are
 * server-side, so this caps a single response, not what is reachable — see the
 * "refine your search" hints, which are what tell a user the 101st row exists.
 */
const PAGE_LIMIT = 100

interface AddRolePermissionsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  roleId: string
  existingPermissionIds: string[]
}

export function AddRolePermissionsDialog({
  open,
  onOpenChange,
  roleId,
  existingPermissionIds,
}: AddRolePermissionsDialogProps) {
  const [apiSearchOpen, setApiSearchOpen] = useState(false)
  const [selectedApi, setSelectedApi] = useState<Api | null>(null)
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([])

  const { showSuccess, showError } = useToast()
  const addRolePermissionsMutation = useAddRolePermissions()

  // The API list is searched on the server (APIFilterDTO.DisplayName is matched
  // with ILIKE — internal/iam/repository_api.go:113), not filtered inside an
  // already-truncated page: a client-side filter over the first 100 rows made
  // API 101 impossible to pick.
  const {
    searchInput: apiSearchInput,
    debouncedValue: apiSearchTerm,
    handleSearchChange: handleApiSearchChange,
    setSearchValue: setApiSearchValue,
  } = useDebouncedSearch()

  const { data: apisData, isLoading: isLoadingApis } = useApis(
    {
      page: 1,
      limit: PAGE_LIMIT,
      sort_by: 'display_name',
      sort_order: 'asc',
      display_name: apiSearchTerm || undefined,
    },
    // Gated rather than conditionally mounted: a closed dialog must not fetch
    // every API in the tenant, but keeping the body mounted leaves the reset
    // effect below as the single place the draft selection is cleared.
    { enabled: open },
  )

  useEffect(() => {
    if (!open) {
      setSelectedApi(null)
      setSelectedPermissions([])
      setApiSearchValue("")
    }
  }, [open, setApiSearchValue])

  const handleApiSelect = (api: Api) => {
    setSelectedApi(api)
    setSelectedPermissions([])
    setApiSearchOpen(false)
  }

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault()
    if (selectedPermissions.length === 0) {
      showError("Please select at least one permission")
      return
    }

    try {
      await addRolePermissionsMutation.mutateAsync({
        roleId,
        data: {
          permissions: selectedPermissions
        }
      })

      showSuccess(`${selectedPermissions.length} permission${selectedPermissions.length !== 1 ? 's' : ''} added successfully`)
      onOpenChange(false)
    } catch (error) {
      showError(error)
    }
  }

  const isLoading = addRolePermissionsMutation.isPending

  const apiRows = apisData?.rows ?? []
  // The server may hold more APIs than one page returns; saying so is what stops
  // a user concluding the API is not registered.
  const hasMoreApis = (apisData?.total ?? 0) > apiRows.length

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Add Permissions to Role</DialogTitle>
          <DialogDescription>
            Select an API and choose which permissions to grant to this role.
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
                {/*
                  shouldFilter={false} because the server already narrowed the
                  page. cmdk's built-in filter scores each item against its
                  `value`, which here is the API's UUID, so leaving it on hid
                  every row as soon as the user typed a word.
                */}
                <Command shouldFilter={false}>
                  <CommandInput
                    placeholder="Search APIs..."
                    value={apiSearchInput}
                    onValueChange={handleApiSearchChange}
                  />
                  <CommandList>
                    <CommandEmpty>
                      {isLoadingApis ? "Loading APIs..." : "No APIs found."}
                    </CommandEmpty>
                    <CommandGroup>
                      {apiRows.map((api) => (
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
                  {!isLoadingApis && hasMoreApis && (
                    <p className="border-t px-3 py-2 text-xs text-muted-foreground">
                      Showing the first {apiRows.length} of {apisData?.total ?? 0} APIs. Refine your search to narrow the list.
                    </p>
                  )}
                </Command>
              </PopoverContent>
            </Popover>
          </div>

          {/*
            Keyed on the API so switching APIs remounts the picker and clears its
            search — a leftover query would otherwise hide the new API's
            permissions behind a filter the user cannot see they still have on.
          */}
          {selectedApi && (
            <PermissionPicker
              key={selectedApi.api_id}
              apiId={selectedApi.api_id}
              existingPermissionIds={existingPermissionIds}
              value={selectedPermissions}
              onChange={setSelectedPermissions}
              disabled={isLoading}
            />
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
          </form>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface PermissionPickerProps {
  apiId: string
  existingPermissionIds: string[]
  value: string[]
  onChange: (permissionIds: string[]) => void
  disabled: boolean
}

function PermissionPicker({
  apiId,
  existingPermissionIds,
  value,
  onChange,
  disabled,
}: PermissionPickerProps) {
  // Search is sent to the API (PermissionFilterDTO.Name is matched with ILIKE),
  // not applied to an already-truncated page — filtering the first 100 rows
  // client-side made permission 101 impossible to attach.
  const { searchInput, debouncedValue, handleSearchChange, handleKeyDown } = useDebouncedSearch()

  const { data: permissionsData, isLoading: isLoadingPermissions } = usePermissions({
    api_id: apiId,
    page: 1,
    limit: PAGE_LIMIT,
    sort_by: 'name',
    sort_order: 'asc',
    name: debouncedValue || undefined,
  })

  // The API has no "exclude these ids" filter, so permissions the role already
  // holds are the one thing still dropped client-side.
  const availablePermissions = permissionsData?.rows.filter(
    permission => !existingPermissionIds.includes(permission.permission_id)
  ) ?? []

  const visiblePermissionIds = availablePermissions.map(p => p.permission_id)
  // Derived from the rendered rows, never from the full fetched page. Comparing
  // against the unfiltered list is what let "Select All" under a `read` search
  // silently tick every write and delete permission the API returned.
  const selectedVisibleCount = visiblePermissionIds.filter(id => value.includes(id)).length
  const allVisibleSelected =
    visiblePermissionIds.length > 0 && selectedVisibleCount === visiblePermissionIds.length
  // Tri-state, so a partial selection reports aria-checked="mixed" instead of
  // an unchecked box that hides the fact that some rows are already ticked.
  const selectAllState: boolean | "indeterminate" = allVisibleSelected
    ? true
    : selectedVisibleCount > 0
      ? "indeterminate"
      : false

  const handleSelectAllPermissions = () => {
    onChange(
      allVisibleSelected
        ? value.filter(id => !visiblePermissionIds.includes(id))
        : [...new Set([...value, ...visiblePermissionIds])]
    )
  }

  const handlePermissionToggle = (permissionId: string) => {
    onChange(
      value.includes(permissionId)
        ? value.filter(id => id !== permissionId)
        : [...value, permissionId]
    )
  }

  // The server may hold more matches than one page returns; saying so is what
  // stops a user concluding the permission does not exist.
  const hasMoreMatches = (permissionsData?.total ?? 0) > (permissionsData?.rows?.length ?? 0)

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Label>Select Permissions</Label>
        {availablePermissions.length > 0 && (
          <div className="flex items-center gap-2">
            <Checkbox
              id="select-all-permissions"
              checked={selectAllState}
              onCheckedChange={handleSelectAllPermissions}
              disabled={disabled}
            />
            <Label
              htmlFor="select-all-permissions"
              className="cursor-pointer text-xs font-normal"
            >
              Select all
            </Label>
          </div>
        )}
      </div>

      {/* Search Permissions */}
      <div className="relative">
        <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder="Search permissions..."
          value={searchInput}
          onChange={(e) => handleSearchChange(e.target.value)}
          onKeyDown={handleKeyDown}
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

        {!isLoadingPermissions && availablePermissions.length === 0 && (
          <div className="text-center py-8 text-muted-foreground text-sm">
            {debouncedValue
              ? "No permissions found matching your search"
              : "All permissions for this API have already been added to the role"}
          </div>
        )}

        {!isLoadingPermissions && availablePermissions.length > 0 && (
          <div className="divide-y">
            {availablePermissions.map((permission) => (
              <div
                key={permission.permission_id}
                className="flex items-start gap-3 p-3 hover:bg-accent/50 transition-colors"
              >
                <Checkbox
                  id={permission.permission_id}
                  checked={value.includes(permission.permission_id)}
                  onCheckedChange={() => handlePermissionToggle(permission.permission_id)}
                  disabled={disabled}
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

      {!isLoadingPermissions && hasMoreMatches && (
        <p className="text-sm text-muted-foreground">
          Showing the first {permissionsData?.rows?.length ?? 0} of {permissionsData?.total ?? 0} permissions. Refine your search to narrow the list.
        </p>
      )}

      {value.length > 0 && (
        <p className="text-sm text-muted-foreground">
          {value.length} permission{value.length !== 1 ? 's' : ''} selected
        </p>
      )}
    </div>
  )
}
