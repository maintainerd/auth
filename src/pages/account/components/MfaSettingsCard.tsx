import { ChevronRight, ShieldCheck } from "lucide-react"
import { useQuery } from "@tanstack/react-query"

import { SettingsCard } from "@/components/card"
import { ListingItemCard, StatusBadge } from "@/components/details"
import { fetchMFAStatus } from "@/services/api/mfa"

interface MfaSettingsCardProps {
  onManage: () => void
}

export function MfaSettingsCard({ onManage }: MfaSettingsCardProps) {
  const { data } = useQuery({ queryKey: ["mfa", "status"], queryFn: fetchMFAStatus, retry: false })
  const totp = data?.is_totp_enabled ?? false
  const sms = data?.is_sms_available ?? false
  const passkeys = (data?.webauthn_keys?.length ?? 0) > 0
  const active = [totp, sms, passkeys].filter(Boolean).length

  return (
    <SettingsCard
      title="Multi-factor authentication"
      description="Add a second step at sign-in to protect your account."
      icon={ShieldCheck}
    >
      <ListingItemCard
        as="button"
        onClick={onManage}
        className="items-center"
        action={
          <span className="flex items-center gap-1 text-sm text-muted-foreground">
            Manage
            <ChevronRight className="size-4" />
          </span>
        }
      >
        <div className="flex min-w-0 items-center gap-3">
          <StatusBadge status={active > 0 ? "active" : "inactive"} />
          <span className="text-sm text-muted-foreground">
            {active > 0 ? `${active} method${active === 1 ? "" : "s"} active` : "No methods set up yet"}
          </span>
        </div>
      </ListingItemCard>
    </SettingsCard>
  )
}
