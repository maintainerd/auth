import { useParams, useNavigate, useSearchParams } from "react-router-dom"
import { Shield, IdCard, KeyRound, Braces, Monitor, Activity, Smartphone, FileText } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { useUser } from "@/hooks/useUsers"
import {
  UserHeader,
  UserMetadata,
  UserRoles,
  UserIdentities,
  UserProfiles,
  UserActivity,
  UserSessions,
  UserMFA,
  UserConsents,
  UserTrustedDevices,
} from "./components"

const TABS = [
  { value: "profiles", label: "Profiles", icon: IdCard },
  { value: "roles", label: "Roles", icon: Shield },
  { value: "identities", label: "Identities", icon: KeyRound },
  { value: "mfa", label: "MFA", icon: Smartphone },
  { value: "sessions", label: "Sessions", icon: Monitor },
  { value: "activity", label: "Activity", icon: Activity },
  { value: "consents", label: "Consents", icon: FileText },
  { value: "devices", label: "Trusted Devices", icon: Smartphone },
  { value: "metadata", label: "Metadata", icon: Braces },
] as const

export default function UserDetailsPage() {
  const { userId } = useParams<{ userId: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const activeTab = searchParams.get("tab") || "profiles"
  const handleTabChange = (tab: string) => setSearchParams({ tab })

  const { data: user, isLoading, isError } = useUser(userId || "")

  return (
    <DetailLayout
      backLabel="Back to Users"
      onBack={() => navigate(`/users`)}
      isLoading={isLoading}
      isError={isError || !user}
      notFoundTitle="User not found"
      notFoundDescription="The user you're looking for doesn't exist or may have been removed."
    >
      <UserHeader user={user!} userId={userId!} />

      <DetailTabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList>
          {TABS.map(({ value, label, icon: Icon }) => (
            <TabsTrigger key={value} value={value} className="gap-2">
              <Icon className="size-4" />
              {label}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="profiles">
          <UserProfiles userId={userId!} />
        </TabsContent>
        <TabsContent value="roles">
          <UserRoles userId={userId!} />
        </TabsContent>
        <TabsContent value="identities">
          <UserIdentities userId={userId!} />
        </TabsContent>
        <TabsContent value="mfa">
          <UserMFA userId={userId!} />
        </TabsContent>
        <TabsContent value="sessions">
          <UserSessions userId={userId!} />
        </TabsContent>
        <TabsContent value="activity">
          <UserActivity userId={userId!} />
        </TabsContent>
        <TabsContent value="consents">
          <UserConsents userId={userId!} />
        </TabsContent>
        <TabsContent value="devices">
          <UserTrustedDevices userId={userId!} />
        </TabsContent>
        <TabsContent value="metadata">
          <UserMetadata user={user!} />
        </TabsContent>
      </DetailTabs>
    </DetailLayout>
  )
}
