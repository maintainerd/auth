import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { AppWindow, Eye, Globe, Link2, Unlink } from "lucide-react"
import { Button } from "@/components/ui/button"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton, StatusBadge } from "@/components/details"
import { DataTablePagination, usePaginationTable, RowActions, type RowActionItem } from "@/components/data-table"
import { useClients, useRemoveClientIdentityProvider } from "@/hooks/useClients"
import { useToast } from "@/hooks/useToast"
import type { Client } from "@/services/api/clients/types"
import { type PaginationState } from "@tanstack/react-table"
import { ConnectClientDialog } from "./ConnectClientDialog"

const CLIENT_TYPE_LABELS: Record<string, string> = {
  traditional: "Traditional Web",
  mobile: "Native Mobile",
  spa: "Single Page App",
  m2m: "Machine to Machine",
}

interface IdentityProviderClientsProps {
  providerId: string
  providerName: string
}

export function IdentityProviderClients({ providerId, providerName }: IdentityProviderClientsProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const removeConnectionMutation = useRemoveClientIdentityProvider()

  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [connectDialogOpen, setConnectDialogOpen] = useState(false)

  const { data, isLoading, isError } = useClients({
    identity_provider_id: providerId,
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    sort_by: "created_at",
    sort_order: "desc",
  })

  const table = usePaginationTable({
    pagination,
    onPaginationChange: setPagination,
    pageCount: data?.total_pages ?? 0,
  })

  const connectedClients = data?.rows ?? []
  const connectedClientIds = connectedClients.map((client) => client.client_id)

  const getProviderConnection = (client: Client) => {
    return client.connections?.find(
      (connection) => connection.identity_provider?.identity_provider_id === providerId,
    )
  }

  const disconnectClient = async (client: Client) => {
    const connectionId = getProviderConnection(client)?.client_identity_provider_id
    if (!connectionId) {
      showError("Unable to find this client connection")
      return
    }

    try {
      await removeConnectionMutation.mutateAsync({ clientId: client.client_id, connectionId })
      showSuccess(`Disconnected ${client.display_name} from ${providerName}`)
    } catch (error) {
      showError(error)
    }
  }

  return (
    <InformationCard
      title="Connected Clients"
      description="Downstream OAuth clients with this identity provider enabled as a login option."
      icon={AppWindow}
      action={
        <Button onClick={() => setConnectDialogOpen(true)} size="sm">
          <Link2 className="mr-2 h-4 w-4" />
          Connect a client
        </Button>
      }
    >
      <div className="space-y-4">
        {isLoading && <ListSkeleton />}

        {isError && (
          <p className="py-8 text-center text-sm text-destructive">Failed to load clients</p>
        )}

        {data && connectedClients.length === 0 && (
          <EmptyState
            icon={AppWindow}
            title="No connected clients"
            description={`No downstream clients have ${providerName} enabled as a login option yet.`}
          />
        )}

        {data && connectedClients.length > 0 && connectedClients.map((client) => {
          const connection = getProviderConnection(client)
          const canDisconnect = Boolean(connection?.identity_provider && !connection.identity_provider.is_system)
          const actions: RowActionItem[] = [
            {
              key: "view",
              label: "View Client",
              icon: Eye,
              onSelect: () => navigate(`/clients/${client.client_id}`),
            },
          ]

          if (canDisconnect) {
            actions.push({
              key: "disconnect",
              label: "Disconnect",
              icon: Unlink,
              destructive: true,
              onSelect: () => disconnectClient(client),
              confirm: {
                title: "Disconnect Client",
                description: `This will remove ${providerName} as a login option for this client. The client itself will not be deleted.`,
                destructive: true,
                itemName: client.name,
                confirmText: "Disconnect",
              },
            })
          }

          return (
            <div
              key={client.client_id}
              role="button"
              tabIndex={0}
              onClick={(e) => {
                const target = e.target as HTMLElement
                if (!e.currentTarget.contains(target) || target.closest("button, a")) return
                navigate(`/clients/${client.client_id}`)
              }}
              onKeyDown={(e) => {
                if (e.target !== e.currentTarget) return
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault()
                  navigate(`/clients/${client.client_id}`)
                }
              }}
              className="flex cursor-pointer items-start justify-between gap-3 rounded-lg border p-4 transition-colors hover:bg-accent/50"
            >
              <div className="flex min-w-0 items-start gap-3">
                <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                  <AppWindow className="size-5" />
                </div>
                <div className="min-w-0 space-y-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-sm font-semibold">{client.display_name}</span>
                    <StatusBadge status={client.status} />
                  </div>
                  <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                    <span className="font-mono">{client.name}</span>
                    <span>{CLIENT_TYPE_LABELS[client.client_type] ?? client.client_type}</span>
                    {client.domain && (
                      <span className="inline-flex items-center gap-1">
                        <Globe className="size-3" />
                        {client.domain}
                      </span>
                    )}
                  </div>
                </div>
              </div>
              <RowActions items={actions} />
            </div>
          )
        })}

        {data && data.total > 0 && (
          <div className="border-t pt-4">
            <DataTablePagination table={table} rowCount={data.total} />
          </div>
        )}
      </div>
      <ConnectClientDialog
        open={connectDialogOpen}
        onOpenChange={setConnectDialogOpen}
        providerId={providerId}
        providerName={providerName}
        existingClientIds={connectedClientIds}
      />
    </InformationCard>
  )
}
