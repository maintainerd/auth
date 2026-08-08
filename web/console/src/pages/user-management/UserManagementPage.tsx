import { Mail, Shield, Users } from "lucide-react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { PageHeader } from "@/components/layout"
import { UserListing } from "@/pages/users/components/UserListing"
import { RoleListing } from "@/pages/roles/components/RoleListing"
import { InvitationListing } from "@/pages/invitations/components/InvitationListing"

const TABS = [
  { value: "users", label: "Users", icon: Users },
  { value: "roles", label: "Roles", icon: Shield },
  { value: "invitations", label: "Invitations", icon: Mail },
] as const

export type UserManagementTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function UserManagementPage({
  defaultTab = "users",
}: {
  defaultTab?: UserManagementTab
}) {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const requestedTab = searchParams.get("tab")
  const activeTab: UserManagementTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as UserManagementTab
    : defaultTab

  const handleTabChange = (tab: string) => {
    navigate(`/user-management?tab=${tab}`)
  }

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="User Management"
        description="Manage users, roles, and invitations from one identity administration workspace."
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

        <TabsContent value="users">
          <UserListing tableInCard />
        </TabsContent>
        <TabsContent value="roles">
          <RoleListing tableInCard />
        </TabsContent>
        <TabsContent value="invitations">
          <InvitationListing tableInCard />
        </TabsContent>
      </DetailTabs>
    </div>
  )
}
