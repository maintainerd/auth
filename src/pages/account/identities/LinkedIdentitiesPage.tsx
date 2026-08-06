import { useEffect, useRef } from "react"
import { useSearchParams } from "react-router-dom"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Link2, Trash2, Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { SettingsCard } from "@/components/card"
import { ListingItemCard } from "@/components/details"
import AccountLayout from "@/components/layout/AccountLayout"
import { useToast } from "@/hooks/useToast"
import { resolvePublicAuthContext } from "@/utils/clientContext"
import { get, post, deleteRequest } from "@/services/api/client"
import { API_ENDPOINTS } from "@/services/api/config"
import { fetchOAuthConnections } from "@/services/api/oauth"
import type { OAuthConnection } from "@/services/api/oauth/types"
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

interface StartLinkResult {
  authorization_url: string
  state: string
}

/** Where the provider sends the user back. Must be a registered redirect URI. */
const LINK_RETURN_PATH = "/account/identities"

function linkRedirectURI(): string {
  return `${window.location.origin}${LINK_RETURN_PATH}`
}

export default function LinkedIdentitiesPage() {
  const queryClient = useQueryClient()
  const { showError, showSuccess } = useToast()
  const [searchParams, setSearchParams] = useSearchParams()
  const completedRef = useRef(false)

  // Same resolver the login page uses, so the provider list shown here is the
  // one this surface actually offers.
  const clientId = resolvePublicAuthContext().clientId ?? ""

  const { data: identities = [], isLoading, isError } = useQuery({
    queryKey: ["account", "identities"],
    queryFn: async () => {
      const res = await get<ApiResponse<LinkedIdentity[]>>(API_ENDPOINTS.AUTH.ACCOUNT_IDENTITIES)
      return res.data ?? []
    },
  })

  // The providers this tenant actually offers. Presenting the real list is the
  // point: a user cannot be expected to know a provider's internal identifier,
  // let alone obtain an id_token for it by hand.
  const { data: connections } = useQuery({
    queryKey: ["oauth", "connections", clientId],
    queryFn: () => fetchOAuthConnections(clientId),
    enabled: Boolean(clientId),
  })

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["account", "identities"] })

  const startMut = useMutation({
    mutationFn: async (providerIdentifier: string) => {
      const res = await post<ApiResponse<StartLinkResult>>(
        API_ENDPOINTS.AUTH.ACCOUNT_IDENTITIES_LINK_START,
        { provider_identifier: providerIdentifier, redirect_uri: linkRedirectURI() },
      )
      if (!res.data?.authorization_url) throw new Error("Could not start linking")
      return res.data
    },
    // Full navigation, not a router push: we are leaving for the provider.
    onSuccess: (data) => { window.location.assign(data.authorization_url) },
    onError: (err) => showError(err, "Could not start linking"),
  })

  const completeMut = useMutation({
    mutationFn: (params: { state: string; code: string }) =>
      post<ApiResponse<unknown>>(API_ENDPOINTS.AUTH.ACCOUNT_IDENTITIES_LINK_CALLBACK, {
        state: params.state,
        code: params.code,
        redirect_uri: linkRedirectURI(),
      }),
    onSuccess: () => { showSuccess("Account connected"); invalidate() },
    onError: (err) => showError(err, "Could not connect that account"),
  })

  const unlinkMut = useMutation({
    mutationFn: (uuid: string) => deleteRequest(`${API_ENDPOINTS.AUTH.ACCOUNT_IDENTITIES}/${uuid}`),
    onSuccess: () => { showSuccess("Account disconnected"); invalidate() },
    onError: (err) => showError(err, "Could not disconnect that account"),
  })

  // Returning from the provider. Strip code/state from the URL immediately so a
  // reload or a shared link cannot replay them, and guard against React strict
  // mode invoking this twice.
  const code = searchParams.get("code")
  const state = searchParams.get("state")
  const providerError = searchParams.get("error")

  useEffect(() => {
    if (completedRef.current) return
    if (providerError) {
      completedRef.current = true
      showError(new Error(searchParams.get("error_description") || providerError))
      setSearchParams({}, { replace: true })
      return
    }
    if (!code || !state) return
    completedRef.current = true
    setSearchParams({}, { replace: true })
    completeMut.mutate({ state, code })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [code, state, providerError])

  const linkedProviders = new Set(identities.map((i: LinkedIdentity) => i.provider))
  const available = (connections?.connections ?? []).filter(
    (c: OAuthConnection) => !linkedProviders.has(c.provider),
  )

  return (
    <AccountLayout title="Linked accounts">
      <SettingsCard
        title="Connected accounts"
        description="Sign in with these providers as well as your password."
        icon={Link2}
        contentClassName="space-y-3"
      >
        {isLoading && <p className="py-6 text-center text-sm text-muted-foreground">Loading…</p>}
        {isError && (
          <p className="py-6 text-center text-sm text-destructive">
            Could not load your connected accounts.
          </p>
        )}
        {!isLoading && !isError && identities.length === 0 && (
          <p className="py-6 text-center text-sm text-muted-foreground">
            You have not connected any accounts yet.
          </p>
        )}
        {identities.map((identity: LinkedIdentity) => (
          <ListingItemCard
            key={identity.identity_uuid}
            title={identity.identity_provider_name || identity.provider}
            action={
              <Button
                variant="ghost"
                size="sm"
                className="size-10 text-destructive hover:text-destructive sm:size-8"
                aria-label={`Disconnect ${identity.identity_provider_name || identity.provider}`}
                disabled={unlinkMut.isPending}
                onClick={() => unlinkMut.mutate(identity.identity_uuid)}
              >
                <Trash2 className="size-4" />
              </Button>
            }
          >
            <p className="break-all text-xs text-muted-foreground">
              {identity.sub ?? identity.identity_uuid?.slice(0, 8)}
            </p>
          </ListingItemCard>
        ))}
      </SettingsCard>

      <SettingsCard
        title="Add an account"
        description="You will be sent to the provider to sign in, then returned here."
        icon={Plus}
        contentClassName="space-y-2"
      >
        {available.length === 0 ? (
          <p className="py-6 text-center text-sm text-muted-foreground">
            {connections?.connections?.length
              ? "Every available provider is already connected."
              : "Your organization has not enabled any sign-in providers."}
          </p>
        ) : (
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            {available.map((connection: OAuthConnection) => (
              <Button
                key={connection.identifier}
                variant="outline"
                className="h-11 w-full justify-start gap-2"
                disabled={startMut.isPending || completeMut.isPending}
                onClick={() => startMut.mutate(connection.identifier)}
              >
                <Link2 className="size-4 shrink-0" />
                <span className="truncate">
                  {startMut.isPending && startMut.variables === connection.identifier
                    ? "Redirecting…"
                    : `Connect ${connection.display_name}`}
                </span>
              </Button>
            ))}
          </div>
        )}
      </SettingsCard>
    </AccountLayout>
  )
}
