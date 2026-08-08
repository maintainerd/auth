import { useSearchParams } from "react-router-dom"
import { Webhook, Radio, Info } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { PageHeader } from "@/components/layout"

import WebhooksPage from "../webhooks/WebhooksPage"
import EventRoutesPage from "../event-routes/EventRoutesPage"
import EventTypesPage from "../event-types"

const TABS = [
  { value: "webhooks", label: "Webhooks", icon: Webhook },
  { value: "routes", label: "Event Routes", icon: Radio },
  { value: "types", label: "Event Types", icon: Info },
] as const

type EventsTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function EventsAndWebhooksPage() {
  const [searchParams, setSearchParams] = useSearchParams()

  const requestedTab = searchParams.get("tab")
  const activeTab: EventsTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as EventsTab
    : "webhooks"

  const handleTabChange = (tab: string) => setSearchParams({ tab })

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Events &amp; Webhooks"
        description="Manage webhook endpoints, event routing to message brokers, and the event type catalog."
      />

      <DetailTabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList>
          {TABS.map(({ value, label, icon: Icon }) => (
            <TabsTrigger key={value} value={value} className="gap-2">
              <Icon className="size-4" />
              {label}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="webhooks">
          <WebhooksPage standalone={false} />
        </TabsContent>
        <TabsContent value="routes">
          <EventRoutesPage standalone={false} />
        </TabsContent>
        <TabsContent value="types">
          <EventTypesPage standalone={false} />
        </TabsContent>
      </DetailTabs>
    </div>
  )
}
