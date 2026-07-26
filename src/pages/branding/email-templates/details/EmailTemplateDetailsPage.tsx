import { useLocation, useNavigate, useParams, useSearchParams } from "react-router-dom"
import { FileText, Eye } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { useEmailTemplate } from "@/hooks/useEmailTemplates"
import { EmailTemplateHeader, EmailTemplateContent, EmailTemplatePreview } from "./components"

const TABS = [
  { value: "content", label: "Content", icon: FileText },
  { value: "preview", label: "Preview", icon: Eye },
] as const

const TAB_VALUES = new Set(TABS.map((tab) => tab.value))

export default function EmailTemplateDetailsPage() {
  const { templateId } = useParams<{ templateId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()

  const { data: template, isLoading, isError } = useEmailTemplate(templateId || '')
  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/branding?tab=email-templates`
  const backLabel = navState?.backLabel ?? "Back to Email Templates"
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
      notFoundTitle="Email template not found"
      notFoundDescription="The email template you're looking for doesn't exist or may have been removed."
    >
      {template && (
        <>
          <EmailTemplateHeader template={template} templateId={templateId!} />

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
              <EmailTemplateContent template={template} />
            </TabsContent>

            <TabsContent value="preview">
              <EmailTemplatePreview template={template} />
            </TabsContent>
          </DetailTabs>
        </>
      )}
    </DetailLayout>
  )
}
