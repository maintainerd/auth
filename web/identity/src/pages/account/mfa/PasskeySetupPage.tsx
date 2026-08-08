import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Key, Plus, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { FormInputField } from "@/components/form"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { ListingItemCard, ListingItemIcon } from "@/components/details/ListingItemCard"
import { useToast } from "@/hooks/useToast"
import {
  beginWebAuthnRegistration, finishWebAuthnRegistration, deleteWebAuthnCredential, fetchMFAStatus,
  type MFAWebAuthnKey,
} from "@/services/api/mfa"
import { base64urlToBuffer, serializeCredential } from "@/lib/webauthn"
import { MFASetupShell, ConfirmRemoveDialog, MfaSetupSkeleton } from "./MfaShell"

export default function PasskeySetupPage() {
  const { showSuccess, showError } = useToast()
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({ queryKey: ["mfa", "status"], queryFn: fetchMFAStatus, retry: false })
  const keys = data?.webauthn_keys ?? []

  const [name, setName] = useState("")
  const [pendingDelete, setPendingDelete] = useState<MFAWebAuthnKey | null>(null)

  const registerMutation = useMutation({
    mutationFn: async () => {
      const result = await beginWebAuthnRegistration()
      const pk = result.publicKey
      const publicKey: PublicKeyCredentialCreationOptions = {
        rp: pk.rp,
        user: { id: base64urlToBuffer(pk.user.id), name: pk.user.name, displayName: pk.user.displayName },
        challenge: base64urlToBuffer(pk.challenge),
        pubKeyCredParams: pk.pubKeyCredParams,
        timeout: pk.timeout,
        attestation: pk.attestation,
        authenticatorSelection: pk.authenticatorSelection,
      }
      try {
        const credential = await navigator.credentials.create({ publicKey })
        if (!credential) throw new Error("Passkey registration was cancelled")
        return finishWebAuthnRegistration(name.trim() || "Security Key", serializeCredential(credential as PublicKeyCredential))
      } catch (err) {
        if (err instanceof DOMException && err.name === "NotAllowedError") throw new Error("Passkey registration was cancelled")
        throw err
      }
    },
    onSuccess: (res) => {
      showSuccess(`Passkey "${res.name}" registered`)
      setName("")
      queryClient.invalidateQueries({ queryKey: ["mfa", "status"] })
    },
    onError: (e) => showError(e),
  })

  const deleteMutation = useMutation({
    mutationFn: (uuid: string) => deleteWebAuthnCredential(uuid),
    onSuccess: () => {
      showSuccess("Passkey removed")
      setPendingDelete(null)
      queryClient.invalidateQueries({ queryKey: ["mfa", "status"] })
    },
    onError: (e) => { showError(e); setPendingDelete(null) },
  })

  if (isLoading) {
    return <MFASetupShell><MfaSetupSkeleton /></MFASetupShell>
  }

  return (
    <MFASetupShell>
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <ListingItemIcon className="size-10">
                <Key className="size-5 text-muted-foreground" />
              </ListingItemIcon>
              <div>
                <CardTitle className="text-base">Register a passkey</CardTitle>
                <CardDescription>Use Face ID, Touch ID, Windows Hello, or a physical security key.</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <FormInputField
              id="key-name"
              label="Passkey name (optional)"
              placeholder="My YubiKey"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <div className="flex justify-end">
              <Button onClick={() => registerMutation.mutate()} disabled={registerMutation.isPending}>
                <Plus className="mr-2 size-4" />
                {registerMutation.isPending ? "Waiting for device…" : "Register passkey"}
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Your passkeys</CardTitle>
            <CardDescription>
              {keys.length > 0 ? `${keys.length} registered passkey${keys.length !== 1 ? "s" : ""}.` : "You haven't registered any passkeys yet."}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {keys.length === 0 ? (
              <div className="flex flex-col items-center gap-2 py-8 text-center text-sm text-muted-foreground">
                <Key className="size-8 text-muted-foreground/40" />
                <p>Register a passkey above to get started.</p>
              </div>
            ) : (
              <div className="space-y-2">
                {keys.map((key) => (
                  <ListingItemCard
                    key={key.credential_uuid}
                    icon={Key}
                    className="items-center p-3"
                    contentClassName="items-center"
                    action={(
                      <Button
                        variant="ghost"
                        size="icon"
                        className="size-10 text-destructive hover:text-destructive sm:size-8"
                        onClick={() => setPendingDelete(key)}
                        title="Remove passkey"
                        aria-label={`Remove passkey ${key.name || key.credential_uuid}`}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    )}
                  >
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium">{key.name || "Passkey"}</p>
                      <p className="text-xs capitalize text-muted-foreground">{key.transport || "Unknown transport"}</p>
                    </div>
                  </ListingItemCard>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <ConfirmRemoveDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => { if (!open) setPendingDelete(null) }}
        onConfirm={() => { if (pendingDelete) deleteMutation.mutate(pendingDelete.credential_uuid) }}
        title="Remove passkey"
        description={pendingDelete ? `"${pendingDelete.name || "This passkey"}" will no longer be able to sign in to your account.` : ""}
        isLoading={deleteMutation.isPending}
      />
    </MFASetupShell>
  )
}
