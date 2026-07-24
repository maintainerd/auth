import { useParams, useNavigate, useSearchParams } from "react-router-dom"
import { Settings, Shield } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { useRegistrationFlow } from "@/hooks/useRegistrationFlows"
import {
  RegistrationFlowHeader,
  RegistrationFlowLink,
  RegistrationFlowConfig,
  RegistrationFlowRoles,
} from "./components"

const TABS = [
  { value: "config", label: "Overview", icon: Settings },
  { value: "roles", label: "Roles", icon: Shield },
] as const

type RegistrationFlowTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function RegistrationFlowDetailsPage() {
  const { registrationFlowId } = useParams<{ registrationFlowId: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  // Validate the requested tab instead of trusting ?tab= verbatim, which would
  // otherwise render a details page with every tab panel hidden.
  const requestedTab = searchParams.get("tab")
  const activeTab: RegistrationFlowTab = TAB_VALUES.has(requestedTab || "")
    ? (requestedTab as RegistrationFlowTab)
    : "config"

  const handleTabChange = (tab: string) => setSearchParams({ tab })

  const { data: registrationFlow, isLoading, isError } = useRegistrationFlow(registrationFlowId || "")

  return (
    <DetailLayout
      backLabel="Back to Registration Flows"
      onBack={() => navigate(`/registration-flows`)}
      isLoading={isLoading}
      isError={isError || !registrationFlow}
      notFoundTitle="Registration flow not found"
      notFoundDescription="The registration flow you're looking for doesn't exist or may have been removed."
    >
      <RegistrationFlowHeader registrationFlow={registrationFlow!} registrationFlowId={registrationFlowId!} />

      <DetailTabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList>
          {TABS.map(({ value, label, icon: Icon }) => (
            <TabsTrigger key={value} value={value} className="gap-2">
              <Icon className="size-4" />
              {label}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="config">
          {/* The link is the primary artifact of a flow, so it leads the default tab. */}
          <div className="space-y-4">
            <RegistrationFlowLink flow={registrationFlow!} />
            <RegistrationFlowConfig flow={registrationFlow!} />
          </div>
        </TabsContent>
        <TabsContent value="roles">
          <RegistrationFlowRoles registrationFlowId={registrationFlowId!} />
        </TabsContent>
      </DetailTabs>
    </DetailLayout>
  )
}
