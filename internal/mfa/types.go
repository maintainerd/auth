package mfa

// ──────────────────────────────────────────────────────────────────────────────
// TOTP
// ──────────────────────────────────────────────────────────────────────────────

// TOTPEnrollResponseDTO is returned when the user starts TOTP enrollment.
// The frontend should render QR code from QRCodeURL and also show Secret
// for manual entry.
type TOTPEnrollResponseDTO struct {
	Secret    string `json:"secret"`      // Base32 TOTP secret for manual entry
	QRCodeURL string `json:"qr_code_url"` // otpauth:// URI for QR code rendering
}

// TOTPVerifyRequestDTO confirms TOTP enrollment or validates a TOTP code.
type TOTPVerifyRequestDTO struct {
	Code string `json:"code"` // 6-digit TOTP code
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
	IsTOTPEnabled     bool                           `json:"is_totp_enabled"`
	IsWebAuthnEnabled bool                           `json:"is_webauthn_enabled"`
	IsSMSEnabled      bool                           `json:"is_sms_available"`
	BackupCodesCount  int                            `json:"backup_codes_count"`
	WebAuthnKeys      []WebAuthnCredentialSummaryDTO `json:"webauthn_keys,omitempty"`
	MFAEnabledAt      *string                        `json:"mfa_enabled_at,omitempty"`
}

// ──────────────────────────────────────────────────────────────────────────────
// WebAuthn
// ──────────────────────────────────────────────────────────────────────────────

// WebAuthnCredentialSummaryDTO is a lightweight view of a registered credential.
type WebAuthnCredentialSummaryDTO struct {
	CredentialUUID string  `json:"credential_uuid"`
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
type StepUpVerifyRequestDTO struct {
	ChallengeToken string `json:"challenge_token"`
	Method         string `json:"method"` // "totp" | "sms" | "webauthn" | "backup_code"
	Code           string `json:"code,omitempty"`
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

// ──────────────────────────────────────────────────────────────────────────────
// WebAuthn Download
// ──────────────────────────────────────────────────────────────────────────────

type WebAuthnCredentialDownloadDTO struct {
	CredentialUUID   string `json:"credential_uuid"`
	Name             string `json:"name"`
	CredentialKeyID  string `json:"credential_key_id"`
	PublicKeyBase64  string `json:"public_key_base64"`
	AAGUID           string `json:"aaguid,omitempty"`
	Transport        string `json:"transport,omitempty"`
	IsBackupEligible bool   `json:"is_backup_eligible"`
	IsBackupState    bool   `json:"is_backup_state"`
	CreatedAt        string `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Admin
// ──────────────────────────────────────────────────────────────────────────────

// MFAAdminResetRequestDTO is an optional body for the admin MFA reset endpoint.
type MFAAdminResetRequestDTO struct {
	Reason string `json:"reason,omitempty"`
}
