/**
 * MFA API Service — self-service enrollment endpoints
 */

import { get, post, deleteRequest } from "./client"
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
   * The tenant's POLICY: which factors this user may enrol at all. Distinct from
   * the is_*_enabled flags above, which say what they have already set up.
   *
   * Enrolment UIs must filter on this. Offering a factor the tenant disabled
   * walks the user through a setup flow that is refused at the final step, which
   * reads as a broken product rather than a policy decision.
   */
  allowed_methods: string[]
  /** True when the tenant requires at least one factor. */
  mfa_required: boolean
  /**
   * Authenticator code length for this tenant — 6 or 8.
   *
   * Never assume 6. A tenant on 8 gets an authenticator showing 8 digits, and an
   * input capped at 6 makes the code impossible to type.
   */
  totp_digits?: number
}

export interface MFAWebAuthnKey {
  credential_uuid: string
  name: string
  transport: string
  last_used_at?: string | null
  created_at: string
}

export interface TOTPEnrollResponse {
  secret: string
  qr_code_url: string
  /** Code length the generated key uses — 6 or 8, per the tenant's policy. */
  digits?: number
  /** Code rotation window in seconds. */
  period_seconds?: number
}

export interface TOTPVerifyRequest {
  code: string
}

export interface BackupCodesResponse {
  codes: string[]
}

export interface WebAuthnCredentialSummary {
  credential_uuid: string
  name: string
  transport: string
  created_at: string
}

// ── API ──────────────────────────────────────────────────────────────────────

const BASE = "/mfa"

export async function fetchMFAStatus(): Promise<MFAStatusResponse> {
  const r: ApiResponse<MFAStatusResponse> = await get<ApiResponse<MFAStatusResponse>>(`${BASE}/status`)
  return unwrap(r, "fetch MFA status")
}

export async function beginTOTPEnrollment(): Promise<TOTPEnrollResponse> {
  const r: ApiResponse<TOTPEnrollResponse> = await post<ApiResponse<TOTPEnrollResponse>>(`${BASE}/totp/enroll`)
  return unwrap(r, "begin TOTP enrollment")
}

export async function finishTOTPEnrollment(data: TOTPVerifyRequest): Promise<BackupCodesResponse> {
  const r: ApiResponse<BackupCodesResponse> = await post<ApiResponse<BackupCodesResponse>>(`${BASE}/totp/verify`, data)
  return unwrap(r, "finish TOTP enrollment")
}

export async function disableTOTP(): Promise<void> {
  const r: ApiResponse<void> = await deleteRequest<ApiResponse<void>>(`${BASE}/totp`)
  assertSuccess(r, "disable TOTP")
}

export async function regenerateBackupCodes(): Promise<BackupCodesResponse> {
  const r: ApiResponse<BackupCodesResponse> = await post<ApiResponse<BackupCodesResponse>>(`${BASE}/backup-codes/regenerate`)
  return unwrap(r, "regenerate backup codes")
}

// Shape returned by /mfa/webauthn/register/begin. Binary fields (challenge,
// user.id) are base64url strings on the wire and must be decoded to ArrayBuffers
// before being handed to navigator.credentials.create.
export interface WebAuthnCreationOptions {
  publicKey: {
    rp: { name: string; id: string }
    user: { name: string; displayName: string; id: string }
    challenge: string
    pubKeyCredParams: Array<{ type: "public-key"; alg: number }>
    timeout: number
    attestation?: AttestationConveyancePreference
    authenticatorSelection?: AuthenticatorSelectionCriteria
  }
}

export async function beginWebAuthnRegistration(): Promise<WebAuthnCreationOptions> {
  const r: ApiResponse<WebAuthnCreationOptions> = await post<ApiResponse<WebAuthnCreationOptions>>(`${BASE}/webauthn/register/begin`)
  return unwrap(r, "begin WebAuthn registration")
}

export async function finishWebAuthnRegistration(
  name: string,
  credential: unknown,
): Promise<WebAuthnCredentialSummary> {
  const r: ApiResponse<WebAuthnCredentialSummary> = await post<ApiResponse<WebAuthnCredentialSummary>>(
    `${BASE}/webauthn/register/finish?name=${encodeURIComponent(name)}`,
    credential,
  )
  return unwrap(r, "finish WebAuthn registration")
}

// Starts a passkey assertion ceremony (step-up). The returned challenge is
// consumed by navigator.credentials.get; the assertion is then sent to
// /step-up/verify. The matching server-side session is keyed to the user.
export async function beginWebAuthnAuthentication(): Promise<WebAuthnAssertionOptions> {
  const r: ApiResponse<WebAuthnAssertionOptions> = await post<ApiResponse<WebAuthnAssertionOptions>>(`${BASE}/webauthn/auth/begin`)
  return unwrap(r, "begin WebAuthn authentication")
}

export async function deleteWebAuthnCredential(credentialUuid: string): Promise<void> {
  const r: ApiResponse<void> = await deleteRequest<ApiResponse<void>>(`${BASE}/webauthn/${credentialUuid}`)
  assertSuccess(r, "delete WebAuthn credential")
}

// ── SMS MFA ──────────────────────────────────────────────────────────────────

export async function beginSMSEnrollment(phone: string): Promise<void> {
  const r = await post<ApiResponse<void>>(`${BASE}/sms/enroll`, { phone })
  assertSuccess(r, "begin SMS enrollment")
}

export async function verifySMSEnrollment(phone: string, code: string): Promise<void> {
  const r = await post<ApiResponse<void>>(`${BASE}/sms/verify`, { phone, code })
  assertSuccess(r, "verify SMS enrollment")
}

export async function disableSMS(): Promise<void> {
  const r = await deleteRequest<ApiResponse<void>>(`${BASE}/sms`)
  assertSuccess(r, "disable SMS")
}

// ── Email OTP MFA ────────────────────────────────────────────────────────────

export async function beginEmailOtpEnrollment(email: string): Promise<void> {
  const r = await post<ApiResponse<void>>(`${BASE}/email-otp/enroll`, { email })
  assertSuccess(r, "begin Email OTP enrollment")
}

export async function verifyEmailOtpEnrollment(email: string, code: string): Promise<void> {
  const r = await post<ApiResponse<void>>(`${BASE}/email-otp/verify`, { email, code })
  assertSuccess(r, "verify Email OTP enrollment")
}

export async function disableEmailOtp(): Promise<void> {
  const r = await deleteRequest<ApiResponse<void>>(`${BASE}/email-otp`)
  assertSuccess(r, "disable Email OTP")
}

// ── Reset all (self-service) ──────────────────────────────────────────────────
// Clears every MFA factor for the signed-in user. The server scopes this to the
// session identity (no target param), so it can only ever reset your own MFA.
export async function resetAllMFA(): Promise<void> {
  const r = await post<ApiResponse<void>>(`${BASE}/reset`)
  assertSuccess(r, "reset MFA")
}

// ── Step-up authentication ─────────────────────────────────────────────────────
// Used to elevate a session (acr=2) before a sensitive action such as deleting
// the account or revoking all sessions. The verified token is then passed as a
// Bearer header on the gated request.

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
