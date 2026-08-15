import {
  ShieldCheck,
  Key,
  Smartphone,
  MessageSquare,
  Mail,
  RefreshCw,
  ShieldOff,
  Clock,
  Calendar,
} from "lucide-react"
import { format } from "date-fns"
import type { LucideIcon } from "lucide-react"
import { InformationCard } from "@/components/card"
import { ListSkeleton, StatusBadge } from "@/components/details"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useUserMFA, useResetUserMfa, useResetUserMfaMethod } from "@/hooks/useUsers"
import { useToast } from "@/hooks/useToast"
import type { UserMfaMethod } from "@/services/api/users"
import type { UserMFAWebAuthnKey } from "@/services/api/users/types"

interface UserMFAProps {
  userId: string
}

function formatDate(value?: string | null) {
  if (!value) return "—"
  try {
    return format(new Date(value), "PPp")
  } catch {
    return "—"
  }
}

/** A compact section heading used to group rows within the card. */
function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{children}</p>
  )
}

interface MethodRowProps {
  icon: LucideIcon
  title: string
  description: string
  enabled: boolean
  actions?: RowActionItem[]
}

/** A single MFA method, rendered as a full-width status row. */
function MethodRow({ icon: Icon, title, description, enabled, actions }: MethodRowProps) {
  return (
    <div data-md-listing-item className="flex items-center justify-between gap-3 rounded-lg border p-4">
      <div className="flex min-w-0 items-center gap-3">
        <div data-md-listing-icon className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
          <Icon className="size-4" />
        </div>
        <div className="min-w-0">
          <p className="text-sm font-medium">{title}</p>
          <p className="mt-0.5 text-sm text-muted-foreground">{description}</p>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-1">
        <StatusBadge status={enabled ? "enabled" : "disabled"} />
        {enabled && actions && actions.length > 0 && <RowActions items={actions} />}
      </div>
    </div>
  )
}

export function UserMFA({ userId }: UserMFAProps) {
  const { data, isLoading, isError } = useUserMFA(userId)
  const { showSuccess, showError } = useToast()
  const resetMfaMutation = useResetUserMfa()
  const resetMethodMutation = useResetUserMfaMethod()

  const handleReset = async () => {
    try {
      await resetMfaMutation.mutateAsync(userId)
      showSuccess("MFA reset successfully")
    } catch (error) {
      showError(error)
    }
  }

  const resetAllActions: RowActionItem[] = [
    {
      key: "reset-mfa",
      label: "Reset MFA",
      icon: ShieldOff,
      destructive: true,
      onSelect: handleReset,
      confirm: {
        title: "Reset MFA",
        description:
          "This removes all multi-factor authentication methods for this user. They will be prompted to set up MFA again on their next sign-in.",
        confirmText: "Reset MFA",
      },
    },
  ]

  // Per-method reset (e.g. wipe a lost phone's TOTP while leaving a passkey).
  // RowActions handles the confirmation step before this runs.
  const resetMethod = async (method: UserMfaMethod, label: string) => {
    try {
      await resetMethodMutation.mutateAsync({ userId, method })
      showSuccess(`${label} reset for this user`)
    } catch (error) {
      showError(error)
    }
  }

  const resetAction = (method: UserMfaMethod, label: string): RowActionItem[] => [
    {
      key: "reset",
      label: `Reset ${label}`,
      icon: ShieldOff,
      destructive: true,
      onSelect: () => resetMethod(method, label),
      confirm: {
        title: `Reset ${label}`,
        description: `This removes ${label} for this user. They will need to set it up again on their next sign-in. Continue?`,
        confirmText: "Reset",
      },
    },
  ]

  const hasAnyMfa =
    !!data && (data.is_totp_enabled || data.is_webauthn_enabled || data.is_sms_enabled || data.is_email_otp_enabled)
  const passkeys = data?.webauthn_keys ?? []

  return (
    <InformationCard
      title="Multi-Factor Authentication"
      description="Authentication methods protecting this user's account. Reset to clear every method at once."
      icon={ShieldCheck}
      action={hasAnyMfa ? <RowActions items={resetAllActions} variant="header" /> : undefined}
    >
      {isLoading && <ListSkeleton />}

      {isError && (
        <p className="py-8 text-center text-sm text-destructive">
          Failed to load MFA configuration
        </p>
      )}

      {data && (
        <div className="space-y-6">
          {/* Methods */}
          <section className="space-y-3">
            <SectionLabel>Authentication methods</SectionLabel>
            <div className="space-y-3">
              <MethodRow
                icon={Smartphone}
                title="TOTP"
                description="Authenticator app — Google Authenticator, Authy, 1Password"
                enabled={data.is_totp_enabled}
                actions={resetAction("totp", "TOTP")}
              />
              <MethodRow
                icon={MessageSquare}
                title="SMS"
                description="One-time codes texted to a verified phone number"
                enabled={data.is_sms_enabled}
                actions={resetAction("sms", "SMS")}
              />
              <MethodRow
                icon={Mail}
                title="Email OTP"
                description="One-time codes sent to a verified email address"
                enabled={data.is_email_otp_enabled}
                actions={resetAction("email_otp", "Email OTP")}
              />
              <MethodRow
                icon={Key}
                title="Passkeys"
                description="Biometrics or security keys — Face ID, Touch ID, YubiKey"
                enabled={data.is_webauthn_enabled}
                actions={resetAction("webauthn", "Passkeys")}
              />
              <MethodRow
                icon={RefreshCw}
                title="Backup Codes"
                description={
                  data.backup_codes_count > 0
                    ? `${data.backup_codes_count} unused recovery codes remaining`
                    : "Single-use recovery codes for account access"
                }
                enabled={data.backup_codes_count > 0}
                actions={resetAction("backup_code", "Backup Codes")}
              />
            </div>
          </section>

          {/* Registered passkeys */}
          {passkeys.length > 0 && (
            <section className="space-y-3">
              <div className="flex items-center gap-2">
                <SectionLabel>Registered Passkeys</SectionLabel>
                <span className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-muted px-1.5 text-xs font-medium text-muted-foreground">
                  {passkeys.length}
                </span>
              </div>
              <div className="space-y-3">
                {passkeys.map((key: UserMFAWebAuthnKey) => (
                  <div
                    data-md-listing-item
                    key={key.credential_id}
                    className="flex items-start gap-3 rounded-lg border p-4"
                  >
                    <div data-md-listing-icon className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                      <Key className="size-4" />
                    </div>
                      <div className="min-w-0 space-y-1">
                        <p className="break-words text-sm font-medium">{key.name}</p>
                      <div data-md-listing-meta className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                        {key.transport && key.transport.length > 0 && (
                          <span className="capitalize">{key.transport.split(",").join(", ")}</span>
                        )}
                        <span className="inline-flex items-center gap-1">
                          <Clock className="size-3" />
                          Last used {formatDate(key.last_used_at)}
                        </span>
                        <span className="inline-flex items-center gap-1">
                          <Calendar className="size-3" />
                          Added {formatDate(key.created_at)}
                        </span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </section>
          )}

          {data.mfa_enabled_at && (
            <p className="flex items-center gap-1.5 border-t pt-4 text-xs text-muted-foreground">
              <Calendar className="size-3" />
              MFA enabled on {formatDate(data.mfa_enabled_at)}
            </p>
          )}
        </div>
      )}

    </InformationCard>
  )
}
