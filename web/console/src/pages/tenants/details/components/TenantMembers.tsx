import { useParams } from "react-router-dom"
import { MembersSettings } from "@/pages/settings/components/MembersSettings"

interface TenantMembersProps {
  isSystemTenant?: boolean
}

export function TenantMembers({ isSystemTenant }: TenantMembersProps) {
  const { id } = useParams<{ id: string }>()

  return <MembersSettings tenantId={id} isSystemTenant={isSystemTenant} />
}
