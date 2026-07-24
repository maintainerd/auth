import { useState } from "react"
import { Copy, KeyRound, RotateCcw, ShieldAlert } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { InformationCard } from "@/components/card"
import { safeFormat } from "@/lib/formatDate"
import { ConfirmationDialog } from "@/components/dialog"
import { useToast } from "@/hooks/useToast"
import { CopyableCode } from "@/components/inputs"
import { authMethodRequiresSecret } from "@/lib/validations"
import { useRotateClientSecret } from "@/hooks/useClients"
import type { ClientResponse } from "@/services/api/clients/types"

interface ClientCredentialsProps {
  client: ClientResponse
}

export function ClientCredentials({ client }: ClientCredentialsProps) {
  const { showSuccess, showError } = useToast()
  const rotateSecretMutation = useRotateClientSecret()
  const [showRotateDialog, setShowRotateDialog] = useState(false)
  const [rotatedSecret, setRotatedSecret] = useState<string | null>(null)
  const [previousSecretExpiresAt, setPreviousSecretExpiresAt] = useState<string | null>(null)

  // Keyed off the auth method rather than the client type: a traditional client
  // using private_key_jwt has no shared secret, so a "Rotate Secret" button there
  // would mint a credential nothing authenticates with. Falls back to client type
  // when the method is absent from the response.
  const authMethod = client.token_endpoint_auth_method
  const shouldUseSecret = authMethod
    ? authMethodRequiresSecret(authMethod)
    : client.client_type === "traditional" || client.client_type === "m2m"
  // Rotating a system client's secret breaks the console or hosted login UI, which
  // are configured with the seeded value — the backend refuses it outright.
  const canRotate = shouldUseSecret && !client.is_system
  const credentialModel = shouldUseSecret ? "Confidential client" : "Public client"

  const copy = async (value: string, label: string) => {
    await navigator.clipboard.writeText(value)
    showSuccess(`${label} copied to clipboard`)
  }

  const rotateSecret = async () => {
    try {
      const result = await rotateSecretMutation.mutateAsync({
        clientId: client.client_id,
        data: { grace_period_hours: 24 },
      })
      setRotatedSecret(result.client_secret)
      setPreviousSecretExpiresAt(result.previous_secret_expires_at ?? null)
      showSuccess("Client secret rotated successfully")
    } catch (error) {
      showError(error)
    }
  }

  return (
    <>
      <InformationCard
        title="Client Credentials"
        description="Secrets are never retrievable after creation. Rotate the secret to issue a new plaintext value."
        icon={KeyRound}
        action={
          canRotate ? (
            <Button size="sm" className="h-9 gap-2" onClick={() => setShowRotateDialog(true)}>
              <RotateCcw className="size-4" />
              Rotate Secret
            </Button>
          ) : undefined
        }
      >
        <div className="space-y-5">
          <div className="grid gap-4 md:grid-cols-2">
            {client.identifier && (
              <div className="space-y-1">
                <p className="text-sm font-medium text-muted-foreground">
                  Client ID <span className="font-normal">(use this in your application)</span>
                </p>
                <CopyableCode value={client.identifier} label="Client ID" variant="block" />
              </div>
            )}
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">Client UUID</p>
              <div className="flex min-w-0 items-center gap-2">
                <code className="min-w-0 flex-1 break-all rounded bg-muted px-2 py-1.5 font-mono text-sm">
                  {client.client_id}
                </code>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8 w-8 shrink-0 p-0"
                  onClick={() => copy(client.client_id, "Client UUID")}
                >
                  <span className="sr-only">Copy client UUID</span>
                  <Copy className="size-4" />
                </Button>
              </div>
            </div>
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">Credential Model</p>
              <p className="rounded bg-muted px-2 py-1.5 text-sm">{credentialModel}</p>
            </div>
          </div>

          {shouldUseSecret ? (
            <Alert>
              <ShieldAlert className="size-4" />
              <AlertDescription>
                The client secret is hashed at rest and cannot be viewed. Use rotation when you need a new secret, then store the returned value immediately.
              </AlertDescription>
            </Alert>
          ) : (
            <Alert>
              <ShieldAlert className="size-4" />
              <AlertDescription>
                This public client type does not use a client secret. Configure PKCE-capable authorization flows for this application.
              </AlertDescription>
            </Alert>
          )}

          {rotatedSecret && (
            <div className="space-y-2 rounded-md border border-amber-200 bg-amber-50 p-4 text-amber-950">
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <p className="text-sm font-semibold">New Client Secret</p>
                  {previousSecretExpiresAt && (
                    <p className="text-xs">Previous secret expires {safeFormat(previousSecretExpiresAt, "PPpp")}.</p>
                  )}
                </div>
                <Button variant="outline" size="sm" className="h-8 gap-2 bg-background" onClick={() => copy(rotatedSecret, "Client secret")}>
                  <Copy className="size-4" />
                  Copy
                </Button>
              </div>
              <code className="block rounded-md bg-background px-3 py-2 font-mono text-sm break-all">
                {rotatedSecret}
              </code>
            </div>
          )}
        </div>
      </InformationCard>

      <ConfirmationDialog
        open={showRotateDialog}
        onOpenChange={setShowRotateDialog}
        onConfirm={rotateSecret}
        title="Rotate Client Secret"
        description="This will issue a new client secret. The current secret remains valid for 24 hours, then expires."
        confirmText="Rotate Secret"
        isLoading={rotateSecretMutation.isPending}
        showWarning
        warningMessage="The new secret is shown only once. Store it before leaving this page."
      />
    </>
  )
}
