import { WebhookListing } from "./components/WebhookListing"
import { PageHeader } from "@/components/layout"

export default function WebhooksPage({ standalone = true }: { standalone?: boolean }) {
  if (!standalone) return <WebhookListing tableInCard />

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Webhooks"
        description="Register endpoints to receive real-time event notifications. Each delivery is signed so your service can verify it came from us."
      />
      <WebhookListing tableInCard />
    </div>
  )
}
