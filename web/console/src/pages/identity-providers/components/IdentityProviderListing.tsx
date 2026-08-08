import { useNavigate } from "react-router-dom"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { identityProviderColumns } from "./IdentityProviderColumns"
import { useIdentityProvidersList } from "@/hooks/useIdentityProviders"

const DEFAULT_SORT: SortingState = [{ id: "name", desc: false }]
const SEARCH_FIELDS = ["search"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive"] },
  { key: "provider_type", label: "Type", options: ["system", "enterprise", "social"] },
  { key: "provider", label: "Provider", options: ["maintainerd", "saml", "cognito", "auth0", "google", "facebook", "github", "gitlab", "microsoft", "linkedin", "twitter"] },
]

export function IdentityProviderListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const navigate = useNavigate()

  return (
    <ResourceListing
      tableInCard={tableInCard}
      columns={identityProviderColumns}
      defaultSort={DEFAULT_SORT}
      searchFields={SEARCH_FIELDS}
      searchPlaceholder="Search identity providers by name, display name, or identifier..."
      useData={useIdentityProvidersList}
      filterGroups={FILTER_GROUPS}
      onRowClick={(provider) => navigate(`/providers/identity/${provider.identity_provider_id}`)}
      onCreate={() => navigate(`/providers/identity/create`)}
      createLabel="New Provider"
      emptyTitle="No identity providers yet"
      emptyDescription="Add an identity provider to let users authenticate through the built-in system or an external service like Cognito, Auth0, or Google."
    />
  )
}
