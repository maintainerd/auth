/**
 * Branded auth-page copy.
 *
 * The console's branding editor lets a tenant rewrite the title and subtitle of
 * every hosted auth page and previews the result. This module is the identity
 * app's side of that contract: it reads `metadata.login_page_content` and falls
 * back to the same defaults the console previews, so what an operator sees in
 * the preview is what actually renders here.
 *
 * The default titles/subtitles below are generated from the console's
 * DEFAULT_LOGIN_PAGE_PREVIEWS — keep the two in step when adding a page.
 */

export const LOGIN_PAGE_IDS = [
  "login",
  "login-methods-loading",
  "login-methods-unavailable",
  "login-error",
  "login-magic-link-sent",
  "login-mfa-code",
  "login-mfa-passkey",
  "login-mfa-backup-code",
  "login-mfa-no-methods",
  "magic-link-verifying",
  "magic-link-success",
  "magic-link-mfa",
  "magic-link-error",
  "sms-login-phone",
  "sms-login-phone-required",
  "sms-login-code",
  "registration",
  "registration-loading",
  "registration-invalid-link",
  "registration-requirements-warning",
  "invite-accept",
  "invite-session-mismatch",
  "invite-invalid",
  "profile-form",
  "profile-success",
  "verify-email-form",
  "verify-email-success",
  "forgot-password-form",
  "forgot-password-sent",
  "reset-password-form",
  "reset-password-success",
  "reset-password-invalid",
  "backup-code-recovery",
  "phone-verification-loading",
  "phone-verification-send",
  "phone-verification-code",
  "phone-verification-verified",
  "account-link-invalid",
  "account-link-loading",
  "account-link-signed-out",
  "account-link-confirm",
  "account-link-success",
  "oauth-authorize-loading",
  "oauth-authorize-error",
  "oauth-consent-loading",
  "oauth-consent-request",
  "oauth-consent-error",
  "oauth-device-authorize",
  "oauth-device-approved",
  "oauth-device-denied",
  "oauth-ciba-confirm",
  "oauth-ciba-error",
  "oauth-ciba-approved",
  "oauth-ciba-denied",
  "oauth-grants-loading",
  "oauth-grants-empty",
  "oauth-grants-list",
  "oauth-grants-error",
  "oauth-end-session",
  "login-success",
  "login-success-mfa-nudge",
  "account-erasure-request",
  "account-erasure-confirm",
  "account-erasure-error",
  "account-erasure-success",
  "account-locked",
  "too-many-requests",
  "no-access-authenticated",
  "no-access-guest",
  "service-unavailable",
  "not-found",
] as const

export type LoginPageId = (typeof LOGIN_PAGE_IDS)[number]

export interface LoginPageCopy {
  title: string
  subtitle: string
}

export const DEFAULT_LOGIN_PAGE_COPY: Record<LoginPageId, LoginPageCopy> = {
  "login": { title: "Welcome back", subtitle: "Sign in to your account to continue." },
  "login-methods-loading": { title: "Welcome back", subtitle: "Loading the sign-in methods available for this application." },
  "login-methods-unavailable": { title: "Welcome back", subtitle: "No sign-in methods are available for this application." },
  "login-error": { title: "Welcome back", subtitle: "Sign in to your account to continue." },
  "login-magic-link-sent": { title: "Check your email", subtitle: "A secure sign-in link will arrive shortly." },
  "login-mfa-code": { title: "Two-step verification", subtitle: "Confirm your second factor to finish signing in." },
  "login-mfa-passkey": { title: "Two-step verification", subtitle: "Confirm your identity with a second factor." },
  "login-mfa-backup-code": { title: "Two-step verification", subtitle: "Enter one backup code to finish signing in." },
  "login-mfa-no-methods": { title: "Two-step verification", subtitle: "MFA is required but no supported factor is available. Contact your administrator." },
  "magic-link-verifying": { title: "Signing you in", subtitle: "We're securely verifying your magic link." },
  "magic-link-success": { title: "You're signed in", subtitle: "Your magic link was verified successfully. We're taking you to the app." },
  "magic-link-mfa": { title: "Two-step verification", subtitle: "Your magic link was accepted. Confirm a different factor to finish signing in." },
  "magic-link-error": { title: "Magic link unavailable", subtitle: "The magic link is invalid or has expired. Please request a new one." },
  "sms-login-phone": { title: "Sign in with SMS", subtitle: "Enter your phone number to receive a one-time code" },
  "sms-login-phone-required": { title: "Sign in with SMS", subtitle: "Enter your phone number to receive a one-time code" },
  "sms-login-code": { title: "Sign in with SMS", subtitle: "Enter the verification code sent to +1234567890" },
  "registration": { title: "Create your account", subtitle: "Sign up to get started." },
  "registration-loading": { title: "Loading registration", subtitle: "Preparing the registration experience." },
  "registration-invalid-link": { title: "This sign-up link is no longer valid", subtitle: "It may have been renamed, deactivated, or replaced. Ask whoever sent it for an up-to-date link." },
  "registration-requirements-warning": { title: "Create your account", subtitle: "Sign up to get started." },
  "invite-accept": { title: "Accept your invitation", subtitle: "Set up your password to complete registration." },
  "invite-session-mismatch": { title: "Accept your invitation", subtitle: "You're signed in as sam@company.com. This invitation is for alex@company.com. Sign out to accept it." },
  "invite-invalid": { title: "Invalid invite link", subtitle: "This invite link is missing the email parameter. Please request a new invitation." },
  "profile-form": { title: "Complete your profile", subtitle: "Just a few details to get started." },
  "profile-success": { title: "All set!", subtitle: "Your profile has been created. You can now access the app." },
  "verify-email-form": { title: "Verify your email", subtitle: "Enter the code sent to alex@company.com." },
  "verify-email-success": { title: "Email verified", subtitle: "Your email has been confirmed. You can now sign in to your account." },
  "forgot-password-form": { title: "Forgot your password?", subtitle: "Enter your email and we'll send reset instructions." },
  "forgot-password-sent": { title: "Check your email", subtitle: "If an account exists with this email, password reset instructions will arrive shortly." },
  "reset-password-form": { title: "Reset your password", subtitle: "Enter your new password below." },
  "reset-password-success": { title: "Password reset", subtitle: "You can now sign in with your new password." },
  "reset-password-invalid": { title: "Invalid reset link", subtitle: "This password reset link is invalid or has expired. Please request a new one." },
  "backup-code-recovery": { title: "Backup Code Recovery", subtitle: "Enter one of your saved backup codes to regain access to your account." },
  "phone-verification-loading": { title: "Verify Phone", subtitle: "Confirm your phone number with a one-time code sent by SMS." },
  "phone-verification-send": { title: "Verify Phone", subtitle: "Confirm your phone number with a one-time code sent by SMS." },
  "phone-verification-code": { title: "Verify Phone", subtitle: "Confirm your phone number with a one-time code sent by SMS." },
  "phone-verification-verified": { title: "Your phone is verified", subtitle: "+15551234567" },
  "account-link-invalid": { title: "Invalid Link", subtitle: "This account link URL is incomplete or has expired." },
  "account-link-loading": { title: "Link your Google account", subtitle: "Checking sign-in status..." },
  "account-link-signed-out": { title: "Link your Google account", subtitle: "Your Google account uses alex@company.com which is already registered. Sign in to confirm you own this account and link them." },
  "account-link-confirm": { title: "Link your Google account", subtitle: "Your Google account uses alex@company.com which is already registered. Sign in to confirm you own this account and link them." },
  "account-link-success": { title: "Accounts linked!", subtitle: "Redirecting you to the app..." },
  "oauth-authorize-loading": { title: "Authorizing", subtitle: "Preparing the secure redirect." },
  "oauth-authorize-error": { title: "Authorization unavailable", subtitle: "Authorization failed." },
  "oauth-consent-loading": { title: "Loading request", subtitle: "" },
  "oauth-consent-request": { title: "Example App", subtitle: "wants access to your account." },
  "oauth-consent-error": { title: "Consent unavailable", subtitle: "Consent request unavailable." },
  "oauth-device-authorize": { title: "Authorize device", subtitle: "Enter the code shown on your device." },
  "oauth-device-approved": { title: "Device approved", subtitle: "" },
  "oauth-device-denied": { title: "Device denied", subtitle: "" },
  "oauth-ciba-confirm": { title: "Confirm sign in", subtitle: "Confirm this sign-in request from Example App." },
  "oauth-ciba-error": { title: "Confirm sign in", subtitle: "" },
  "oauth-ciba-approved": { title: "Request approved", subtitle: "" },
  "oauth-ciba-denied": { title: "Request denied", subtitle: "" },
  "oauth-grants-loading": { title: "Connected apps", subtitle: "Review applications you have authorized." },
  "oauth-grants-empty": { title: "Connected apps", subtitle: "Review applications you have authorized." },
  "oauth-grants-list": { title: "Connected apps", subtitle: "Review applications you have authorized." },
  "oauth-grants-error": { title: "Connected apps", subtitle: "Review applications you have authorized." },
  "oauth-end-session": { title: "Signing out", subtitle: "Completing the logout request." },
  "login-success": { title: "You're signed in", subtitle: "You can continue from this session or sign out when you are done." },
  "login-success-mfa-nudge": { title: "You're signed in", subtitle: "You can continue from this session or sign out when you are done." },
  "account-erasure-request": { title: "Request account deletion", subtitle: "Under GDPR Article 17 you have the right to have your personal data erased." },
  "account-erasure-confirm": { title: "Request account deletion", subtitle: "Under GDPR Article 17 you have the right to have your personal data erased." },
  "account-erasure-error": { title: "Request account deletion", subtitle: "Under GDPR Article 17 you have the right to have your personal data erased." },
  "account-erasure-success": { title: "Request submitted", subtitle: "Your data will be anonymised within 30 days. You will continue to have access to your account until then." },
  "account-locked": { title: "Account Temporarily Locked", subtitle: "Your account has been locked due to too many failed login attempts. Please wait a few minutes and try again." },
  "too-many-requests": { title: "Too Many Requests", subtitle: "You've made too many attempts in a short period. Please wait a moment before trying again." },
  "no-access-authenticated": { title: "You don't have access", subtitle: "You're not allowed to view this page. If you think this is a mistake, contact your administrator." },
  "no-access-guest": { title: "You don't have access", subtitle: "You're not allowed to view this page. If you think this is a mistake, contact your administrator." },
  "service-unavailable": { title: "Service Unavailable", subtitle: "We're unable to connect to our servers right now. This is usually temporary - please wait a moment and try again." },
  "not-found": { title: "Page not found", subtitle: "The page you're looking for doesn't exist or may have moved." },
}

/**
 * Resolve a page's copy: the tenant's configured override when present and
 * non-empty, otherwise the shared default. A blank string in metadata is
 * treated as "not set" so clearing a field in the console restores the default
 * rather than rendering an empty heading.
 */
export function loginPageCopy(
  metadata: Record<string, unknown> | null | undefined,
  id: LoginPageId,
): LoginPageCopy {
  const fallback = DEFAULT_LOGIN_PAGE_COPY[id]
  const configured = metadata?.login_page_content
  if (!configured || typeof configured !== 'object') return fallback

  const page = (configured as Record<string, unknown>)[id]
  if (!page || typeof page !== 'object') return fallback

  const record = page as Record<string, unknown>
  return {
    title: readCopy(record.title) ?? fallback.title,
    subtitle: readCopy(record.subtitle) ?? fallback.subtitle,
  }
}

function readCopy(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  return trimmed === '' ? undefined : trimmed
}
