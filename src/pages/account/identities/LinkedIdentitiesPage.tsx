import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Link2, Trash2, Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { SettingsCard } from "@/components/card"
import { FormInputField } from "@/components/form"
import { ListingItemCard } from "@/components/details"
import AccountLayout from "@/components/layout/AccountLayout"
import { useToast } from "@/hooks/useToast"
import { get, post, deleteRequest } from "@/services/api/client"
import { API_ENDPOINTS } from "@/services/api/config"
import type { ApiResponse } from "@/services/api/types"

interface LinkedIdentity {
  identity_uuid: string
  provider: string
  sub?: string
  identity_provider_name?: string
  is_default?: boolean
  linked_at?: string
  created_at?: string
}

// Mirrors idp.LinkIdentityRequestDTO. The field is provider_identifier, not
// provider — sending `provider` left it empty server-side and the handler
// rejected every link attempt with 400 "provider_identifier and external_token
// are required".
interface LinkIdentityRequest {
  provider_identifier: string
  external_token: string
}

export default function LinkedIdentitiesPage() {
  const queryClient = useQueryClient()
  const { showError, showSuccess } = useToast()
  const [showLinkForm, setShowLinkForm] = useState(false)
  const [provider, setProvider] = useState("")
  const [externalToken, setExternalToken] = useState("")

  const { data, isLoading } = useQuery({
    queryKey: ["account", "identities"],
    queryFn: async () => {
      const res = await get<ApiResponse<LinkedIdentity[]>>(API_ENDPOINTS.AUTH.ACCOUNT_IDENTITIES)
      return res.data ?? []
    },
  })

  const unlinkMut = useMutation({
    mutationFn: (uuid: string) => deleteRequest(`${API_ENDPOINTS.AUTH.ACCOUNT_IDENTITIES}/${uuid}`),
    onSuccess: () => {
      showSuccess("Identity unlinked")
      queryClient.invalidateQueries({ queryKey: ["account", "identities"] })
    },
    onError: (e) => showError(e, "Failed to unlink"),
  })

  const linkMut = useMutation({
    mutationFn: (req: LinkIdentityRequest) => post(API_ENDPOINTS.AUTH.ACCOUNT_IDENTITIES_LINK, req),
    onSuccess: () => {
      showSuccess("Identity linked")
      setShowLinkForm(false)
      setProvider("")
      setExternalToken("")
      queryClient.invalidateQueries({ queryKey: ["account", "identities"] })
    },
    onError: (e) => showError(e, "Failed to link"),
  })

  const identities = Array.isArray(data) ? data : []

  return (
    <AccountLayout title="Linked Accounts">
      <div className="grid gap-6">
        <SettingsCard
          title="External identities"
          description="External provider accounts connected to your profile."
          icon={Link2}
          action={(
            <Button
              variant="outline"
              size="sm"
              className="w-full gap-1.5 sm:w-auto"
              onClick={() => setShowLinkForm(!showLinkForm)}
              disabled={linkMut.isPending}
            >
              <Plus className="size-4" /> Link identity
            </Button>
          )}
        >
          <div className="space-y-4">
            {showLinkForm && (
              <div data-md-listing-nested className="space-y-3 rounded-md border p-3">
                <FormInputField
                  label="Identity provider"
                  value={provider}
                  onChange={(e) => setProvider(e.target.value)}
                  placeholder="e.g. google, github, apple"
                />
                <FormInputField
                  label="External token"
                  value={externalToken}
                  onChange={(e) => setExternalToken(e.target.value)}
                  placeholder="Paste the external provider ID token"
                />
                <div className="flex justify-end">
                  <Button onClick={() => linkMut.mutate({ provider_identifier: provider, external_token: externalToken })} disabled={linkMut.isPending || !provider || !externalToken}>
                    {linkMut.isPending ? "Linking…" : "Link identity"}
                  </Button>
                </div>
              </div>
            )}

            {isLoading ? (
              <p className="text-sm text-muted-foreground">Loading identities…</p>
            ) : identities.length === 0 ? (
              <div className="py-12 text-center text-muted-foreground">
                <Link2 className="mx-auto mb-3 size-12 opacity-30" />
                <p>No external identities linked yet.</p>
              </div>
            ) : (
              <div className="space-y-2">
                {identities.map((identity) => {
                  const isBuiltIn = identity.is_default || identity.provider === 'maintainerd'
                  return (
                    <ListingItemCard
                      key={identity.identity_uuid}
                      icon={Link2}
                      className="items-center p-3"
                      contentClassName="items-center"
                      action={isBuiltIn ? (
                        <span className="rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary">
                          Primary sign-in
                        </span>
                      ) : (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => unlinkMut.mutate(identity.identity_uuid)}
                          disabled={unlinkMut.isPending}
                          aria-label={`Unlink ${identity.provider ?? identity.identity_provider_name ?? 'identity'}`}
                        >
                          <Trash2 className="size-3 text-destructive" />
                        </Button>
                      )}
                    >
                      <p className="font-medium capitalize">{identity.provider ?? identity.identity_provider_name ?? "Unknown"}</p>
                      <p className="text-xs text-muted-foreground">{identity.sub ?? identity.identity_uuid?.slice(0, 8)}</p>
                      {identity.created_at && (
                        <p className="text-xs text-muted-foreground">
                          Linked {new Date(identity.created_at).toLocaleDateString()}
                        </p>
                      )}
                    </ListingItemCard>
                  )
                })}
              </div>
            )}
          </div>
        </SettingsCard>
      </div>
    </AccountLayout>
  )
}
