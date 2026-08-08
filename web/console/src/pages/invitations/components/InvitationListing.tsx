import { useNavigate } from "react-router-dom"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { useInvitesList } from "@/hooks/useInvites"
import { inviteColumns } from "./InviteColumns"

const DEFAULT_SORT: SortingState = [{ id: "Invited", desc: true }]
const SEARCH_FIELDS = ["invited_email"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["pending", "accepted", "expired", "revoked"] },
]

export function InvitationListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const navigate = useNavigate()

  return (
    <ResourceListing
      tableInCard={tableInCard}
      columns={inviteColumns}
      defaultSort={DEFAULT_SORT}
      searchFields={SEARCH_FIELDS}
      searchPlaceholder="Search invitations by email..."
      useData={useInvitesList}
      filterGroups={FILTER_GROUPS}
      onRowClick={(invite) => navigate(`/invites/${invite.invite_id}`)}
      onCreate={() => navigate(`/invites/create`)}
      createLabel="Invite User"
      emptyTitle="No invitations yet"
      emptyDescription="Invite users to register by sending a signed email link with their onboarding flow."
    />
  )
}
