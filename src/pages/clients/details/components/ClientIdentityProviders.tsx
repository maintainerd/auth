import { useState } from "react"
import { useNavigate, useParams } from "react-router-dom"
import { safeFormat } from "@/lib/formatDate"
import { Building2, Eye, Link2, Unlink, Settings, Check } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { Input } from "@/components/ui/input"
import { InformationCard } from "@/components/card"
import { EmptyState, StatusBadge } from "@/components/details"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { SystemBadge } from "@/components/badges"
import { PROVIDER_LABELS, ProviderLogo } from "@/components/provider-config"
import { useRemoveClientIdentityProvider, useUpdateClientIdentityProvider } from "@/hooks/useClients"
import { useToast } from "@/hooks/useToast"
import type { ClientResponse, ClientIdentityProviderConnection } from "@/services/api/clients/types"
import { ConnectProviderDialog } from "./ConnectProviderDialog"

interface ClientIdentityProvidersProps {
  client: ClientResponse
}

export function ClientIdentityProviders({ client }: ClientIdentityProvidersProps) {
  const { clientId } = useParams<{ clientId: string }>()
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const removeConnectionMutation = useRemoveClientIdentityProvider()
  const updateConnectionMutation = useUpdateClientIdentityProvider()
  const [connectDialogOpen, setConnectDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)

  const connections = [...(client.connections ?? [])].sort((a, b) => a.display_order - b.display_order)
  const connectedProviderIds = connections.map((connection) => connection.identity_provider.identity_provider_id)

  const disconnectProvider = async (connection: ClientIdentityProviderConnection) => {
    if (!clientId) return

    try {
      await removeConnectionMutation.mutateAsync({
        clientId,
        connectionId: connection.client_identity_provider_id,
      })
      showSuccess(`Disconnected ${connection.identity_provider.display_name} from ${client.display_name}`)
    } catch (error) {
      showError(error)
    }
  }

  const toggleEdit = (id: string) => {
    setEditingId(editingId === id ? null : id)
  }

  const updateConnection = async (connection: ClientIdentityProviderConnection, data: { is_default?: boolean; enabled?: boolean; display_order?: number }) => {
    if (!clientId) return
    try {
      await updateConnectionMutation.mutateAsync({
        clientId,
        connectionId: connection.client_identity_provider_id,
        data,
      })
      showSuccess("Connection updated")
    } catch (error) {
      showError(error)
    }
  }

  return (
    <InformationCard
      title="Identity Providers"
      description="Identity sources this client can use during sign-in."
      icon={Building2}
      action={
        <Button onClick={() => setConnectDialogOpen(true)} size="sm">
          <Link2 className="mr-2 size-4" />
          Connect provider
        </Button>
      }
    >
      <div className="space-y-4">
        {connections.length === 0 ? (
          <EmptyState
            icon={Building2}
            title="No identity providers connected"
            description="Connect an identity provider to make it available as a sign-in option for this client."
          />
        ) : (
          connections.map((connection) => {
            const provider = connection.identity_provider
            const actions: RowActionItem[] = [
              {
                key: "view",
                label: "View Provider",
                icon: Eye,
                onSelect: () => navigate(`/providers/identity/${provider.identity_provider_id}`),
              },
              {
                key: "edit",
                label: editingId === connection.client_identity_provider_id ? "Done" : "Edit Connection",
                icon: editingId === connection.client_identity_provider_id ? Check : Settings,
                onSelect: () => toggleEdit(connection.client_identity_provider_id),
              },
            ]

            if (!provider.is_system) {
              actions.push({
                key: "disconnect",
                label: "Disconnect",
                icon: Unlink,
                destructive: true,
                onSelect: () => disconnectProvider(connection),
                confirm: {
                  title: "Disconnect Identity Provider",
                  description: `This will remove ${provider.display_name} as a sign-in option for this client. The provider itself will not be deleted.`,
                  destructive: true,
                  itemName: provider.name,
                  confirmText: "Disconnect",
                },
              })
            }

            return (
              <div data-md-listing-item key={connection.client_identity_provider_id} className="rounded-lg border p-4">
                <div className="flex items-start justify-between gap-3">
                  <div className="flex min-w-0 items-start gap-3">
                    <ProviderLogo provider={provider.provider} className="size-10" />
                    <div className="min-w-0 space-y-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-sm font-semibold">{provider.display_name}</span>
                        <Badge variant="secondary" className="text-xs">
                          {PROVIDER_LABELS[provider.provider] ?? provider.provider}
                        </Badge>
                        <StatusBadge status={provider.status} />
                        <SystemBadge isSystem={provider.is_system} />
                        {connection.is_default && (
                          <Badge variant="outline" className="text-xs">
                            Default
                          </Badge>
                        )}
                        {!connection.enabled && (
                          <Badge variant="outline" className="text-xs">
                            Disabled
                          </Badge>
                        )}
                      </div>
                      <p className="text-sm text-muted-foreground">{provider.name}</p>
                      <p className="break-all font-mono text-xs text-muted-foreground">{provider.identifier}</p>
                    </div>
                  </div>
                  <RowActions items={actions} />
                </div>

                <div className="mt-4 grid gap-3 border-t pt-4 text-xs text-muted-foreground md:grid-cols-3">
                  <div>
                    <span className="font-medium">Connection UUID</span>
                    <p className="break-all font-mono">{connection.client_identity_provider_id}</p>
                  </div>
                  <div>
                    <span className="font-medium">Display order</span>
                    <p>{connection.display_order}</p>
                  </div>
                  <div>
                    <span className="font-medium">Connected</span>
                    <p>{safeFormat(connection.created_at, "PPpp")}</p>
                  </div>
                </div>

                {editingId === connection.client_identity_provider_id && (
                  <div className="mt-3 border-t pt-3 space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-medium">Default</span>
                      <Switch
                        checked={connection.is_default}
                        onCheckedChange={(v) => updateConnection(connection, { is_default: v })}
                      />
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-medium">Enabled</span>
                      <Switch
                        checked={connection.enabled}
                        onCheckedChange={(v) => updateConnection(connection, { enabled: v })}
                      />
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-medium">Display Order</span>
                      <Input
                        type="number"
                        className="w-20 h-7 text-xs"
                        defaultValue={connection.display_order}
                        onBlur={(e) => {
                          const v = parseInt(e.target.value, 10)
                          if (!isNaN(v)) updateConnection(connection, { display_order: v })
                        }}
                      />
                    </div>
                  </div>
                )}
              </div>
            )
          })
        )}
      </div>

      <ConnectProviderDialog
        open={connectDialogOpen}
        onOpenChange={setConnectDialogOpen}
        clientId={client.client_id}
        clientName={client.display_name}
        existingProviderIds={connectedProviderIds}
      />
    </InformationCard>
  )
}
