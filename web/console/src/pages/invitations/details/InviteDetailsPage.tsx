import { useState } from "react"
import { useParams, useNavigate } from "react-router-dom"
import { Mail, Send, Ban, MoreVertical, Workflow, CalendarClock, CalendarDays, CheckCircle2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { ConfirmationDialog } from "@/components/dialog"
import { DetailLayout, DetailHeaderCard, StatusBadge, type DetailAttribute } from "@/components/details"
import { useInvite, useResendInvite, useRevokeInvite } from "@/hooks/useInvites"
import { useToast } from "@/hooks/useToast"
import { safeFormat } from "@/lib/formatDate"

export default function InviteDetailsPage() {
  const { inviteId } = useParams<{ inviteId: string }>()
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const resendMutation = useResendInvite()
  const revokeMutation = useRevokeInvite()
  const [showRevoke, setShowRevoke] = useState(false)

  const { data: invite, isLoading, isError } = useInvite(inviteId)
  const isPending = invite?.status === "pending"

  const resend = async () => {
    try {
      await resendMutation.mutateAsync(inviteId!)
      showSuccess("Invitation resent — a fresh link is on its way")
    } catch (error) {
      showError(error)
    }
  }

  const revoke = async () => {
    try {
      await revokeMutation.mutateAsync(inviteId!)
      showSuccess("Invitation revoked")
    } catch (error) {
      showError(error)
    }
  }

  const attributes: DetailAttribute[] = invite
    ? [
        {
          icon: Workflow,
          label: "Registration flow",
          value: invite.registration_flow_name ?? (invite.registration_flow_id ? "—" : "Default registration"),
        },
        { icon: CalendarDays, label: "Invited", value: safeFormat(invite.created_at, "PP") || "—" },
        { icon: CalendarClock, label: "Expires", value: safeFormat(invite.expires_at, "PP") || "—" },
        { icon: CheckCircle2, label: "Accepted", value: safeFormat(invite.used_at, "PP") || "—" },
      ]
    : []

  return (
    <DetailLayout
      backLabel="Back to Invitations"
      onBack={() => navigate(`/invites`)}
      isLoading={isLoading}
      isError={isError || !invite}
      notFoundTitle="Invitation not found"
      notFoundDescription="The invitation you're looking for doesn't exist or may have been removed."
    >
      {invite && (
        <>
          <DetailHeaderCard
            leading={
              <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
                <Mail className="size-6" />
              </div>
            }
            title={invite.invited_email}
            badge={<StatusBadge status={invite.status} />}
            subtitle={
              invite.registration_flow_name
                ? `Onboards via the "${invite.registration_flow_name}" registration flow`
                : "Onboards via default registration"
            }
            attributes={attributes}
            actions={
              isPending ? (
                <>
                  <Button
                    data-md-details-edit-button
                    variant="outline"
                    size="sm"
                    className="h-9 gap-2"
                    onClick={resend}
                    disabled={resendMutation.isPending}
                  >
                    <Send className="size-4" />
                    Resend
                  </Button>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button data-md-details-menu-button variant="outline" size="sm" className="h-9 w-9 p-0">
                        <span className="sr-only">Open actions</span>
                        <MoreVertical className="size-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        onClick={() => setShowRevoke(true)}
                        className="text-destructive focus:text-destructive"
                      >
                        <Ban className="mr-2 size-4" />
                        Revoke Invitation
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </>
              ) : undefined
            }
          />

          <ConfirmationDialog
            open={showRevoke}
            onOpenChange={setShowRevoke}
            onConfirm={revoke}
            title="Revoke Invitation"
            description={`Revoke the invitation to ${invite.invited_email}? Their sign-up link will stop working.`}
            confirmText="Revoke"
            variant="destructive"
            isLoading={revokeMutation.isPending}
          />
        </>
      )}
    </DetailLayout>
  )
}
