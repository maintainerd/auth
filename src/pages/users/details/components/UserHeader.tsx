import { useState } from "react"
  import {
  Edit,
  Trash2,
  MoreVertical,
  CheckCircle2,
  Phone,
  Mail,
  Building2,
  CalendarDays,
  Minus,
  ShieldOff,
  Play,
  Pause,
  Ban,
  KeyRound,
  Eraser,
  LockOpen,
} from "lucide-react"
import { useNavigate } from "react-router-dom"
import { format } from "date-fns"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { DeleteConfirmationDialog, ConfirmationDialog } from "@/components/dialog"
import { DetailHeaderCard, StatusBadge, type DetailAttribute } from "@/components/details"
import {
  useDeleteUser,
  useVerifyUserEmail,
  useVerifyUserPhone,
  useResetUserMfa,
  useUpdateUserStatus,
  useForcePasswordChange,
  useCreateErasureRequest,
  useUnlockUser,
} from "@/hooks/useUsers"
import { useToast } from "@/hooks/useToast"
import type { User, UserStatus } from "@/services/api/users/types"

interface UserHeaderProps {
  user: User
  userId: string
}

/** A subtle verified / not-verified inline indicator (no pastel blocks). */
function VerifiedMark({ verified }: { verified: boolean }) {
  return verified ? (
    <span className="inline-flex items-center gap-1 text-xs font-medium text-emerald-600">
      <CheckCircle2 className="size-3.5" />
      Verified
    </span>
  ) : (
    <span className="inline-flex items-center gap-1 text-xs font-medium text-muted-foreground">
      <Minus className="size-3.5" />
      Not verified
    </span>
  )
}

export function UserHeader({ user, userId }: UserHeaderProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const deleteUserMutation = useDeleteUser()
  const verifyEmailMutation = useVerifyUserEmail()
  const verifyPhoneMutation = useVerifyUserPhone()
  const resetMfaMutation = useResetUserMfa()
  const updateStatusMutation = useUpdateUserStatus()
  const forcePasswordChangeMutation = useForcePasswordChange()
  const erasureRequestMutation = useCreateErasureRequest()
  const unlockMutation = useUnlockUser()
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [showVerifyEmailDialog, setShowVerifyEmailDialog] = useState(false)
  const [showVerifyPhoneDialog, setShowVerifyPhoneDialog] = useState(false)
  const [showResetMfaDialog, setShowResetMfaDialog] = useState(false)
  const [showForcePasswordDialog, setShowForcePasswordDialog] = useState(false)
  const [showUnlockDialog, setShowUnlockDialog] = useState(false)
  const [showErasureDialog, setShowErasureDialog] = useState(false)
  const [statusAction, setStatusAction] = useState<{ status: UserStatus; title: string; description: string } | null>(null)

  const runAction = async (mutate: () => Promise<unknown>, successMessage: string) => {
    try {
      await mutate()
      showSuccess(successMessage)
    } catch (error) {
      showError(error)
    }
  }

  const handleDelete = async () => {
    try {
      await deleteUserMutation.mutateAsync(userId)
      showSuccess("User deleted successfully")
      navigate(`/users`)
    } catch (error) {
      showError(error)
    }
  }

  const initials = (user.fullname || user.username).slice(0, 2).toUpperCase()

  const attributes: DetailAttribute[] = [
    {
      icon: Mail,
      label: "Email",
      value: (
        <div className="flex flex-col gap-0.5">
          <span className="break-all">{user.email}</span>
          <VerifiedMark verified={user.is_email_verified} />
        </div>
      ),
    },
    {
      icon: Phone,
      label: "Phone",
      value: user.phone ? (
        <div className="flex flex-col gap-0.5">
          <span>{user.phone}</span>
          <VerifiedMark verified={user.is_phone_verified} />
        </div>
      ) : (
        <span className="text-muted-foreground">—</span>
      ),
    },
    ...(user.tenant
      ? [
          {
            icon: Building2,
            label: "Tenant",
            value: (
              <div className="flex flex-col gap-0.5">
                <span className="font-medium">{user.tenant.display_name}</span>
                <span className="font-mono text-xs text-muted-foreground">{user.tenant.name}</span>
              </div>
            ),
          } satisfies DetailAttribute,
        ]
      : []),
    { icon: CalendarDays, label: "Created", value: format(new Date(user.created_at), "PP") },
    { icon: CalendarDays, label: "Last updated", value: format(new Date(user.updated_at), "PP") },
  ]

  return (
    <>
      <DetailHeaderCard
        leading={
          <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-blue-500 to-indigo-600 text-lg font-semibold text-white shadow-sm shadow-blue-500/20">
            {initials}
          </div>
        }
        title={user.fullname || user.username}
        badge={<StatusBadge status={user.status} />}
        subtitle={`@${user.username}`}
        attributes={attributes}
        actions={
          <>
            <Button
              data-md-details-edit-button
              variant="outline"
              size="sm"
              className="h-9 gap-2"
              onClick={() =>
                navigate(`/users/${userId}/edit`, {
                  state: { from: `/users/${userId}`, backLabel: "Back to User Details" },
                })
              }
            >
              <Edit className="size-4" />
              Edit
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button data-md-details-menu-button variant="outline" size="sm" className="h-9 w-9 p-0">
                  <span className="sr-only">Open actions</span>
                  <MoreVertical className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {user.status !== "active" && (
                  <DropdownMenuItem onClick={() => setStatusAction({ status: "active", title: "Activate User", description: "Are you sure you want to activate this user? They will be able to sign in and access the system." })}>
                    <Play className="mr-2 size-4" />
                    Activate User
                  </DropdownMenuItem>
                )}
                {user.status === "active" && (
                  <>
                    <DropdownMenuItem onClick={() => setStatusAction({ status: "inactive", title: "Deactivate User", description: "Are you sure you want to deactivate this user? They will no longer be able to sign in." })}>
                      <Pause className="mr-2 size-4" />
                      Deactivate User
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setStatusAction({ status: "suspended", title: "Suspend User", description: "Are you sure you want to suspend this user? This is typically used for security concerns or policy violations. They will be immediately logged out and cannot sign in." })}>
                      <Ban className="mr-2 size-4" />
                      Suspend User
                    </DropdownMenuItem>
                  </>
                )}
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => setShowVerifyEmailDialog(true)}>
                  <CheckCircle2 className="mr-2 size-4" />
                  Mark Email as Verified
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setShowVerifyPhoneDialog(true)}>
                  <Phone className="mr-2 size-4" />
                  Mark Phone as Verified
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setShowResetMfaDialog(true)}>
                  <ShieldOff className="mr-2 size-4" />
                  Reset MFA
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setShowForcePasswordDialog(true)}>
                  <KeyRound className="mr-2 size-4" />
                  Force Password Change
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setShowUnlockDialog(true)}>
                  <LockOpen className="mr-2 size-4" />
                  Unlock Account
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => setShowErasureDialog(true)}
                  className="text-destructive focus:text-destructive"
                >
                  <Eraser className="mr-2 size-4" />
                  Erase User Data
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => setShowDeleteDialog(true)}
                  className="text-destructive focus:text-destructive"
                >
                  <Trash2 className="mr-2 size-4" />
                  Delete User
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </>
        }
      />

      <DeleteConfirmationDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        onConfirm={handleDelete}
        title="Delete User"
        description="This permanently deletes the user account and all associated data. This action cannot be undone."
        itemName={user.fullname || user.username}
        isDeleting={deleteUserMutation.isPending}
      />

      <ConfirmationDialog
        open={showVerifyEmailDialog}
        onOpenChange={setShowVerifyEmailDialog}
        onConfirm={() => runAction(() => verifyEmailMutation.mutateAsync(userId), "Email verified successfully")}
        title="Verify Email"
        description="Are you sure you want to mark this user's email as verified?"
        isLoading={verifyEmailMutation.isPending}
      />

      <ConfirmationDialog
        open={showVerifyPhoneDialog}
        onOpenChange={setShowVerifyPhoneDialog}
        onConfirm={() => runAction(() => verifyPhoneMutation.mutateAsync(userId), "Phone verified successfully")}
        title="Verify Phone"
        description="Are you sure you want to mark this user's phone as verified?"
        isLoading={verifyPhoneMutation.isPending}
      />

      <ConfirmationDialog
        open={showResetMfaDialog}
        onOpenChange={setShowResetMfaDialog}
        onConfirm={() => runAction(() => resetMfaMutation.mutateAsync(userId), "MFA reset successfully")}
        title="Reset MFA"
        description="This removes all of the user's multi-factor authentication enrollments. They'll be prompted to set up MFA again on next sign-in. Continue?"
        confirmText="Reset MFA"
        isLoading={resetMfaMutation.isPending}
      />

      <ConfirmationDialog
        open={showForcePasswordDialog}
        onOpenChange={setShowForcePasswordDialog}
        onConfirm={() => runAction(() => forcePasswordChangeMutation.mutateAsync(userId), "Password change forced successfully")}
        title="Force Password Change"
        description="This forces the user to change their password on their next login. Continue?"
        confirmText="Force Change"
        isLoading={forcePasswordChangeMutation.isPending}
      />

      <ConfirmationDialog
        open={showUnlockDialog}
        onOpenChange={setShowUnlockDialog}
        onConfirm={() => runAction(() => unlockMutation.mutateAsync(userId), "Account unlocked")}
        title="Unlock Account"
        description="This clears the user's failed-login lockout so they can sign in again immediately. Continue?"
        confirmText="Unlock"
        isLoading={unlockMutation.isPending}
      />

      <ConfirmationDialog
        open={!!statusAction}
        onOpenChange={(open) => { if (!open) setStatusAction(null) }}
        onConfirm={() => {
          if (!statusAction) return
          runAction(
            () => updateStatusMutation.mutateAsync({ userId, data: { status: statusAction.status } }),
            `User status updated to ${statusAction.status}`,
          ).then(() => setStatusAction(null))
        }}
        title={statusAction?.title ?? ""}
        description={statusAction?.description ?? ""}
        // Suspend/Deactivate are disruptive (immediate logout, blocks sign-in) →
        // red confirm. Activate is restorative → normal confirm.
        variant={statusAction && statusAction.status !== "active" ? "destructive" : "default"}
        confirmText={statusAction?.status === "active" ? "Activate" : statusAction?.status === "suspended" ? "Suspend" : "Deactivate"}
        isLoading={updateStatusMutation.isPending}
      />

      <DeleteConfirmationDialog
        open={showErasureDialog}
        onOpenChange={setShowErasureDialog}
        onConfirm={() =>
          runAction(
            () => erasureRequestMutation.mutateAsync(userId),
            "Data erasure request submitted",
          ).then(() => setShowErasureDialog(false))
        }
        title="Erase User Data"
        description="This schedules irreversible anonymization of all personal data for this user in 30 days. The account cannot be restored once erasure begins."
        itemName={user.fullname || user.username}
        confirmLabel="Schedule Erasure"
        isDeleting={erasureRequestMutation.isPending}
      />

    </>
  )
}
