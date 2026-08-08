import { AuthEventListing } from "./components/AuthEventListing"
import { PageHeader } from "@/components/layout"

export default function LogMonitoringPage({ standalone = true }: { standalone?: boolean }) {
  if (!standalone) return <AuthEventListing tableInCard />

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Sign-in Logs"
        description="Monitor authentication, authorization, and security events across your tenant. Filter by category, severity, result, or search by event type and IP."
      />
      <AuthEventListing tableInCard />
    </div>
  )
}
