import { useNavigate } from "react-router-dom"
import { Eye, Send, Ban } from "lucide-react"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useResendInvite, useRevokeInvite } from "@/hooks/useInvites"
import { useToast } from "@/hooks/useToast"
import type { Invite } from "@/services/api/invites/types"

interface InviteActionsProps {
  invite: Invite
}

export function InviteActions({ invite }: InviteActionsProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const resendMutation = useResendInvite()
  const revokeMutation = useRevokeInvite()

  const items: RowActionItem[] = [
    {
      key: "view",
      label: "View Details",
      icon: Eye,
      onSelect: () => navigate(`/invites/${invite.invite_id}`),
    },
  ]

  // Resend / revoke only make sense while the invitation is still pending.
  if (invite.status === "pending") {
    items.push(
      {
        key: "resend",
        label: "Resend Invitation",
        icon: Send,
        onSelect: async () => {
          try {
            await resendMutation.mutateAsync(invite.invite_id)
            showSuccess("Invitation resent — a fresh link is on its way")
          } catch (error) {
            showError(error)
          }
        },
      },
      {
        key: "revoke",
        label: "Revoke Invitation",
        icon: Ban,
        destructive: true,
        separatorBefore: true,
        onSelect: async () => {
          try {
            await revokeMutation.mutateAsync(invite.invite_id)
            showSuccess("Invitation revoked")
          } catch (error) {
            showError(error)
          }
        },
        confirm: {
          title: "Revoke Invitation",
          description: `Revoke the invitation to ${invite.invited_email}? Their sign-up link will stop working.`,
          confirmText: "Revoke",
        },
      },
    )
  }

  return <RowActions items={items} />
}
