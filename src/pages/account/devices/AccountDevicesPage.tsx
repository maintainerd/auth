import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { MonitorSmartphone, Trash2, Globe, MapPin, Clock, Calendar, ShieldCheck } from 'lucide-react'
import AccountLayout from '@/components/layout/AccountLayout'
import { SettingsCard } from '@/components/card'
import { Button } from '@/components/ui/button'
import { ListingItemCard, ListingItemMeta } from '@/components/details'
import { useToast } from '@/hooks/useToast'
import { formatUserAgent } from '@/lib/userAgent'
import {
  fetchTrustedDevices,
  revokeTrustedDevice,
  type TrustedDevice,
} from '@/services/api/account'

function fmt(value?: string | null) {
  if (!value) return '—'
  try {
    return new Date(value).toLocaleString()
  } catch {
    return '—'
  }
}

function DeviceRow({
  device,
  onRevoke,
  revoking,
}: {
  device: TrustedDevice
  onRevoke: (uuid: string) => void
  revoking: boolean
}) {
  const [confirming, setConfirming] = useState(false)
  const label = device.device_name || formatUserAgent(device.user_agent)

  return (
    <li>
      <ListingItemCard
        icon={MonitorSmartphone}
        className="items-start p-3"
        actionClassName="flex shrink-0 items-center gap-2"
        action={confirming ? (
          <>
            <Button
              size="sm"
              variant="destructive"
              className="h-9 sm:h-7 text-xs"
              disabled={revoking}
              onClick={() => {
                onRevoke(device.uuid)
                setConfirming(false)
              }}
            >
              Confirm revoke
            </Button>
            <Button
              size="sm"
              variant="outline"
              className="h-9 sm:h-7 text-xs"
              onClick={() => setConfirming(false)}
            >
              Cancel
            </Button>
          </>
        ) : (
          <Button
            size="sm"
            variant="ghost"
            className="h-9 sm:h-7 gap-1 text-xs text-destructive hover:text-destructive"
            disabled={revoking}
            onClick={() => setConfirming(true)}
          >
            <Trash2 className="size-3" />
            Revoke
          </Button>
        )}
      >
        <div className="min-w-0 space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <p className="truncate text-sm font-medium" title={device.user_agent ?? undefined}>
              {label}
            </p>
            {device.current && (
              <span className="inline-flex items-center rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
                This device
              </span>
            )}
          </div>
          <ListingItemMeta>
            {device.location && (
              <span className="inline-flex items-center gap-1">
                <MapPin className="size-3" />
                {device.location}
              </span>
            )}
            {device.ip_address && (
              <span className="inline-flex items-center gap-1 font-mono">
                <Globe className="size-3" />
                <span className="break-all">{device.ip_address}</span>
              </span>
            )}
            <span className="inline-flex items-center gap-1">
              <Clock className="size-3" />
              Last used {fmt(device.last_seen_at)}
            </span>
            <span className="inline-flex items-center gap-1">
              <Calendar className="size-3" />
              Trusted {fmt(device.created_at)}
            </span>
            {device.trusted_until && (
              <span className="inline-flex items-center gap-1">
                <ShieldCheck className="size-3" />
                Expires {fmt(device.trusted_until)}
              </span>
            )}
          </ListingItemMeta>
        </div>
      </ListingItemCard>
    </li>
  )
}

export default function AccountDevicesPage() {
  const qc = useQueryClient()
  const { showError, showSuccess } = useToast()

  const { data: devices = [], isLoading } = useQuery({
    queryKey: ['account', 'devices'],
    queryFn: fetchTrustedDevices,
  })

  const revokeMutation = useMutation({
    mutationFn: (uuid: string) => revokeTrustedDevice(uuid),
    onSuccess: () => {
      showSuccess('Device trust revoked')
      qc.invalidateQueries({ queryKey: ['account', 'devices'] })
    },
    onError: (err) => showError(err, 'Could not revoke device'),
  })

  return (
    <AccountLayout title="Trusted Devices">
      <SettingsCard
        title="Devices that skip MFA"
        description="Devices are added when you choose to trust them during MFA."
        icon={MonitorSmartphone}
      >
        {isLoading && <p className="text-sm text-muted-foreground">Loading devices…</p>}
        {!isLoading && devices.length === 0 && (
          <div className="flex flex-col items-center gap-3 py-8 text-center">
            <MonitorSmartphone className="size-8 text-muted-foreground/50" />
            <p className="text-sm text-muted-foreground">No trusted devices</p>
          </div>
        )}
        {!isLoading && devices.length > 0 && (
          <ul className="space-y-2">
            {devices.map((device: TrustedDevice) => (
              <DeviceRow
                key={device.uuid}
                device={device}
                onRevoke={(uuid) => revokeMutation.mutate(uuid)}
                revoking={revokeMutation.isPending}
              />
            ))}
          </ul>
        )}
      </SettingsCard>
    </AccountLayout>
  )
}
