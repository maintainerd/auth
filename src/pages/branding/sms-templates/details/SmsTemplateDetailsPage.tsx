import { useLocation, useNavigate, useParams, useSearchParams } from "react-router-dom"
import { FileText } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { useSmsTemplate } from "@/hooks/useSmsTemplates"
import { SmsTemplateHeader, SmsTemplateContent } from "./components"

const TABS = [
  { value: "content", label: "Content", icon: FileText },
] as const

const TAB_VALUES = new Set(TABS.map((tab) => tab.value))

export default function SmsTemplateDetailsPage() {
  const { templateId } = useParams<{ templateId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()
  const { data: template, isLoading, isError } = useSmsTemplate(templateId || '')
  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/branding?tab=sms-templates`
  const backLabel = navState?.backLabel ?? "Back to SMS Templates"
  const requestedTab = searchParams.get("tab") || ""
  const activeTab = TAB_VALUES.has(requestedTab as (typeof TABS)[number]["value"])
    ? requestedTab
    : "content"
  const handleTabChange = (tab: string) => setSearchParams({ tab })

  return (
    <DetailLayout
      backLabel={backLabel}
      onBack={() => navigate(backTo)}
      isLoading={isLoading}
      isError={isError || !template}
      notFoundTitle="SMS template not found"
      notFoundDescription="The SMS template you're looking for doesn't exist or may have been removed."
    >
      {template && (
        <>
          <SmsTemplateHeader template={template} templateId={templateId!} />

          <DetailTabs value={activeTab} onValueChange={handleTabChange}>
            <TabsList>
              {TABS.map(({ value, label, icon: Icon }) => (
                <TabsTrigger key={value} value={value} className="gap-2">
                  <Icon className="size-4" />
                  {label}
                </TabsTrigger>
              ))}
            </TabsList>
            <TabsContent value="content">
              <SmsTemplateContent template={template} />
            </TabsContent>
          </DetailTabs>
        </>
      )}
    </DetailLayout>
  )
}
