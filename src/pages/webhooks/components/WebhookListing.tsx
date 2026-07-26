import { useNavigate } from "react-router-dom"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { webhookColumns } from "./WebhookColumns"
import { useWebhooksList } from "@/hooks/useWebhooks"
import { WEBHOOKS_BACK_STATE } from "../webhookNavigation"

const DEFAULT_SORT: SortingState = [{ id: "created_at", desc: false }]
const SEARCH_FIELDS = ["url"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive", "quarantined"] },
]

export function WebhookListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const navigate = useNavigate()

  return (
    <ResourceListing
      tableInCard={tableInCard}
      columns={webhookColumns}
      defaultSort={DEFAULT_SORT}
      searchFields={SEARCH_FIELDS}
      searchPlaceholder="Search webhooks by URL..."
      useData={useWebhooksList}
      filterGroups={FILTER_GROUPS}
      onRowClick={(webhook) =>
        navigate(`/webhooks/${webhook.webhook_endpoint_id}`, { state: WEBHOOKS_BACK_STATE })
      }
      onCreate={() => navigate(`/webhooks/create`, { state: WEBHOOKS_BACK_STATE })}
      createLabel="New Webhook"
      emptyTitle="No webhooks yet"
      emptyDescription="Register your first webhook to receive event notifications whenever something happens in your tenant."
    />
  )
}
