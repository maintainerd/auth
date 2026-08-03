export const LOGIN_PAGE_PREVIEW_IDS = [
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

export type LoginPagePreviewId = typeof LOGIN_PAGE_PREVIEW_IDS[number]

export const LOGIN_PAGE_PREVIEW_GROUPS = [
  "Sign-in",
  "Registration",
  "Recovery",
  "OAuth",
  "Account",
  "Status",
] as const

export type LoginPagePreviewGroup = typeof LOGIN_PAGE_PREVIEW_GROUPS[number]

export type LoginPageElement =
  | { type: "alert"; tone: "error" | "warning" | "info"; text: string }
  | { type: "button"; label: string; variant?: "primary" | "outline" | "ghost" }
  | { type: "checkbox"; label: string }
  | { type: "divider"; label: string }
  | { type: "field"; label: string; value?: string; kind?: "email" | "password" | "tel" | "code" | "text" }
  | { type: "link"; label: string }
  | { type: "readonly"; label: string; value: string }
  | { type: "scope-list"; items: string[] }
  | { type: "section"; title: string; description?: string }
  | { type: "select"; label: string; value: string }
  | { type: "tile-list"; columns?: 1 | 2; items: Array<{ title: string; description?: string; scopes?: string[]; actionLabel?: string }> }

export type LoginPageCopy = {
  title: string
  subtitle: string
}

export type LoginTemplateFormsConfig = {
  pages: LoginPagePreview[]
  imageUrl: string
}

export type LoginPagePreview = LoginPageCopy & {
  id: LoginPagePreviewId
  label: string
  group: LoginPagePreviewGroup
  elements: LoginPageElement[]
}

const accountTiles = [
  { title: "Profile", description: "Names & avatar" },
  { title: "Security", description: "Password & email" },
  { title: "Sessions", description: "Active sign-ins" },
  { title: "Two-Factor", description: "MFA settings" },
  { title: "Linked Accounts", description: "Social logins" },
  { title: "Preferences", description: "App settings" },
]

// Keep fields/states aligned to maintainerd-auth-identity because those flows are
// already wired there. Layout, ordering, and visual treatment remain owned by
// the console branding preview; the identity app will consume those designs later.
export const DEFAULT_LOGIN_PAGE_PREVIEWS: Record<LoginPagePreviewId, LoginPagePreview> = {
  login: {
    id: "login",
    label: "Login",
    group: "Sign-in",
    title: "Welcome back",
    subtitle: "Sign in to your account to continue.",
    elements: [
      { type: "button", label: "Continue with Google", variant: "outline" },
      { type: "button", label: "Continue with GitHub", variant: "outline" },
      { type: "divider", label: "or continue with email" },
      { type: "field", label: "Email", value: "you@company.com", kind: "email" },
      { type: "field", label: "Password", value: "********", kind: "password" },
      { type: "link", label: "Forgot password?" },
      { type: "button", label: "Sign in" },
      { type: "button", label: "Email me a sign-in link", variant: "outline" },
      { type: "link", label: "Create an account" },
    ],
  },
  "login-methods-loading": {
    id: "login-methods-loading",
    label: "Login: Loading Methods",
    group: "Sign-in",
    title: "Welcome back",
    subtitle: "Loading the sign-in methods available for this application.",
    elements: [
      { type: "section", title: "Loading sign-in methods...", description: "Available password and identity provider options are loading." },
      { type: "field", label: "Email", value: "you@company.com", kind: "email" },
      { type: "field", label: "Password", value: "********", kind: "password" },
      { type: "button", label: "Sign in" },
    ],
  },
  "login-methods-unavailable": {
    id: "login-methods-unavailable",
    label: "Login: No Methods",
    group: "Sign-in",
    title: "Welcome back",
    subtitle: "No sign-in methods are available for this application.",
    elements: [
      { type: "section", title: "No sign-in methods are available for this application." },
    ],
  },
  "login-error": {
    id: "login-error",
    label: "Login: Error",
    group: "Sign-in",
    title: "Welcome back",
    subtitle: "Sign in to your account to continue.",
    elements: [
      { type: "alert", tone: "error", text: "Invalid email or password." },
      { type: "field", label: "Email", value: "you@company.com", kind: "email" },
      { type: "field", label: "Password", value: "********", kind: "password" },
      { type: "link", label: "Forgot password?" },
      { type: "button", label: "Sign in" },
      { type: "button", label: "Email me a sign-in link", variant: "outline" },
      { type: "link", label: "Create an account" },
    ],
  },
  "login-magic-link-sent": {
    id: "login-magic-link-sent",
    label: "Login: Magic Link Sent",
    group: "Sign-in",
    title: "Check your email",
    subtitle: "A secure sign-in link will arrive shortly.",
    elements: [
      { type: "button", label: "Back to password sign in", variant: "outline" },
    ],
  },
  "login-mfa-code": {
    id: "login-mfa-code",
    label: "Login MFA: Code",
    group: "Sign-in",
    title: "Two-step verification",
    subtitle: "Confirm your second factor to finish signing in.",
    elements: [
      { type: "tile-list", items: [
        { title: "Authenticator app", description: "Use a six-digit code." },
        { title: "Passkey", description: "Use Face ID, Touch ID, Windows Hello, or your security key." },
        { title: "Text message", description: "Send code to my phone." },
        { title: "Email OTP", description: "Send code to my email." },
        { title: "Backup code", description: "Enter one saved recovery code." },
      ], columns: 1 },
      { type: "field", label: "Authenticator app code", value: "000000", kind: "code" },
      { type: "checkbox", label: "Trust this device — skip verification here next time" },
      { type: "button", label: "Verify" },
      { type: "button", label: "Cancel", variant: "ghost" },
    ],
  },
  "login-mfa-passkey": {
    id: "login-mfa-passkey",
    label: "Login MFA: Passkey",
    group: "Sign-in",
    title: "Two-step verification",
    subtitle: "Confirm your identity with a second factor.",
    elements: [
      { type: "section", title: "Passkey", description: "Use Face ID, Touch ID, Windows Hello, or your security key to confirm." },
      { type: "checkbox", label: "Trust this device — skip verification here next time" },
      { type: "button", label: "Use passkey" },
      { type: "button", label: "Cancel", variant: "ghost" },
    ],
  },
  "login-mfa-backup-code": {
    id: "login-mfa-backup-code",
    label: "Login MFA: Backup Code",
    group: "Sign-in",
    title: "Two-step verification",
    subtitle: "Enter one backup code to finish signing in.",
    elements: [
      { type: "field", label: "Backup code", value: "Enter one backup code", kind: "code" },
      { type: "checkbox", label: "Trust this device — skip verification here next time" },
      { type: "button", label: "Verify" },
      { type: "button", label: "Cancel", variant: "ghost" },
    ],
  },
  "login-mfa-no-methods": {
    id: "login-mfa-no-methods",
    label: "Login MFA: No Methods",
    group: "Sign-in",
    title: "Two-step verification",
    subtitle: "MFA is required but no supported factor is available. Contact your administrator.",
    elements: [
      { type: "button", label: "Back to login", variant: "ghost" },
    ],
  },
  "magic-link-verifying": {
    id: "magic-link-verifying",
    label: "Magic Link: Verifying",
    group: "Sign-in",
    title: "Signing you in",
    subtitle: "We're securely verifying your magic link.",
    elements: [],
  },
  "magic-link-success": {
    id: "magic-link-success",
    label: "Magic Link: Success",
    group: "Sign-in",
    title: "You're signed in",
    subtitle: "Your magic link was verified successfully. We're taking you to the app.",
    elements: [
      { type: "button", label: "Continue" },
      { type: "section", title: "Redirecting automatically in a few seconds..." },
    ],
  },
  "magic-link-mfa": {
    id: "magic-link-mfa",
    label: "Magic Link: MFA",
    group: "Sign-in",
    title: "Two-step verification",
    subtitle: "Your magic link was accepted. Confirm a different factor to finish signing in.",
    elements: [
      { type: "tile-list", items: [
        { title: "Authenticator app", description: "Use a six-digit code." },
        { title: "Passkey", description: "Use Face ID, Touch ID, Windows Hello, or your security key." },
        { title: "Text message", description: "Send code to my phone." },
        { title: "Backup code", description: "Enter one saved recovery code." },
      ], columns: 1 },
      { type: "button", label: "Send code to my phone", variant: "outline" },
      { type: "field", label: "Authenticator app code", value: "000000", kind: "code" },
      { type: "checkbox", label: "Trust this device — skip verification here next time" },
      { type: "button", label: "Verify" },
      { type: "button", label: "Cancel", variant: "ghost" },
    ],
  },
  "magic-link-error": {
    id: "magic-link-error",
    label: "Magic Link: Error",
    group: "Sign-in",
    title: "Magic link unavailable",
    subtitle: "The magic link is invalid or has expired. Please request a new one.",
    elements: [
      { type: "button", label: "Back to sign in" },
    ],
  },
  "sms-login-phone": {
    id: "sms-login-phone",
    label: "SMS Login: Phone",
    group: "Sign-in",
    title: "Sign in with SMS",
    subtitle: "Enter your phone number to receive a one-time code",
    elements: [
      { type: "link", label: "Back to login" },
      { type: "field", label: "Phone Number", value: "+1234567890", kind: "tel" },
      { type: "button", label: "Send Code" },
    ],
  },
  "sms-login-phone-required": {
    id: "sms-login-phone-required",
    label: "SMS Login: Phone Required",
    group: "Sign-in",
    title: "Sign in with SMS",
    subtitle: "Enter your phone number to receive a one-time code",
    elements: [
      { type: "link", label: "Back to login" },
      { type: "alert", tone: "warning", text: "Your phone must be verified before you can sign in. Enter your number to receive a code." },
      { type: "field", label: "Phone Number", value: "+1234567890", kind: "tel" },
      { type: "button", label: "Send Code" },
    ],
  },
  "sms-login-code": {
    id: "sms-login-code",
    label: "SMS Login: Code",
    group: "Sign-in",
    title: "Sign in with SMS",
    subtitle: "Enter the verification code sent to +1234567890",
    elements: [
      { type: "link", label: "Back to login" },
      { type: "readonly", label: "Phone Number", value: "+1234567890" },
      { type: "field", label: "Verification Code", value: "000000", kind: "code" },
      { type: "button", label: "Verify & Sign In" },
      { type: "button", label: "Resend code", variant: "ghost" },
    ],
  },
  registration: {
    id: "registration",
    label: "Registration",
    group: "Registration",
    title: "Create your account",
    subtitle: "Sign up to get started.",
    elements: [
      { type: "field", label: "Email", value: "you@company.com", kind: "email" },
      { type: "field", label: "Password", value: "********", kind: "password" },
      { type: "section", title: "Password requirements", description: "Tenant password policy checklist." },
      { type: "field", label: "Confirm password", value: "********", kind: "password" },
      { type: "checkbox", label: "I agree to the terms and privacy policy" },
      { type: "button", label: "Create account" },
      { type: "link", label: "Sign in" },
    ],
  },
  "registration-loading": {
    id: "registration-loading",
    label: "Registration: Loading",
    group: "Registration",
    title: "Loading registration",
    subtitle: "Preparing the registration experience.",
    elements: [
      { type: "section", title: "Loading registration context", description: "Tenant registration policy is loading." },
    ],
  },
  "registration-invalid-link": {
    id: "registration-invalid-link",
    label: "Registration: Invalid Link",
    group: "Registration",
    title: "This sign-up link is no longer valid",
    subtitle: "It may have been renamed, deactivated, or replaced. Ask whoever sent it for an up-to-date link.",
    elements: [
      { type: "button", label: "Sign in instead", variant: "outline" },
    ],
  },
  "registration-requirements-warning": {
    id: "registration-requirements-warning",
    label: "Registration: Requirements Warning",
    group: "Registration",
    title: "Create your account",
    subtitle: "Sign up to get started.",
    elements: [
      { type: "alert", tone: "warning", text: "We could not confirm this sign-up link's requirements. You can continue - anything still missing will be flagged when you submit." },
      { type: "field", label: "Email", value: "you@company.com", kind: "email" },
      { type: "field", label: "Password", value: "********", kind: "password" },
      { type: "section", title: "Password requirements", description: "Tenant password policy checklist." },
      { type: "field", label: "Confirm password", value: "********", kind: "password" },
      { type: "checkbox", label: "I agree to the terms and privacy policy" },
      { type: "button", label: "Create account" },
      { type: "button", label: "Retry", variant: "outline" },
      { type: "link", label: "Sign in" },
    ],
  },
  "invite-accept": {
    id: "invite-accept",
    label: "Invite: Accept",
    group: "Registration",
    title: "Accept your invitation",
    subtitle: "Set up your password to complete registration.",
    elements: [
      { type: "readonly", label: "Email", value: "alex@company.com" },
      { type: "field", label: "Password", value: "********", kind: "password" },
      { type: "section", title: "Password requirements", description: "Tenant password policy checklist." },
      { type: "field", label: "Confirm password", value: "********", kind: "password" },
      { type: "checkbox", label: "I agree to the terms and privacy policy" },
      { type: "button", label: "Create account" },
      { type: "link", label: "Sign in" },
    ],
  },
  "invite-session-mismatch": {
    id: "invite-session-mismatch",
    label: "Invite: Session Mismatch",
    group: "Registration",
    title: "Accept your invitation",
    subtitle: "You're signed in as sam@company.com. This invitation is for alex@company.com. Sign out to accept it.",
    elements: [
      { type: "button", label: "Sign out to continue" },
    ],
  },
  "invite-invalid": {
    id: "invite-invalid",
    label: "Invite: Invalid Link",
    group: "Registration",
    title: "Invalid invite link",
    subtitle: "This invite link is missing the email parameter. Please request a new invitation.",
    elements: [
      { type: "button", label: "Back to sign in" },
    ],
  },
  "profile-form": {
    id: "profile-form",
    label: "Profile: Form",
    group: "Registration",
    title: "Complete your profile",
    subtitle: "Just a few details to get started.",
    elements: [
      { type: "field", label: "First Name", value: "John" },
      { type: "field", label: "Last Name", value: "Doe" },
      { type: "select", label: "Gender", value: "Select gender" },
      { type: "button", label: "Create Profile" },
    ],
  },
  "profile-success": {
    id: "profile-success",
    label: "Profile: Success",
    group: "Registration",
    title: "All set!",
    subtitle: "Your profile has been created. You can now access the app.",
    elements: [
      { type: "button", label: "Continue" },
      { type: "section", title: "Redirecting automatically in a few seconds..." },
    ],
  },
  "verify-email-form": {
    id: "verify-email-form",
    label: "Email Verification: Form",
    group: "Registration",
    title: "Verify your email",
    subtitle: "Enter the code sent to alex@company.com.",
    elements: [
      { type: "field", label: "Verification code", value: "000000", kind: "code" },
      { type: "button", label: "Verify email" },
      { type: "button", label: "Didn't receive a code? Resend", variant: "ghost" },
      { type: "button", label: "Back to login", variant: "ghost" },
    ],
  },
  "verify-email-success": {
    id: "verify-email-success",
    label: "Email Verification: Success",
    group: "Registration",
    title: "Email verified",
    subtitle: "Your email has been confirmed. You can now sign in to your account.",
    elements: [
      { type: "button", label: "Sign in" },
    ],
  },
  "forgot-password-form": {
    id: "forgot-password-form",
    label: "Forgot Password: Form",
    group: "Recovery",
    title: "Forgot your password?",
    subtitle: "Enter your email and we'll send reset instructions.",
    elements: [
      { type: "field", label: "Email", value: "you@company.com", kind: "email" },
      { type: "button", label: "Send reset instructions" },
      { type: "link", label: "Back to login" },
    ],
  },
  "forgot-password-sent": {
    id: "forgot-password-sent",
    label: "Forgot Password: Sent",
    group: "Recovery",
    title: "Check your email",
    subtitle: "If an account exists with this email, password reset instructions will arrive shortly.",
    elements: [
      { type: "button", label: "Back to login" },
    ],
  },
  "reset-password-form": {
    id: "reset-password-form",
    label: "Reset Password: Form",
    group: "Recovery",
    title: "Reset your password",
    subtitle: "Enter your new password below.",
    elements: [
      { type: "field", label: "New password", value: "********", kind: "password" },
      { type: "section", title: "Password requirements", description: "Tenant password policy checklist." },
      { type: "field", label: "Confirm password", value: "********", kind: "password" },
      { type: "button", label: "Reset password" },
    ],
  },
  "reset-password-success": {
    id: "reset-password-success",
    label: "Reset Password: Success",
    group: "Recovery",
    title: "Password reset",
    subtitle: "You can now sign in with your new password.",
    elements: [
      { type: "button", label: "Go to login" },
    ],
  },
  "reset-password-invalid": {
    id: "reset-password-invalid",
    label: "Reset Password: Invalid",
    group: "Recovery",
    title: "Invalid reset link",
    subtitle: "This password reset link is invalid or has expired. Please request a new one.",
    elements: [
      { type: "button", label: "Request new reset link" },
      { type: "button", label: "Back to login", variant: "outline" },
    ],
  },
  "backup-code-recovery": {
    id: "backup-code-recovery",
    label: "Backup Code Recovery",
    group: "Recovery",
    title: "Backup Code Recovery",
    subtitle: "Enter one of your saved backup codes to regain access to your account.",
    elements: [
      { type: "link", label: "Back to login" },
      { type: "field", label: "Backup Code", value: "Enter your backup code", kind: "code" },
      { type: "button", label: "Recover Account" },
    ],
  },
  "phone-verification-loading": {
    id: "phone-verification-loading",
    label: "Phone Verification: Loading",
    group: "Account",
    title: "Verify Phone",
    subtitle: "Confirm your phone number with a one-time code sent by SMS.",
    elements: [
      { type: "link", label: "Back" },
      { type: "section", title: "Loading..." },
    ],
  },
  "phone-verification-send": {
    id: "phone-verification-send",
    label: "Phone Verification: Send Code",
    group: "Account",
    title: "Verify Phone",
    subtitle: "Confirm your phone number with a one-time code sent by SMS.",
    elements: [
      { type: "link", label: "Back" },
      { type: "field", label: "Phone number", value: "+15551234567", kind: "tel" },
      { type: "button", label: "Send code" },
    ],
  },
  "phone-verification-code": {
    id: "phone-verification-code",
    label: "Phone Verification: Enter Code",
    group: "Account",
    title: "Verify Phone",
    subtitle: "Confirm your phone number with a one-time code sent by SMS.",
    elements: [
      { type: "link", label: "Back" },
      { type: "readonly", label: "Phone number", value: "+15551234567" },
      { type: "field", label: "Verification code", value: "000000", kind: "code" },
      { type: "button", label: "Verify" },
      { type: "button", label: "Resend", variant: "outline" },
    ],
  },
  "phone-verification-verified": {
    id: "phone-verification-verified",
    label: "Phone Verification: Verified",
    group: "Account",
    title: "Your phone is verified",
    subtitle: "+15551234567",
    elements: [
      { type: "link", label: "Back" },
      { type: "section", title: "Your phone is verified", description: "+15551234567" },
    ],
  },
  "account-link-invalid": {
    id: "account-link-invalid",
    label: "Account Link: Invalid",
    group: "Sign-in",
    title: "Invalid Link",
    subtitle: "This account link URL is incomplete or has expired.",
    elements: [
      { type: "button", label: "Back to sign in", variant: "outline" },
    ],
  },
  "account-link-loading": {
    id: "account-link-loading",
    label: "Account Link: Loading",
    group: "Sign-in",
    title: "Link your Google account",
    subtitle: "Checking sign-in status...",
    elements: [],
  },
  "account-link-signed-out": {
    id: "account-link-signed-out",
    label: "Account Link: Signed Out",
    group: "Sign-in",
    title: "Link your Google account",
    subtitle: "Your Google account uses alex@company.com which is already registered. Sign in to confirm you own this account and link them.",
    elements: [
      { type: "section", title: "You need to be signed in to confirm this link." },
      { type: "button", label: "Sign in to continue" },
      { type: "link", label: "Back to sign in" },
    ],
  },
  "account-link-confirm": {
    id: "account-link-confirm",
    label: "Account Link: Confirm",
    group: "Sign-in",
    title: "Link your Google account",
    subtitle: "Your Google account uses alex@company.com which is already registered. Sign in to confirm you own this account and link them.",
    elements: [
      { type: "button", label: "Link Google account" },
      { type: "link", label: "Cancel" },
    ],
  },
  "account-link-success": {
    id: "account-link-success",
    label: "Account Link: Success",
    group: "Sign-in",
    title: "Accounts linked!",
    subtitle: "Redirecting you to the app...",
    elements: [],
  },
  "oauth-authorize-loading": {
    id: "oauth-authorize-loading",
    label: "OAuth Authorize: Loading",
    group: "OAuth",
    title: "Authorizing",
    subtitle: "Preparing the secure redirect.",
    elements: [],
  },
  "oauth-authorize-error": {
    id: "oauth-authorize-error",
    label: "OAuth Authorize: Error",
    group: "OAuth",
    title: "Authorization unavailable",
    subtitle: "Authorization failed.",
    elements: [
      { type: "button", label: "Back to sign in" },
    ],
  },
  "oauth-consent-loading": {
    id: "oauth-consent-loading",
    label: "OAuth Consent: Loading",
    group: "OAuth",
    title: "Loading request",
    subtitle: "",
    elements: [],
  },
  "oauth-consent-request": {
    id: "oauth-consent-request",
    label: "OAuth Consent: Request",
    group: "OAuth",
    title: "Example App",
    subtitle: "wants access to your account.",
    elements: [
      { type: "scope-list", items: ["openid", "profile", "email"] },
      { type: "button", label: "Allow" },
      { type: "button", label: "Deny", variant: "outline" },
    ],
  },
  "oauth-consent-error": {
    id: "oauth-consent-error",
    label: "OAuth Consent: Error",
    group: "OAuth",
    title: "Consent unavailable",
    subtitle: "Consent request unavailable.",
    elements: [
      { type: "button", label: "Back", variant: "outline" },
    ],
  },
  "oauth-device-authorize": {
    id: "oauth-device-authorize",
    label: "OAuth Device: Authorize",
    group: "OAuth",
    title: "Authorize device",
    subtitle: "Enter the code shown on your device.",
    elements: [
      { type: "field", label: "User code", value: "ABCD-EFGH", kind: "code" },
      { type: "button", label: "Approve" },
      { type: "button", label: "Deny", variant: "outline" },
    ],
  },
  "oauth-device-approved": {
    id: "oauth-device-approved",
    label: "OAuth Device: Approved",
    group: "OAuth",
    title: "Device approved",
    subtitle: "",
    elements: [
      { type: "button", label: "Continue" },
    ],
  },
  "oauth-device-denied": {
    id: "oauth-device-denied",
    label: "OAuth Device: Denied",
    group: "OAuth",
    title: "Device denied",
    subtitle: "",
    elements: [
      { type: "button", label: "Continue" },
    ],
  },
  "oauth-ciba-confirm": {
    id: "oauth-ciba-confirm",
    label: "OAuth CIBA: Confirm",
    group: "OAuth",
    title: "Confirm sign in",
    subtitle: "Confirm this sign-in request from Example App.",
    elements: [
      { type: "button", label: "Approve" },
      { type: "button", label: "Deny", variant: "outline" },
    ],
  },
  "oauth-ciba-error": {
    id: "oauth-ciba-error",
    label: "OAuth CIBA: Error",
    group: "OAuth",
    title: "Confirm sign in",
    subtitle: "",
    elements: [
      { type: "alert", tone: "error", text: "Authentication request is missing." },
      { type: "button", label: "Approve" },
      { type: "button", label: "Deny", variant: "outline" },
    ],
  },
  "oauth-ciba-approved": {
    id: "oauth-ciba-approved",
    label: "OAuth CIBA: Approved",
    group: "OAuth",
    title: "Request approved",
    subtitle: "",
    elements: [
      { type: "button", label: "Continue" },
    ],
  },
  "oauth-ciba-denied": {
    id: "oauth-ciba-denied",
    label: "OAuth CIBA: Denied",
    group: "OAuth",
    title: "Request denied",
    subtitle: "",
    elements: [
      { type: "button", label: "Continue" },
    ],
  },
  "oauth-grants-loading": {
    id: "oauth-grants-loading",
    label: "Connected Apps: Loading",
    group: "OAuth",
    title: "Connected apps",
    subtitle: "Review applications you have authorized.",
    elements: [
      { type: "section", title: "Loading" },
    ],
  },
  "oauth-grants-empty": {
    id: "oauth-grants-empty",
    label: "Connected Apps: Empty",
    group: "OAuth",
    title: "Connected apps",
    subtitle: "Review applications you have authorized.",
    elements: [
      { type: "section", title: "No connected applications." },
    ],
  },
  "oauth-grants-list": {
    id: "oauth-grants-list",
    label: "Connected Apps: List",
    group: "OAuth",
    title: "Connected apps",
    subtitle: "Review applications you have authorized.",
    elements: [
      { type: "tile-list", items: [
        { title: "Example App", scopes: ["openid", "profile", "email"], actionLabel: "Revoke access for Example App" },
        { title: "Reporting Portal", scopes: ["openid", "profile"], actionLabel: "Revoke access for Reporting Portal" },
      ], columns: 1 },
    ],
  },
  "oauth-grants-error": {
    id: "oauth-grants-error",
    label: "Connected Apps: Error",
    group: "OAuth",
    title: "Connected apps",
    subtitle: "Review applications you have authorized.",
    elements: [
      { type: "alert", tone: "error", text: "Failed to load connected applications." },
    ],
  },
  "oauth-end-session": {
    id: "oauth-end-session",
    label: "End Session",
    group: "OAuth",
    title: "Signing out",
    subtitle: "Completing the logout request.",
    elements: [],
  },
  "login-success": {
    id: "login-success",
    label: "Login Success",
    group: "Status",
    title: "You're signed in",
    subtitle: "You can continue from this session or sign out when you are done.",
    elements: [
      { type: "tile-list", items: accountTiles },
      { type: "button", label: "Sign out", variant: "outline" },
      { type: "link", label: "Delete account" },
    ],
  },
  "login-success-mfa-nudge": {
    id: "login-success-mfa-nudge",
    label: "Login Success: MFA Nudge",
    group: "Status",
    title: "You're signed in",
    subtitle: "You can continue from this session or sign out when you are done.",
    elements: [
      { type: "section", title: "Add an extra layer of security", description: "Your account has no two-factor authentication set up. Enable it to better protect your account." },
      { type: "button", label: "Set up MFA" },
      { type: "button", label: "Not now", variant: "ghost" },
      { type: "tile-list", items: accountTiles },
      { type: "button", label: "Sign out", variant: "outline" },
      { type: "link", label: "Delete account" },
    ],
  },
  "account-erasure-request": {
    id: "account-erasure-request",
    label: "Account Erasure: Request",
    group: "Account",
    title: "Request account deletion",
    subtitle: "Under GDPR Article 17 you have the right to have your personal data erased.",
    elements: [
      { type: "link", label: "Back" },
      { type: "alert", tone: "warning", text: "All your personal data will be permanently anonymised. This process begins after a 30-day window and cannot be undone." },
      { type: "button", label: "Request data deletion" },
    ],
  },
  "account-erasure-confirm": {
    id: "account-erasure-confirm",
    label: "Account Erasure: Confirm",
    group: "Account",
    title: "Request account deletion",
    subtitle: "Under GDPR Article 17 you have the right to have your personal data erased.",
    elements: [
      { type: "link", label: "Back" },
      { type: "alert", tone: "warning", text: "All your personal data will be permanently anonymised. This action cannot be undone." },
      { type: "section", title: "Are you sure? This cannot be undone." },
      { type: "button", label: "Yes, delete my data" },
      { type: "button", label: "Cancel", variant: "outline" },
    ],
  },
  "account-erasure-error": {
    id: "account-erasure-error",
    label: "Account Erasure: Error",
    group: "Account",
    title: "Request account deletion",
    subtitle: "Under GDPR Article 17 you have the right to have your personal data erased.",
    elements: [
      { type: "link", label: "Back" },
      { type: "alert", tone: "error", text: "Request failed. Please try again." },
      { type: "button", label: "Request data deletion" },
    ],
  },
  "account-erasure-success": {
    id: "account-erasure-success",
    label: "Account Erasure: Success",
    group: "Account",
    title: "Request submitted",
    subtitle: "Your data will be anonymised within 30 days. You will continue to have access to your account until then.",
    elements: [
      { type: "link", label: "Back to account" },
    ],
  },
  "account-locked": {
    id: "account-locked",
    label: "Account Locked",
    group: "Status",
    title: "Account Temporarily Locked",
    subtitle: "Your account has been locked due to too many failed login attempts. Please wait a few minutes and try again.",
    elements: [
      { type: "button", label: "Back to Login", variant: "outline" },
      { type: "link", label: "backup code recovery" },
    ],
  },
  "too-many-requests": {
    id: "too-many-requests",
    label: "Too Many Requests",
    group: "Status",
    title: "Too Many Requests",
    subtitle: "You've made too many attempts in a short period. Please wait a moment before trying again.",
    elements: [
      { type: "button", label: "Back to Login", variant: "outline" },
    ],
  },
  "no-access-authenticated": {
    id: "no-access-authenticated",
    label: "No Access: Authenticated",
    group: "Status",
    title: "You don't have access",
    subtitle: "You're not allowed to view this page. If you think this is a mistake, contact your administrator.",
    elements: [
      { type: "button", label: "Back to the app" },
    ],
  },
  "no-access-guest": {
    id: "no-access-guest",
    label: "No Access: Guest",
    group: "Status",
    title: "You don't have access",
    subtitle: "You're not allowed to view this page. If you think this is a mistake, contact your administrator.",
    elements: [
      { type: "button", label: "Back to sign in" },
    ],
  },
  "service-unavailable": {
    id: "service-unavailable",
    label: "Service Unavailable",
    group: "Status",
    title: "Service Unavailable",
    subtitle: "We're unable to connect to our servers right now. This is usually temporary - please wait a moment and try again.",
    elements: [
      { type: "button", label: "Try again" },
    ],
  },
  "not-found": {
    id: "not-found",
    label: "Page Not Found",
    group: "Status",
    title: "Page not found",
    subtitle: "The page you're looking for doesn't exist or may have moved.",
    elements: [
      { type: "button", label: "Back to sign in" },
    ],
  },
}

export function loginPagePreviewsFromMetadata(
  metadata: Record<string, unknown> | null | undefined,
): LoginPagePreview[] {
  const configured = metadata?.login_page_content
  const configuredByPage = configured && typeof configured === "object"
    ? configured as Record<string, unknown>
    : {}

  return LOGIN_PAGE_PREVIEW_IDS.map((id) => ({
    ...DEFAULT_LOGIN_PAGE_PREVIEWS[id],
    ...readPageOverrides(configuredByPage[id]),
  }))
}

export function loginPageContentMetadata(
  currentMetadata: Record<string, unknown> | null | undefined,
  page: LoginPagePreview,
): Record<string, unknown> {
  return loginPageContentCollectionMetadata(currentMetadata, [page])
}

export function loginPageContentCollectionMetadata(
  currentMetadata: Record<string, unknown> | null | undefined,
  pages: LoginPagePreview[],
  imageUrl?: string,
): Record<string, unknown> {
  const existing = currentMetadata?.login_page_content
  const existingContent = existing && typeof existing === "object"
    ? existing as Record<string, unknown>
    : {}

  const nextContent = pages.reduce<Record<string, { title: string; subtitle: string }>>(
    (acc, page) => ({
      ...acc,
      [page.id]: {
        title: page.title.trim(),
        subtitle: page.subtitle.trim(),
      },
    }),
    {},
  )

  const nextMetadata: Record<string, unknown> = {
    ...(currentMetadata ?? {}),
    login_page_content: {
      ...existingContent,
      ...nextContent,
    },
  }

  if (imageUrl !== undefined) {
    nextMetadata.login_template_image_url = imageUrl.trim()
  }

  return nextMetadata
}

export function loginTemplateImageUrlFromMetadata(
  metadata: Record<string, unknown> | null | undefined,
): string {
  const value = metadata?.login_template_image_url
  return typeof value === "string" ? value : ""
}

function readPageOverrides(value: unknown): Partial<LoginPageCopy> {
  if (!value || typeof value !== "object") return {}
  const record = value as Record<string, unknown>
  const overrides: Partial<LoginPageCopy> = {}
  const title = readString(record.title)
  const subtitle = readString(record.subtitle)
  if (title !== undefined) overrides.title = title
  if (subtitle !== undefined) overrides.subtitle = subtitle
  return overrides
}

function readString(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined
}
