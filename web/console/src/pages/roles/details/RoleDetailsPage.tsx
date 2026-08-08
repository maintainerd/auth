import { useParams, useNavigate, useSearchParams } from "react-router-dom"
import { Shield, Users } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { useRole } from "@/hooks/useRoles"
import { RoleHeader, RolePermissionsTab, RoleUsers } from "./components"

const TABS = [
  { value: "permissions", label: "Permissions", icon: Shield },
  { value: "users", label: "Users", icon: Users },
] as const

export default function RoleDetailsPage() {
  const { roleId } = useParams<{ roleId: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const activeTab = searchParams.get("tab") || "permissions"
  const handleTabChange = (tab: string) => setSearchParams({ tab })

  const { data: role, isLoading, isError } = useRole(roleId || "")

  return (
    <DetailLayout
      backLabel="Back to Roles"
      onBack={() => navigate(`/roles`)}
      isLoading={isLoading}
      isError={isError || !role}
      notFoundTitle="Role not found"
      notFoundDescription="The role you're looking for doesn't exist or may have been removed."
    >
      <RoleHeader role={role!} roleId={roleId!} />

      <DetailTabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList>
          {TABS.map(({ value, label, icon: Icon }) => (
            <TabsTrigger key={value} value={value} className="gap-2">
              <Icon className="size-4" />
              {label}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="permissions">
          <RolePermissionsTab roleId={roleId!} />
        </TabsContent>
        <TabsContent value="users">
          <RoleUsers roleId={roleId!} />
        </TabsContent>
      </DetailTabs>
    </DetailLayout>
  )
}
