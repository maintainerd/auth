import { useEffect, useMemo, useState } from "react"
import { AppWindow, Link2, Search } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useAddClientIdentityProvider, useClients } from "@/hooks/useClients"
import { useToast } from "@/hooks/useToast"
import { SelectableOptionRow } from "@/components/inputs"
import type { Client } from "@/services/api/clients/types"

const CLIENT_TYPE_LABELS: Record<string, string> = {
  traditional: "Traditional Web",
  mobile: "Native Mobile",
  spa: "Single Page App",
  m2m: "Machine to Machine",
}

interface ConnectClientDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  providerId: string
  providerName: string
  existingClientIds: string[]
}

export function ConnectClientDialog({
  open,
  onOpenChange,
  providerId,
  providerName,
  existingClientIds,
}: ConnectClientDialogProps) {
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedClientIds, setSelectedClientIds] = useState<string[]>([])
  const { showSuccess, showError } = useToast()
  const addConnectionMutation = useAddClientIdentityProvider()

  const { data: clientsData, isLoading } = useClients({
    page: 1,
    limit: 100,
    sort_by: "display_name",
    sort_order: "asc",
  })

  useEffect(() => {
    if (!open) {
      setSearchQuery("")
      setSelectedClientIds([])
    }
  }, [open])

  const availableClients = useMemo(() => {
    const existing = new Set(existingClientIds)
    const q = searchQuery.trim().toLowerCase()

    return (clientsData?.rows ?? [])
      .filter((client) => !existing.has(client.client_id))
      .filter((client) => {
        if (!q) return true
        return [
          client.display_name,
          client.name,
          client.domain ?? "",
          CLIENT_TYPE_LABELS[client.client_type] ?? client.client_type,
        ].some((value) => value.toLowerCase().includes(q))
      })
  }, [clientsData?.rows, existingClientIds, searchQuery])

  const availableClientIds = availableClients.map((client) => client.client_id)
  const isSubmitting = addConnectionMutation.isPending

  const toggleClient = (clientId: string) => {
    setSelectedClientIds((prev) =>
      prev.includes(clientId) ? prev.filter((id) => id !== clientId) : [...prev, clientId],
    )
  }

  const toggleAll = () => {
    setSelectedClientIds((prev) =>
      prev.length === availableClientIds.length ? [] : availableClientIds,
    )
  }

  const describeClient = (client: Client) => {
    const parts = [
      client.name,
      CLIENT_TYPE_LABELS[client.client_type] ?? client.client_type,
      client.domain ?? "",
    ].filter(Boolean)

    return parts.join(" · ")
  }

  const handleSubmit = async () => {
    if (selectedClientIds.length === 0) {
      showError("Please select at least one client")
      return
    }

    try {
      await Promise.all(
        selectedClientIds.map((clientId) =>
          addConnectionMutation.mutateAsync({
            clientId,
            data: {
              identity_provider_id: providerId,
              is_default: false,
              enabled: true,
              display_order: 0,
            },
          }),
        ),
      )
      showSuccess(
        `${selectedClientIds.length} client${selectedClientIds.length !== 1 ? "s" : ""} connected to ${providerName}`,
      )
      onOpenChange(false)
    } catch (error) {
      showError(error)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-2xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Connect a client</DialogTitle>
          <DialogDescription>
            Select existing clients to connect to {providerName}. Already connected clients are not shown.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label>Select Clients</Label>
              {availableClientIds.length > 0 && (
                <Button type="button" variant="ghost" size="sm" onClick={toggleAll} className="h-7 text-xs">
                  {selectedClientIds.length === availableClientIds.length ? "Deselect All" : "Select All"}
                </Button>
              )}
            </div>

            <div className="relative">
              <Search className="absolute left-2 top-2.5 size-4 text-muted-foreground" />
              <Input
                placeholder="Search clients..."
                value={searchQuery}
                onChange={(event) => setSearchQuery(event.target.value)}
                className="pl-8"
              />
            </div>
          </div>

          <div className="max-h-[400px] overflow-y-auto rounded-md border divide-y">
            {isLoading ? (
              <div className="p-8 text-center text-sm text-muted-foreground">Loading clients...</div>
            ) : availableClients.length === 0 ? (
              <div className="flex flex-col items-center gap-2 p-8 text-center text-sm text-muted-foreground">
                <AppWindow className="size-5" />
                {searchQuery ? "No clients found matching your search" : "All clients are already connected"}
              </div>
            ) : (
              availableClients.map((client) => (
                <SelectableOptionRow
                  key={client.client_id}
                  selected={selectedClientIds.includes(client.client_id)}
                  onToggle={() => toggleClient(client.client_id)}
                  title={client.display_name}
                  description={describeClient(client)}
                  disabled={isSubmitting}
                />
              ))
            )}
          </div>

          {selectedClientIds.length > 0 && (
            <p className="text-sm text-muted-foreground">
              {selectedClientIds.length} client{selectedClientIds.length !== 1 ? "s" : ""} selected
            </p>
          )}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={selectedClientIds.length === 0 || isSubmitting}>
            {isSubmitting ? (
              <>
                <Link2 className="mr-2 size-4 animate-spin" />
                Connecting...
              </>
            ) : (
              <>
                <Link2 className="mr-2 size-4" />
                Connect
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
