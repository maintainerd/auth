import { useSearchParams } from "react-router-dom"
import { Palette, Mail, MessageSquare } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { PageHeader } from "@/components/layout"

import BrandingTemplatesPage from "./templates"
import EmailTemplatesPage from "./email-templates/EmailTemplatesPage"
import SmsTemplatesPage from "./sms-templates/SmsTemplatesPage"

const TABS = [
  { value: "themes", label: "Themes", icon: Palette },
  { value: "email-templates", label: "Email Templates", icon: Mail },
  { value: "sms-templates", label: "SMS Templates", icon: MessageSquare },
] as const

type BrandingTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function BrandingPage() {
  const [searchParams, setSearchParams] = useSearchParams()

  const requestedTab = searchParams.get("tab")
  const activeTab: BrandingTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as BrandingTab
    : "themes"

  const handleTabChange = (tab: string) => setSearchParams({ tab })

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Branding"
        description="Configure themes, hosted-auth login templates, previews, and messaging templates for the auth experience."
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

        <TabsContent value="themes">
          <BrandingTemplatesPage standalone={false} />
        </TabsContent>
        <TabsContent value="email-templates">
          <EmailTemplatesPage standalone={false} />
        </TabsContent>
        <TabsContent value="sms-templates">
          <SmsTemplatesPage standalone={false} />
        </TabsContent>
      </DetailTabs>
    </div>
  )
}
