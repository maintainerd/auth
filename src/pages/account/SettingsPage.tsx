import { useNavigate, useSearchParams } from "react-router-dom"
import { SlidersHorizontal, ShieldCheck } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailsContainer } from "@/components/container"
import { FormPageHeader } from "@/components/header"
import { PreferencesForm } from "./components/PreferencesForm"
import { SecuritySessions } from "./components/SecuritySessions"
import { AccountActions } from "./components/AccountActions"
import { MfaSettingsCard } from "./components/MfaSettingsCard"

const TABS = [
  { value: "preferences", label: "Preferences", icon: SlidersHorizontal },
  { value: "security", label: "Security", icon: ShieldCheck },
] as const

export default function SettingsPage() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = searchParams.get("tab") === "security" ? "security" : "preferences"

  return (
    <DetailsContainer>
      <div className="flex flex-col gap-6">
        <FormPageHeader
          backUrl={`/dashboard`}
          backLabel="Back"
          title="Settings"
          description="Manage your preferences and account security."
        />

        <DetailTabs value={tab} onValueChange={(v) => setSearchParams(v === "security" ? { tab: "security" } : {})}>
          <TabsList>
            {TABS.map(({ value, label, icon: Icon }) => (
              <TabsTrigger key={value} value={value} className="gap-2">
                <Icon className="size-4" />
                {label}
              </TabsTrigger>
            ))}
          </TabsList>

          <TabsContent value="preferences">
            <PreferencesForm />
          </TabsContent>

          <TabsContent value="security" className="space-y-6">
            <MfaSettingsCard onManage={() => navigate(`/account/mfa?from=settings`)} />
            <SecuritySessions />
            <AccountActions />
          </TabsContent>
        </DetailTabs>
      </div>
    </DetailsContainer>
  )
}
