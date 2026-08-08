import { Smartphone, MessageSquare, KeyRound, Fingerprint, Mail } from "lucide-react"
import type { LucideIcon } from "lucide-react"

/**
 * Presentation metadata for the MFA factors the step-up / login-MFA UIs drive.
 * `numeric` masks the code input to 6 digits; `webauthn` factors take no typed
 * code — they invoke the platform authenticator instead. Shared by the global
 * StepUpDialog and the login second-step (LoginMFAStep) so the factor list,
 * labels, and icons stay consistent.
 */
export interface MFAMethodMeta {
  label: string
  icon: LucideIcon
  numeric: boolean
  webauthn?: boolean
}

export const MFA_METHOD_META: Record<string, MFAMethodMeta> = {
  totp: { label: "Authenticator app", icon: Smartphone, numeric: true },
  webauthn: { label: "Passkey", icon: Fingerprint, numeric: false, webauthn: true },
  sms: { label: "Text message", icon: MessageSquare, numeric: true },
  email_otp: { label: "Email OTP", icon: Mail, numeric: true },
  backup_code: { label: "Backup code", icon: KeyRound, numeric: false },
}

/**
 * Normalizes a typed MFA proof before submission. Backup codes are single-use
 * and entered one at a time, but users commonly paste the whole downloaded list
 * (one code per line) — so for backup_code we trim and take the first non-empty
 * token. Other methods are just trimmed.
 */
export function extractMFACode(method: string, raw: string): string {
  const trimmed = raw.trim()
  if (method === "backup_code") {
    return trimmed.split(/\s+/)[0] ?? ""
  }
  return trimmed
}
