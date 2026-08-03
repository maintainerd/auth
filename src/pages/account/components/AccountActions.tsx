import { useState } from "react"
import { logout } from "@/services/api/auth"
import { Mail, AtSign, Download, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { SettingsCard } from "@/components/card"
import { FormEmailField } from "@/components/inputs"
import { useToast } from "@/hooks/useToast"
import { changeEmail, verifyEmailChange, changeUsername, exportAccountData, deleteAccount } from "@/services/api/account"
import { useMutation } from "@tanstack/react-query"
import { StepUpDialog } from "@/components/stepup/StepUpDialog"
import { SettingsActionRow } from "./SettingsActionRow"

export function AccountActions() {
  const [emailOpen, setEmailOpen] = useState(false)
  const [usernameOpen, setUsernameOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const { showSuccess, showError } = useToast()

  const exportMutation = useMutation({
    mutationFn: exportAccountData,
    onSuccess: (data) => {
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" })
      const a = document.createElement("a")
      a.href = URL.createObjectURL(blob)
      a.download = "account-data.json"
      a.click()
      showSuccess("Your data export has downloaded")
    },
    onError: (e) => showError(e),
  })

  return (
    <>
      <SettingsCard
        title="Account"
        description="Manage your sign-in identity and personal data."
        icon={AtSign}
        contentClassName="p-0"
      >
        <div className="divide-y">
          <SettingsActionRow
            icon={Mail}
            title="Email address"
            description="Update the email you sign in with."
            actionLabel="Change"
            onAction={() => setEmailOpen(true)}
          />
          <SettingsActionRow
            icon={AtSign}
            title="Username"
            description="Update your account username."
            actionLabel="Change"
            onAction={() => setUsernameOpen(true)}
          />
          <SettingsActionRow
            icon={Download}
            title="Export your data"
            description="Download a copy of your account data."
            actionLabel={exportMutation.isPending ? "Preparing…" : "Download"}
            onAction={() => exportMutation.mutate()}
            disabled={exportMutation.isPending}
          />
        </div>
      </SettingsCard>

      <SettingsCard
        title="Danger zone"
        description="Irreversible actions for your account."
        icon={Trash2}
        className="border-destructive/30"
        contentClassName="p-0"
      >
        <SettingsActionRow
          icon={Trash2}
          title="Delete account"
          description="Permanently delete your account and all associated data."
          action={<Button variant="destructive" onClick={() => setDeleteOpen(true)}>Delete account</Button>}
        />
      </SettingsCard>

      <ChangeEmailDialog open={emailOpen} onOpenChange={setEmailOpen} />
      <ChangeUsernameDialog open={usernameOpen} onOpenChange={setUsernameOpen} />
      <DeleteAccountDialog open={deleteOpen} onOpenChange={setDeleteOpen} />
    </>
  )
}

function ChangeEmailDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const { showSuccess, showError } = useToast()
  const [step, setStep] = useState<"request" | "verify">("request")
  const [newEmail, setNewEmail] = useState("")
  const [password, setPassword] = useState("")
  const [otp, setOtp] = useState("")

  const close = () => { onOpenChange(false); setTimeout(() => { setStep("request"); setNewEmail(""); setPassword(""); setOtp("") }, 200) }

  const requestMutation = useMutation({
    mutationFn: () => changeEmail(newEmail.trim(), password),
    onSuccess: () => { setStep("verify"); showSuccess("Verification code sent to your new email") },
    onError: (e) => showError(e),
  })
  const verifyMutation = useMutation({
    mutationFn: () => verifyEmailChange(otp.trim()),
    onSuccess: () => { showSuccess("Email address updated"); close() },
    onError: (e) => showError(e),
  })

  return (
    <Dialog open={open} onOpenChange={(o) => (o ? onOpenChange(true) : close())}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Change email address</DialogTitle>
          <DialogDescription>
            {step === "request" ? "Enter your new email and current password. We'll send a code to verify it." : `Enter the 6-digit code we sent to ${newEmail}.`}
          </DialogDescription>
        </DialogHeader>
        {step === "request" ? (
          <div className="space-y-4">
            <FormEmailField id="ce-email" label="New email" value={newEmail} onChange={(e) => setNewEmail(e.target.value)} placeholder="you@example.com" />
            <div className="space-y-2"><Label htmlFor="ce-pw">Current password</Label><Input id="ce-pw" type="password" value={password} onChange={(e) => setPassword(e.target.value)} /></div>
          </div>
        ) : (
          <div className="space-y-2">
            <Label htmlFor="ce-otp">Verification code</Label>
            <Input id="ce-otp" inputMode="numeric" autoComplete="one-time-code" className="font-mono tracking-[0.4em] text-center" maxLength={6} value={otp} onChange={(e) => setOtp(e.target.value.replace(/\D/g, "").slice(0, 6))} placeholder="000000" />
          </div>
        )}
        <DialogFooter>
          <Button variant="ghost" onClick={close}>Cancel</Button>
          {step === "request" ? (
            <Button onClick={() => requestMutation.mutate()} disabled={!newEmail.trim() || !password || requestMutation.isPending}>{requestMutation.isPending ? "Sending…" : "Send code"}</Button>
          ) : (
            <Button onClick={() => verifyMutation.mutate()} disabled={otp.length !== 6 || verifyMutation.isPending}>{verifyMutation.isPending ? "Verifying…" : "Verify & update"}</Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function ChangeUsernameDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const { showSuccess, showError } = useToast()
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")

  const close = () => { onOpenChange(false); setTimeout(() => { setUsername(""); setPassword("") }, 200) }

  const mutation = useMutation({
    mutationFn: () => changeUsername(username.trim(), password),
    onSuccess: () => { showSuccess("Username updated"); close() },
    onError: (e) => showError(e),
  })

  return (
    <Dialog open={open} onOpenChange={(o) => (o ? onOpenChange(true) : close())}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Change username</DialogTitle>
          <DialogDescription>Enter your new username and confirm with your current password.</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2"><Label htmlFor="cu-name">New username</Label><Input id="cu-name" value={username} onChange={(e) => setUsername(e.target.value)} /></div>
          <div className="space-y-2"><Label htmlFor="cu-pw">Current password</Label><Input id="cu-pw" type="password" value={password} onChange={(e) => setPassword(e.target.value)} /></div>
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={close}>Cancel</Button>
          <Button onClick={() => mutation.mutate()} disabled={!username.trim() || !password || mutation.isPending}>{mutation.isPending ? "Saving…" : "Update username"}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function DeleteAccountDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const { showError } = useToast()
  const [password, setPassword] = useState("")
  const [confirm, setConfirm] = useState("")
  const [stepUpOpen, setStepUpOpen] = useState(false)

  const close = () => { onOpenChange(false); setTimeout(() => { setPassword(""); setConfirm("") }, 200) }

  const mutation = useMutation({
    mutationFn: (token: string) => deleteAccount(password, token),
    onSuccess: () => {
      // Account is gone; end the console session locally and land on login.
      logout()
    },
    onError: (e) => showError(e),
  })

  return (
    <>
      <Dialog open={open} onOpenChange={(o) => (o ? onOpenChange(true) : close())}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-destructive">Delete account</DialogTitle>
            <DialogDescription>
              This permanently deletes your account and all associated data. This cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2"><Label htmlFor="da-pw">Current password</Label><Input id="da-pw" type="password" value={password} onChange={(e) => setPassword(e.target.value)} /></div>
            <div className="space-y-2">
              <Label htmlFor="da-confirm">Type <span className="font-mono font-semibold">DELETE</span> to confirm</Label>
              <Input id="da-confirm" value={confirm} onChange={(e) => setConfirm(e.target.value)} placeholder="DELETE" />
            </div>
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={close}>Cancel</Button>
            <Button variant="destructive" disabled={!password || confirm !== "DELETE"} onClick={() => { onOpenChange(false); setStepUpOpen(true) }}>Continue</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <StepUpDialog
        open={stepUpOpen}
        onOpenChange={setStepUpOpen}
        onVerified={(token) => mutation.mutate(token)}
        title="Verify to delete your account"
        description="Confirm your identity with a second factor to permanently delete your account."
      />
    </>
  )
}
