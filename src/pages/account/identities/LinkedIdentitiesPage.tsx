import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Link2, Trash2, Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
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

interface LinkIdentityRequest {
  provider: string
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
      <div className="mb-6 flex items-center justify-between">
        <p className="text-muted-foreground">External provider accounts connected to your profile</p>
        <Button variant="outline" size="sm" onClick={() => setShowLinkForm(!showLinkForm)} disabled={linkMut.isPending}>
          <Plus className="h-4 w-4 mr-1" /> Link Identity
        </Button>
      </div>

      {showLinkForm && (
        <Card className="mb-6 border-primary/40">
          <CardHeader>
            <CardTitle className="text-base">Link an External Identity</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <Label>Identity Provider</Label>
              <Input
                value={provider}
                onChange={(e) => setProvider(e.target.value)}
                placeholder="e.g. google, github, apple"
              />
            </div>
            <div>
              <Label>External Token (ID Token)</Label>
              <Input
                value={externalToken}
                onChange={(e) => setExternalToken(e.target.value)}
                placeholder="Paste your external provider's ID token"
              />
            </div>
            <Button onClick={() => linkMut.mutate({ provider, external_token: externalToken })} disabled={linkMut.isPending || !provider || !externalToken}>
              {linkMut.isPending ? "Linking..." : "Link Identity"}
            </Button>
          </CardContent>
        </Card>
      )}

      {isLoading ? (
        <p className="text-muted-foreground">Loading...</p>
      ) : identities.length === 0 ? (
        <div className="text-center py-12 text-muted-foreground">
          <Link2 className="h-12 w-12 mx-auto mb-3 opacity-30" />
          <p>No external identities linked yet.</p>
          <p className="text-sm">Link a provider to sign in with it.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {identities.map((id) => {
            const isBuiltIn = id.is_default || id.provider === 'maintainerd'
            return (
              <Card key={id.identity_uuid}>
                <CardContent className="flex items-center justify-between py-3">
                  <div>
                    <p className="font-medium capitalize">{id.provider ?? id.identity_provider_name ?? "Unknown"}</p>
                    <p className="text-xs text-muted-foreground">{id.sub ?? id.identity_uuid?.slice(0, 8)}</p>
                    {id.created_at && <p className="text-xs text-muted-foreground">Linked: {new Date(id.created_at).toLocaleDateString()}</p>}
                  </div>
                  {isBuiltIn ? (
                    <span className="rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary">
                      Primary sign-in
                    </span>
                  ) : (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => unlinkMut.mutate(id.identity_uuid)}
                      disabled={unlinkMut.isPending}
                    >
                      <Trash2 className="h-3 w-3 text-destructive" />
                    </Button>
                  )}
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}
    </AccountLayout>
  )
}
