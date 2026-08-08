/**
 * MFA API service — step-up only.
 *
 * The console does not manage anyone's own MFA factors. Enrolling, disabling
 * and recovering your own factors lives in the identity app, which owns every
 * account-management flow; the console links out to it. The internal API mounts
 * only the step-up ceremony to match, so the enrollment client that used to
 * live here had no screens left to serve and no routes left to call.
 *
 * What remains is what the console needs to satisfy its OWN sensitive actions:
 * read which factors the admin holds, then use one to reach acr=2.
 *
 * Admin remediation (resetting somebody ELSE's MFA) is a user-management
 * concern and lives with the other user endpoints, not here.
 */

import { get, post } from "./client"
import { unwrap, assertSuccess } from "./_lib/unwrap"
import type { ApiResponse } from "./types"
import type { WebAuthnAssertionOptions } from "@/lib/webauthn"

// ── Types ────────────────────────────────────────────────────────────────────

export interface MFAStatusResponse {
  is_totp_enabled: boolean
  is_webauthn_enabled: boolean
  is_sms_available: boolean
  is_email_otp_available: boolean
  backup_codes_count: number
  webauthn_keys: MFAWebAuthnKey[]
  mfa_enabled_at?: string | null
  /**
   * The tenant's POLICY: which factors a user may enrol at all. Distinct from
   * the is_*_enabled flags, which say what they have already set up.
   *
   * The step-up dialog filters on this — offering a factor the tenant disabled
   * presents a challenge the server will refuse.
   */
  allowed_methods: string[]
  /** True when the tenant requires at least one factor. */
  mfa_required: boolean
  /** Authenticator code length for this tenant — 6 or 8. Never assume 6. */
  totp_digits?: number
}

export interface MFAWebAuthnKey {
  credential_uuid: string
  name: string
  transport: string
  last_used_at?: string | null
  created_at: string
}

// ── API ──────────────────────────────────────────────────────────────────────

const BASE = "/mfa"

export async function fetchMFAStatus(): Promise<MFAStatusResponse> {
  const r: ApiResponse<MFAStatusResponse> = await get<ApiResponse<MFAStatusResponse>>(`${BASE}/status`)
  return unwrap(r, "fetch MFA status")
}

// Starts a passkey assertion ceremony (step-up). The returned challenge is
// consumed by navigator.credentials.get; the assertion is then sent to
// /step-up/verify. The matching server-side session is keyed to the user.
export async function beginWebAuthnAuthentication(): Promise<WebAuthnAssertionOptions> {
  const r: ApiResponse<WebAuthnAssertionOptions> = await post<ApiResponse<WebAuthnAssertionOptions>>(`${BASE}/webauthn/auth/begin`)
  return unwrap(r, "begin WebAuthn authentication")
}

// ── Step-up authentication ─────────────────────────────────────────────────────
// Elevates a session (acr=2) before a sensitive action. The verified token is
// then passed as a Bearer header on the gated request.

export interface StepUpChallenge {
  challenge_token: string
  allowed_methods: string[]
}

export interface StepUpVerifyResult {
  access_token: string
  expires_in: number
}

export async function issueStepUpChallenge(): Promise<StepUpChallenge> {
  const r = await post<ApiResponse<StepUpChallenge>>(`${BASE}/step-up/challenge`)
  return unwrap(r, "issue step-up challenge")
}

export async function sendStepUpSMS(): Promise<void> {
  const r = await post<ApiResponse<void>>(`${BASE}/step-up/send-sms`)
  assertSuccess(r, "send step-up SMS code")
}

export async function sendStepUpEmailOtp(): Promise<void> {
  const r = await post<ApiResponse<void>>(`${BASE}/step-up/send-email-otp`)
  assertSuccess(r, "send step-up Email OTP code")
}

// Proof for a step-up verification: a typed code (totp/sms/backup_code) or a
// WebAuthn assertion (passkey). Exactly one is supplied per call.
export interface StepUpProof {
  code?: string
  assertion?: unknown
}

export async function verifyStepUp(
  challengeToken: string,
  method: string,
  proof: StepUpProof,
): Promise<StepUpVerifyResult> {
  const r = await post<ApiResponse<StepUpVerifyResult>>(`${BASE}/step-up/verify`, {
    challenge_token: challengeToken,
    method,
    code: proof.code,
    assertion: proof.assertion,
  })
  return unwrap(r, "verify step-up authentication")
}
