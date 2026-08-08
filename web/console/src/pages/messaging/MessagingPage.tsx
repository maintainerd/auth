import { useSearchParams } from "react-router-dom"
import { Mail, MessageSquare } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { PageHeader } from "@/components/layout"

import EmailDeliveryPage from "./email/EmailDeliveryPage"
import SMSDeliveryPage from "./sms/SMSDeliveryPage"

const TABS = [
  { value: "email", label: "Email Delivery", icon: Mail },
  { value: "sms", label: "SMS Delivery", icon: MessageSquare },
] as const

type MessagingTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function MessagingPage() {
  const [searchParams, setSearchParams] = useSearchParams()

  const requestedTab = searchParams.get("tab")
  const activeTab: MessagingTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as MessagingTab
    : "email"

  const handleTabChange = (tab: string) => setSearchParams({ tab })

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Messaging"
        description="Configure email and SMS delivery providers for verification codes, notifications, and security alerts."
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

        <TabsContent value="email">
          <EmailDeliveryPage standalone={false} />
        </TabsContent>
        <TabsContent value="sms">
          <SMSDeliveryPage standalone={false} />
        </TabsContent>
      </DetailTabs>
    </div>
  )
}
