import { useState } from "react"
import { ChevronDown, ChevronRight, Key, Plus, Server, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { SystemBadge } from "@/components/badges"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton, StatusBadge } from "@/components/details"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useClientApis, useRemoveClientApi, useRemoveClientApiPermission } from "@/hooks/useClients"
import { useToast } from "@/hooks/useToast"
import { AddClientApiDialog } from "./AddClientApiDialog"
import { AddClientApiPermissionsDialog } from "./AddClientApiPermissionsDialog"
import type { ClientApiAssociation } from "@/services/api/clients/types"

interface ClientApisProps {
  clientId: string
}

export function ClientApis({ clientId }: ClientApisProps) {
  const [expandedApis, setExpandedApis] = useState<Set<string>>(new Set())
  const [addApiDialogOpen, setAddApiDialogOpen] = useState(false)
  const [addPermissionsDialog, setAddPermissionsDialog] = useState<{
    open: boolean
    apiId: string | null
    apiName: string | null
    existingPermissionIds: string[]
  }>({
    open: false,
    apiId: null,
    apiName: null,
    existingPermissionIds: []
  })

  const { data, isLoading, error } = useClientApis(clientId)
  const removeApiMutation = useRemoveClientApi()
  const removePermissionMutation = useRemoveClientApiPermission()
  const { showSuccess, showError } = useToast()

  const toggleApiExpanded = (apiId: string) => {
    const newExpanded = new Set(expandedApis)
    if (newExpanded.has(apiId)) {
      newExpanded.delete(apiId)
    } else {
      newExpanded.add(apiId)
    }
    setExpandedApis(newExpanded)
  }

  const removeApi = async (api: ClientApiAssociation) => {
    try {
      await removeApiMutation.mutateAsync({
        clientId,
        apiId: api.api.api_id
      })
      showSuccess("API removed from client successfully")
    } catch (error) {
      showError(error)
    }
  }

  const removePermission = async (apiId: string, permissionId: string) => {
    try {
      await removePermissionMutation.mutateAsync({
        clientId,
        apiId,
        permissionId
      })
      showSuccess("Permission removed from client API successfully")
    } catch (error) {
      showError(error)
    }
  }

  return (
    <>
      <InformationCard
        title="API Permissions"
        description="Manage API access and assigned permissions for this client."
        icon={Server}
        action={
          <Button size="sm" onClick={() => setAddApiDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Add API
          </Button>
        }
      >
        <div className="space-y-4">
          {isLoading && <ListSkeleton />}

          {error && (
            <p className="py-8 text-center text-sm text-destructive">Failed to load APIs</p>
          )}

          {data && data.apis.length === 0 && (
            <EmptyState
              icon={Server}
              title="No API permissions"
              description="This client has no APIs assigned yet. Add an API to grant access and permissions."
            />
          )}

          {data && data.apis.length > 0 && (
            <div className="space-y-3">
              {data.apis.map((item) => {
                const isExpanded = expandedApis.has(item.api.api_id)
                const permissions = item.permissions || []
                const apiActions: RowActionItem[] = [
                  {
                    key: "add-permission",
                    label: "Add Permission",
                    icon: Plus,
                    onSelect: () => setAddPermissionsDialog({
                      open: true,
                      apiId: item.api.api_id,
                      apiName: item.api.display_name,
                      existingPermissionIds: permissions.map((permission) => permission.permission_id),
                    }),
                  },
                  {
                    key: "remove-api",
                    label: "Remove API",
                    icon: Trash2,
                    destructive: true,
                    separatorBefore: true,
                    onSelect: () => removeApi(item),
                    confirm: {
                      title: "Remove API from Client",
                      description: "This will remove the API and all of its permissions from this client. The API itself will not be deleted.",
                      destructive: true,
                      itemName: item.api.display_name,
                    },
                  },
                ]

                return (
                  <div data-md-listing-item key={item.client_api_id} className="rounded-lg border p-4">
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex min-w-0 items-start gap-3">
                        <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                          <Server className="size-5" />
                        </div>
                        <div className="min-w-0 space-y-1">
                          <div className="flex flex-wrap items-center gap-2">
                            <span className="text-sm font-semibold">{item.api.display_name}</span>
                            <StatusBadge status={item.api.status} />
                            <SystemBadge isSystem={item.api.is_system} />
                          </div>
                          <p className="text-sm text-muted-foreground">{item.api.description}</p>
                          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                            <span className="font-mono">{item.api.name}</span>
                            {item.api.identifier && <span className="font-mono">{item.api.identifier}</span>}
                          </div>
                        </div>
                      </div>
                      <div className="flex shrink-0 items-center gap-2">
                        <Badge variant="secondary" className="text-xs">
                          {permissions.length} {permissions.length === 1 ? "permission" : "permissions"}
                        </Badge>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 w-8 p-0"
                          aria-expanded={isExpanded}
                          onClick={() => toggleApiExpanded(item.api.api_id)}
                        >
                          <span className="sr-only">{isExpanded ? "Collapse permissions" : "Expand permissions"}</span>
                          {isExpanded ? <ChevronDown className="size-4" /> : <ChevronRight className="size-4" />}
                        </Button>
                        <RowActions items={apiActions} />
                      </div>
                    </div>

                    {isExpanded && (
                      <div className="mt-4 border-t pt-4">
                        <div className="mb-3 flex items-center justify-between gap-3">
                          <div>
                            <h4 className="text-sm font-semibold">Permissions</h4>
                            <p className="text-sm text-muted-foreground">Permissions granted for this API.</p>
                          </div>
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-8 gap-2"
                            onClick={() => setAddPermissionsDialog({
                              open: true,
                              apiId: item.api.api_id,
                              apiName: item.api.display_name,
                              existingPermissionIds: permissions.map((permission) => permission.permission_id),
                            })}
                          >
                            <Plus className="size-4" />
                            Add Permission
                          </Button>
                        </div>

                        {permissions.length === 0 ? (
                          <EmptyState
                            icon={Key}
                            title="No permissions"
                            description="This API is assigned to the client, but no permissions are granted yet."
                          />
                        ) : (
                          <div className="space-y-2">
                            {permissions.map((permission) => {
                              const permissionActions: RowActionItem[] = [
                                {
                                  key: "remove-permission",
                                  label: "Remove Permission",
                                  icon: Trash2,
                                  destructive: true,
                                  onSelect: () => removePermission(item.api.api_id, permission.permission_id),
                                  confirm: {
                                    title: "Remove Permission",
                                    description: `This will remove "${permission.name}" from this client's access to ${item.api.display_name}.`,
                                    confirmText: "Remove",
                                  },
                                },
                              ]

                              return (
                                <div
                                  key={permission.permission_id}
                                  data-md-listing-nested
                                  className="flex items-start justify-between gap-3 rounded-md border bg-background p-3"
                                >
                                  <div className="min-w-0 space-y-1">
                                    <div className="flex flex-wrap items-center gap-2">
                                      <span className="font-mono text-sm font-medium">{permission.name}</span>
                                      <StatusBadge status={permission.status} />
                                      <SystemBadge isSystem={permission.is_system} />
                                    </div>
                                    <p className="text-sm text-muted-foreground">{permission.description}</p>
                                  </div>
                                  <RowActions items={permissionActions} />
                                </div>
                              )
                            })}
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </InformationCard>

      {/* Add API Dialog */}
      <AddClientApiDialog
        open={addApiDialogOpen}
        onOpenChange={setAddApiDialogOpen}
        clientId={clientId}
      />

      {/* Add Permissions Dialog */}
      {addPermissionsDialog.apiId && (
        <AddClientApiPermissionsDialog
          open={addPermissionsDialog.open}
          onOpenChange={(open) => setAddPermissionsDialog({
            open,
            apiId: null,
            apiName: null,
            existingPermissionIds: []
          })}
          clientId={clientId}
          apiId={addPermissionsDialog.apiId}
          apiName={addPermissionsDialog.apiName || ""}
          existingPermissionIds={addPermissionsDialog.existingPermissionIds}
        />
      )}
    </>
  )
}
