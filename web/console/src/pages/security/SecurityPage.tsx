import { useSearchParams } from "react-router-dom"
import {
  Shield,
  ShieldAlert,
  ShieldCheck,
  Ban,
  Clock,
  Key,
  UserPlus,
  Smartphone,
} from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { PageHeader } from "@/components/layout"

// View pages render their own skeleton/error/data states.
import PasswordPolicyPage from "./password-policies/PasswordPolicyPage"
import MfaViewPage from "./mfa/MfaViewPage"
import LockoutViewPage from "./lockout/LockoutViewPage"
import SessionViewPage from "./session-management/SessionViewPage"
import TokenViewPage from "./token/TokenViewPage"
import RegistrationViewPage from "./registration/RegistrationViewPage"
import ThreatViewPage from "./threat-detection/ThreatViewPage"
import IpRestrictionsPage from "./ip-restrictions/IpRestrictionsPage"

const TABS = [
  { value: "password", label: "Password Policy", icon: Key },
  { value: "mfa", label: "Multi-Factor Auth", icon: Smartphone },
  { value: "lockout", label: "Account Lockout", icon: ShieldAlert },
  { value: "sessions", label: "Sessions", icon: Clock },
  { value: "tokens", label: "Tokens", icon: ShieldCheck },
  { value: "registration", label: "Registration", icon: UserPlus },
  { value: "threat", label: "Threat Protection", icon: Shield },
  { value: "ip-restrictions", label: "IP Restrictions", icon: Ban },
] as const

type SecurityTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function SecurityPage() {
  const [searchParams, setSearchParams] = useSearchParams()

  const requestedTab = searchParams.get("tab")
  const activeTab: SecurityTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as SecurityTab
    : "password"

  const handleTabChange = (tab: string) => setSearchParams({ tab })

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Security"
        description="Manage authentication policies, session settings, threat protection, and IP access controls."
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

        <TabsContent value="password">
          <PasswordPolicyPage standalone={false} />
        </TabsContent>
        <TabsContent value="mfa">
          <MfaViewPage standalone={false} />
        </TabsContent>
        <TabsContent value="lockout">
          <LockoutViewPage standalone={false} />
        </TabsContent>
        <TabsContent value="sessions">
          <SessionViewPage standalone={false} />
        </TabsContent>
        <TabsContent value="tokens">
          <TokenViewPage standalone={false} />
        </TabsContent>
        <TabsContent value="registration">
          <RegistrationViewPage standalone={false} />
        </TabsContent>
        <TabsContent value="threat">
          <ThreatViewPage standalone={false} />
        </TabsContent>
        <TabsContent value="ip-restrictions">
          <IpRestrictionsPage standalone={false} />
        </TabsContent>
      </DetailTabs>
    </div>
  )
}
