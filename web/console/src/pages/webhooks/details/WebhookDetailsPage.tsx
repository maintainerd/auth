import { useParams, useNavigate, useSearchParams, useLocation } from "react-router-dom"
import { AppWindow, Radio, History } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { useWebhook } from "@/hooks/useWebhooks"
import { WebhookHeader, WebhookInformation, WebhookEvents } from "./components"
import { WebhookDeliveries } from "./components/WebhookDeliveries"
import { WEBHOOKS_LIST_URL } from "../webhookNavigation"

const TABS = [
  { value: "overview", label: "Overview", icon: AppWindow },
  { value: "events", label: "Events", icon: Radio },
  { value: "deliveries", label: "Deliveries", icon: History },
] as const

type WebhookDetailsTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function WebhookDetailsPage() {
  const { webhookId } = useParams<{ webhookId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()

  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? WEBHOOKS_LIST_URL
  const backLabel = navState?.backLabel ?? "Back to Webhooks"

  const requestedTab = searchParams.get("tab")
  const activeTab: WebhookDetailsTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as WebhookDetailsTab
    : "overview"

  const handleTabChange = (tab: string) => {
    setSearchParams({ tab })
  }

  const { data: webhook, isLoading, isError } = useWebhook(webhookId || "")

  return (
    <DetailLayout
      backLabel={backLabel}
      onBack={() => navigate(backTo)}
      isLoading={isLoading}
      isError={isError || !webhook}
      notFoundTitle="Webhook not found"
      notFoundDescription="The webhook you're looking for doesn't exist or may have been removed."
    >
      {webhook && (
        <>
          <WebhookHeader webhook={webhook} webhookId={webhookId!} afterDeleteTo={backTo} />

          <DetailTabs value={activeTab} onValueChange={handleTabChange}>
            <TabsList>
              {TABS.map(({ value, label, icon: Icon }) => (
                <TabsTrigger key={value} value={value} className="gap-2">
                  <Icon className="size-4" />
                  {label}
                </TabsTrigger>
              ))}
            </TabsList>

            <TabsContent value="overview">
              <WebhookInformation webhook={webhook} />
            </TabsContent>

            <TabsContent value="events">
              <WebhookEvents webhookId={webhookId!} />
            </TabsContent>

            <TabsContent value="deliveries">
              <WebhookDeliveries webhookId={webhookId!} />
            </TabsContent>
          </DetailTabs>
        </>
      )}
    </DetailLayout>
  )
}
