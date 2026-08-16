package mfa

import "encoding/json"

// ──────────────────────────────────────────────────────────────────────────────
// TOTP
// ──────────────────────────────────────────────────────────────────────────────

// TOTPEnrollResponseDTO is returned when the user starts TOTP enrollment.
// The frontend should render QR code from QRCodeURL and also show Secret
// for manual entry.
type TOTPEnrollResponseDTO struct {
	Secret    string `json:"secret"`      // Base32 TOTP secret for manual entry
	QRCodeURL string `json:"qr_code_url"` // otpauth:// URI for QR code rendering
	// Digits is how long the generated codes are — 6 or 8, per the tenant's
	// totp_digits policy, which the key above was generated with.
	//
	// The client cannot assume 6. A tenant on 8 gets an authenticator showing 8
	// digits, and a UI that caps its input at 6 makes the code impossible to type
	// and enrolment impossible to complete.
	Digits int `json:"digits"`
	// PeriodSeconds is the code rotation window, for "expires in" hints.
	PeriodSeconds int `json:"period_seconds"`
}

// TOTPVerifyRequestDTO confirms TOTP enrollment or validates a TOTP code.
type TOTPVerifyRequestDTO struct {
	Code string `json:"code"` // TOTP code; 6 or 8 digits per the tenant's policy
}

// TOTPStatusResponseDTO reports the user's current TOTP state.
type TOTPStatusResponseDTO struct {
	IsEnabled  bool    `json:"is_enabled"`
	EnrolledAt *string `json:"enrolled_at,omitempty"`
	LastUsedAt *string `json:"last_used_at,omitempty"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Backup Codes
// ──────────────────────────────────────────────────────────────────────────────

// BackupCodesResponseDTO is returned when new backup codes are generated.
type BackupCodesResponseDTO struct {
	Codes []string `json:"codes"` // Plaintext codes — shown once, never again
}

// BackupCodeVerifyRequestDTO submits a backup code for verification.
type BackupCodeVerifyRequestDTO struct {
	Code string `json:"code"`
}

// ──────────────────────────────────────────────────────────────────────────────
// MFA Status
// ──────────────────────────────────────────────────────────────────────────────

// MFAStatusResponseDTO describes all enrolled MFA factors for a user.
type MFAStatusResponseDTO struct {
	IsTOTPEnabled      bool                           `json:"is_totp_enabled"`
	IsWebAuthnEnabled  bool                           `json:"is_webauthn_enabled"`
	IsSMSEnabled       bool                           `json:"is_sms_available"`
	IsEmailOTPEnabled  bool                           `json:"is_email_otp_available"`
	BackupCodesCount   int                            `json:"backup_codes_count"`
	WebAuthnKeys       []WebAuthnCredentialSummaryDTO `json:"webauthn_keys,omitempty"`
	FirstMFAEnrolledAt *string                        `json:"mfa_enabled_at,omitempty"`

	// AllowedMethods is the tenant's POLICY: which factors a user may enrol at
	// all. Everything above it describes what this user has already set up.
	//
	// The two are different questions and the UI needs both. Without the policy a
	// client can only render every factor the product supports, so a tenant that
	// disabled SMS still shows "Text message" in its enrolment list and a user who
	// picks it gets refused at the end of the flow. Never empty in practice — a
	// tenant with no allowed methods cannot enrol anyone.
	AllowedMethods []string `json:"allowed_methods"`
	// MFARequired mirrors the policy's enforcement mode so a client can tell
	// "optional, none set up" from "required, none set up yet".
	MFARequired bool `json:"mfa_required"`
	// TOTPDigits is the code length this tenant's authenticator codes use, 6 or
	// 8. Carried here as well as on enrolment because the LOGIN second step has
	// to size its input for an already-enrolled user, and never sees the
	// enrolment response.
	TOTPDigits int `json:"totp_digits"`
}

// ──────────────────────────────────────────────────────────────────────────────
// WebAuthn
// ──────────────────────────────────────────────────────────────────────────────

// WebAuthnCredentialSummaryDTO is a lightweight view of a registered credential.
type WebAuthnCredentialSummaryDTO struct {
	CredentialUUID string  `json:"credential_id"`
	Name           string  `json:"name"`
	Transport      string  `json:"transport,omitempty"`
	LastUsedAt     *string `json:"last_used_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

// WebAuthnNameRequestDTO allows a user to rename a credential.
type WebAuthnNameRequestDTO struct {
	Name string `json:"name"`
}

// ──────────────────────────────────────────────────────────────────────────────
// MFA Policy (per tenant / pool — admin)
// ──────────────────────────────────────────────────────────────────────────────

// MFAPolicyDTO is the shape stored in secpolicy.SecuritySetting.MFAConfig.
type MFAPolicyDTO struct {
	// Required means all users must set up at least one MFA factor to log in.
	Required bool `json:"required"`
	// AllowedMethods is the list of accepted MFA methods: "totp", "sms", "webauthn", "backup_code".
	AllowedMethods []string `json:"allowed_methods"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Step-up authentication
// ──────────────────────────────────────────────────────────────────────────────

// StepUpChallengeResponseDTO is returned when step-up authentication is required.
type StepUpChallengeResponseDTO struct {
	ChallengeToken string   `json:"challenge_token"` // Short-lived JWT referencing the original request
	AllowedMethods []string `json:"allowed_methods"` // Which factors the user may use
}

// StepUpVerifyRequestDTO submits the completed second-factor proof.
//
// Code carries the typed proof for "totp" | "sms" | "backup_code".
// Assertion carries the raw WebAuthn assertion (the publicKeyCredential JSON
// produced by navigator.credentials.get) for the "webauthn" method; the
// preceding /mfa/webauthn/auth/begin call establishes the matching session.
type StepUpVerifyRequestDTO struct {
	ChallengeToken string          `json:"challenge_token"`
	Method         string          `json:"method"` // "totp" | "sms" | "webauthn" | "backup_code"
	Code           string          `json:"code,omitempty"`
	Assertion      json.RawMessage `json:"assertion,omitempty"`
}

// StepUpVerifyResponseDTO is returned when step-up succeeds.
type StepUpVerifyResponseDTO struct {
	AccessToken string `json:"access_token"` // New token with elevated acr
	ExpiresIn   int64  `json:"expires_in"`
}

// ──────────────────────────────────────────────────────────────────────────────
// SMS MFA enrollment
// ──────────────────────────────────────────────────────────────────────────────

type SMSEnrollRequestDTO struct {
	Phone string `json:"phone"`
}
type SMSVerifyRequestDTO struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}
type EmailOTPEnrollRequestDTO struct {
	Email string `json:"email"`
}
type EmailOTPVerifyRequestDTO struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// ──────────────────────────────────────────────────────────────────────────────
// WebAuthn Download
// ──────────────────────────────────────────────────────────────────────────────

type WebAuthnCredentialDownloadDTO struct {
	CredentialUUID   string `json:"credential_id"`
	Name             string `json:"name"`
	CredentialKeyID  string `json:"credential_key_id"`
	PublicKeyBase64  string `json:"public_key_base64"`
	AAGUID           string `json:"aaguid,omitempty"`
	Transport        string `json:"transport,omitempty"`
	IsBackupEligible bool   `json:"is_backup_eligible"`
	IsBackupActive   bool   `json:"is_backup_active"`
	CreatedAt        string `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Admin
// ──────────────────────────────────────────────────────────────────────────────

// MFAAdminResetRequestDTO is an optional body for the admin MFA reset endpoint.
type MFAAdminResetRequestDTO struct {
	Reason string `json:"reason,omitempty"`
}
