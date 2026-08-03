import { useMemo } from "react"
import { MonitorSmartphone, Globe, MapPin, Calendar, Clock, ShieldCheck, Trash2 } from "lucide-react"
import { safeFormat } from "@/lib/formatDate"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton } from "@/components/details"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useUserTrustedDevices, useRevokeUserDevice } from "@/hooks/useUsers"
import { useToast } from "@/hooks/useToast"
import { formatUserAgent } from "@/lib/userAgent"
import type { TrustedDevice } from "@/services/api/users/types"

interface UserTrustedDevicesProps {
  userId: string
}

function formatDate(value?: string | null) {
  return safeFormat(value, "PPp")
}

export function UserTrustedDevices({ userId }: UserTrustedDevicesProps) {
  const { data, isLoading, isError } = useUserTrustedDevices(userId)
  const devices = useMemo(() => data ?? [], [data])
  const { showSuccess, showError } = useToast()
  const revokeMutation = useRevokeUserDevice()

  const revokeDevice = async (device: TrustedDevice) => {
    try {
      await revokeMutation.mutateAsync({ userId, deviceId: device.uuid })
      showSuccess("Device revoked")
    } catch (e) {
      showError(e)
    }
  }

  const deviceActions = (device: TrustedDevice): RowActionItem[] => [
    {
      key: "revoke",
      label: "Revoke device",
      icon: Trash2,
      destructive: true,
      onSelect: () => revokeDevice(device),
      confirm: {
        title: "Revoke trusted device",
        description:
          "The user will need to re-verify this device (e.g. via MFA) the next time they sign in from it. Continue?",
        confirmText: "Revoke",
      },
    },
  ]

  return (
    <InformationCard
      title="Trusted Devices"
      description="Devices this user has marked as trusted so they can skip MFA. Revoking one forces re-verification on the next sign-in from it."
      icon={MonitorSmartphone}
    >
      <div className="space-y-4">
        {isLoading && <ListSkeleton />}

        {isError && <p className="py-8 text-center text-sm text-destructive">Failed to load devices</p>}

        {data && data.length === 0 && (
          <EmptyState
            icon={MonitorSmartphone}
            title="No trusted devices"
            description="This user has not trusted any devices."
          />
        )}

        {devices.length > 0 && (
          <div className="space-y-3">
            {devices.map((device) => (
              <div
                data-md-listing-item
                key={device.uuid}
                className="flex items-start justify-between gap-3 rounded-lg border p-4"
              >
                <div className="flex min-w-0 items-start gap-3">
                  <div data-md-listing-icon className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <MonitorSmartphone className="size-4" />
                  </div>
                  <div className="min-w-0 space-y-1">
                    <p className="break-words text-sm font-medium" title={device.user_agent || undefined}>
                      {device.device_name || formatUserAgent(device.user_agent)}
                    </p>
                    <div data-md-listing-meta className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                      {device.location && (
                        <span className="inline-flex items-center gap-1">
                          <MapPin className="size-3" />
                          {device.location}
                        </span>
                      )}
                      {device.ip_address && (
                        <span className="inline-flex items-center gap-1 font-mono">
                          <Globe className="size-3" />
                          {device.ip_address}
                        </span>
                      )}
                      <span className="inline-flex items-center gap-1">
                        <Clock className="size-3" />
                        Last seen {formatDate(device.last_seen_at)}
                      </span>
                      <span className="inline-flex items-center gap-1">
                        <Calendar className="size-3" />
                        Trusted {formatDate(device.created_at)}
                      </span>
                      {device.trusted_until && (
                        <span className="inline-flex items-center gap-1">
                          <ShieldCheck className="size-3" />
                          Expires {formatDate(device.trusted_until)}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
                <RowActions items={deviceActions(device)} />
              </div>
            ))}
          </div>
        )}
      </div>
    </InformationCard>
  )
}
