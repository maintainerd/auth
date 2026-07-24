import { useEffect, useMemo, useState } from "react"
import { Building2, Link2, Search } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { PROVIDER_LABELS } from "@/components/provider-config"
import { SelectableOptionRow } from "@/components/inputs"
import { useAddClientIdentityProvider } from "@/hooks/useClients"
import { useIdentityProviders } from "@/hooks/useIdentityProviders"
import { useToast } from "@/hooks/useToast"
import type { IdentityProvider } from "@/services/api/identity-providers/types"

interface ConnectProviderDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  clientId: string
  clientName: string
  existingProviderIds: string[]
}

export function ConnectProviderDialog({
  open,
  onOpenChange,
  clientId,
  clientName,
  existingProviderIds,
}: ConnectProviderDialogProps) {
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedProviderIds, setSelectedProviderIds] = useState<string[]>([])
  const { showSuccess, showError } = useToast()
  const addConnectionMutation = useAddClientIdentityProvider()

  const { data: providersData, isLoading } = useIdentityProviders({
    page: 1,
    limit: 100,
    sort_by: "display_name",
    sort_order: "asc",
  })

  useEffect(() => {
    if (!open) {
      setSearchQuery("")
      setSelectedProviderIds([])
    }
  }, [open])

  const availableProviders = useMemo(() => {
    const existing = new Set(existingProviderIds)
    const q = searchQuery.trim().toLowerCase()

    return (providersData?.rows ?? [])
      .filter((provider) => !existing.has(provider.identity_provider_id))
      .filter((provider) => {
        if (!q) return true
        return [
          provider.display_name,
          provider.name,
          provider.identifier,
          PROVIDER_LABELS[provider.provider] ?? provider.provider,
        ].some((value) => value.toLowerCase().includes(q))
      })
  }, [existingProviderIds, providersData?.rows, searchQuery])

  const availableProviderIds = availableProviders.map((provider) => provider.identity_provider_id)
  const isSubmitting = addConnectionMutation.isPending

  const toggleProvider = (providerId: string) => {
    setSelectedProviderIds((prev) =>
      prev.includes(providerId) ? prev.filter((id) => id !== providerId) : [...prev, providerId],
    )
  }

  const toggleAll = () => {
    setSelectedProviderIds((prev) =>
      prev.length === availableProviderIds.length ? [] : availableProviderIds,
    )
  }

  const describeProvider = (provider: IdentityProvider) => {
    const parts = [
      provider.name,
      PROVIDER_LABELS[provider.provider] ?? provider.provider,
      provider.provider_type,
    ].filter(Boolean)

    return parts.join(" · ")
  }

  const handleSubmit = async () => {
    if (selectedProviderIds.length === 0) {
      showError("Please select at least one identity provider")
      return
    }

    try {
      await Promise.all(
        selectedProviderIds.map((providerId, index) =>
          addConnectionMutation.mutateAsync({
            clientId,
            data: {
              identity_provider_id: providerId,
              is_default: existingProviderIds.length === 0 && index === 0,
              enabled: true,
              display_order: existingProviderIds.length + index,
            },
          }),
        ),
      )
      showSuccess(
        `${selectedProviderIds.length} identity provider${selectedProviderIds.length !== 1 ? "s" : ""} connected to ${clientName}`,
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
          <DialogTitle>Connect identity providers</DialogTitle>
          <DialogDescription>
            Select identity providers this client can use for sign-in. Already connected providers are not shown.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label>Select Identity Providers</Label>
              {availableProviderIds.length > 0 && (
                <Button type="button" variant="ghost" size="sm" onClick={toggleAll} className="h-7 text-xs">
                  {selectedProviderIds.length === availableProviderIds.length ? "Deselect All" : "Select All"}
                </Button>
              )}
            </div>

            <div className="relative">
              <Search className="absolute left-2 top-2.5 size-4 text-muted-foreground" />
              <Input
                placeholder="Search identity providers..."
                value={searchQuery}
                onChange={(event) => setSearchQuery(event.target.value)}
                className="pl-8"
              />
            </div>
          </div>

          <div className="max-h-[400px] overflow-y-auto rounded-md border divide-y">
            {isLoading ? (
              <div className="p-8 text-center text-sm text-muted-foreground">Loading identity providers...</div>
            ) : availableProviders.length === 0 ? (
              <div className="flex flex-col items-center gap-2 p-8 text-center text-sm text-muted-foreground">
                <Building2 className="size-5" />
                {searchQuery ? "No identity providers found matching your search" : "All identity providers are already connected"}
              </div>
            ) : (
              availableProviders.map((provider) => (
                <SelectableOptionRow
                  key={provider.identity_provider_id}
                  selected={selectedProviderIds.includes(provider.identity_provider_id)}
                  onToggle={() => toggleProvider(provider.identity_provider_id)}
                  title={provider.display_name}
                  description={describeProvider(provider)}
                  disabled={isSubmitting}
                />
              ))
            )}
          </div>

          {selectedProviderIds.length > 0 && (
            <p className="text-sm text-muted-foreground">
              {selectedProviderIds.length} identity provider{selectedProviderIds.length !== 1 ? "s" : ""} selected
            </p>
          )}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={selectedProviderIds.length === 0 || isSubmitting}>
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
