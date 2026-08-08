import { KeyRound, UserPlus } from "lucide-react"
import { Link, useNavigate, useSearchParams } from "react-router-dom"
import { DetailTabs } from "@/components/details/DetailTabs"
import { PageHeader } from "@/components/layout"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { IdentityProviderListing } from "@/pages/identity-providers/components/IdentityProviderListing"
import { RegistrationFlowListing } from "@/pages/registration-flows/components/RegistrationFlowListing"

const TABS = [
  { value: "identity-providers", label: "Identity Providers", icon: KeyRound },
  { value: "registration-flows", label: "Registration Flows", icon: UserPlus },
] as const

export type AuthenticationTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function AuthenticationPage({
  defaultTab = "identity-providers",
}: {
  defaultTab?: AuthenticationTab
}) {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const requestedTab = searchParams.get("tab")
  const activeTab: AuthenticationTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as AuthenticationTab
    : defaultTab

  const handleTabChange = (tab: string) => {
    navigate(`/authentication?tab=${tab}`)
  }

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Authentication"
        description="Manage identity providers and registration flows for sign-in and user onboarding."
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

        <TabsContent value="identity-providers">
          <IdentityProviderListing tableInCard />
        </TabsContent>
        <TabsContent value="registration-flows">
          <div className="flex flex-col gap-4">
            <p className="text-sm text-muted-foreground">
              These are per-application flows. The tenant-wide registration defaults they override live in{" "}
              <Link to="/security?tab=registration" className="font-medium underline underline-offset-4">
                Security - Registration
              </Link>
              .
            </p>
            <RegistrationFlowListing tableInCard />
          </div>
        </TabsContent>
      </DetailTabs>
    </div>
  )
}
