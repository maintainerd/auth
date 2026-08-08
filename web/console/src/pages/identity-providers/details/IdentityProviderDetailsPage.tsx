import { useParams, useNavigate, useSearchParams } from "react-router-dom"
import { AppWindow, KeyRound, Settings2 } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { useIdentityProvider } from "@/hooks/useIdentityProviders"
import {
  IdentityProviderHeader,
  IdentityProviderConnectionTab,
  IdentityProviderConfigurationTab,
  IdentityProviderClients,
} from "./components"

const TABS = [
  { value: "connection", label: "Connection", icon: KeyRound },
  { value: "configuration", label: "Configuration", icon: Settings2 },
  { value: "connected-clients", label: "Connected Clients", icon: AppWindow },
] as const

const TAB_VALUES = new Set(TABS.map((tab) => tab.value))
const LEGACY_TAB_MAP: Record<string, (typeof TABS)[number]["value"]> = {
  information: "connection",
  metadata: "configuration",
  clients: "connected-clients",
}

export default function IdentityProviderDetailsPage() {
  const { providerId } = useParams<{ providerId: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const requestedTab = searchParams.get("tab") || ""
  const activeTab = TAB_VALUES.has(requestedTab as (typeof TABS)[number]["value"])
    ? requestedTab
    : LEGACY_TAB_MAP[requestedTab] ?? "connection"
  const handleTabChange = (tab: string) => setSearchParams({ tab })

  const { data: provider, isLoading, isError } = useIdentityProvider(providerId || "")
  const hasProvider = Boolean(provider)

  return (
    <DetailLayout
      backLabel="Back to Identity Providers"
      onBack={() => navigate(`/providers/identity`)}
      isLoading={isLoading}
      isError={isError || (!isLoading && !hasProvider)}
      notFoundTitle="Identity Provider not found"
      notFoundDescription="The identity provider you're looking for doesn't exist or may have been removed."
    >
      {provider && (
        <>
          <IdentityProviderHeader provider={provider} providerId={providerId!} />

          <DetailTabs value={activeTab} onValueChange={handleTabChange}>
            <TabsList>
              {TABS.map(({ value, label, icon: Icon }) => (
                <TabsTrigger key={value} value={value} className="gap-2">
                  <Icon className="size-4" />
                  {label}
                </TabsTrigger>
              ))}
            </TabsList>

            <TabsContent value="connection">
              <IdentityProviderConnectionTab provider={provider} />
            </TabsContent>

            <TabsContent value="configuration">
              <IdentityProviderConfigurationTab provider={provider} />
            </TabsContent>

            <TabsContent value="connected-clients">
              <IdentityProviderClients
                providerId={providerId!}
                providerName={provider.display_name}
              />
            </TabsContent>
          </DetailTabs>
        </>
      )}
    </DetailLayout>
  )
}
