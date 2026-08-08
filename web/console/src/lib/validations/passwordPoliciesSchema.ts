import * as yup from 'yup'

export const PASSWORD_POLICY_LIMITS = {
  // Mirrors internal/secpolicy/validation_setting.go. The backend rejects
  // anything outside these bounds; declaring them here only saves a round trip.
  // minLength stays at 1 to match the backend floor — see the comment on
  // minPasswordMinLength there for why raising it is a breaking change. The
  // recommended minimum lives in PASSWORD_POLICY_DEFAULTS and the field hint.
  minLength: 1,
  maxLength: 128,
  maxHistoryCount: 24,
  maxAgeDays: 3650,
  maxTemporaryPasswordValidityHours: 720,
} as const

/**
 * The shipped tenant baseline, mirroring internal/secpolicy/defaults_setting.go.
 *
 * This is the NIST SP 800-63B posture: length over composition. The four
 * `require_*` rules default to FALSE on purpose — mandatory character classes
 * push people toward "Password1!" and measurably reduce entropy. Length,
 * breach screening and the common-password list do the real work.
 */
export const PASSWORD_POLICY_DEFAULTS = {
  min_length: 12,
  max_length: 128,
  require_uppercase: false,
  require_lowercase: false,
  require_number: false,
  require_symbol: false,
  reject_common_passwords: true,
  check_hibp: true,
  password_history_count: 5,
  max_age_days: 0,
  temporary_password_validity_hours: 72,
  hash_algorithm: 'argon2id',
  min_strength_score: 2,
} as const

export const passwordPoliciesSchema = yup.object({
  min_length: yup
    .number()
    .integer('Must be a whole number')
    .required('Minimum length is required')
    .min(PASSWORD_POLICY_LIMITS.minLength, `Must be at least ${PASSWORD_POLICY_LIMITS.minLength}`)
    .max(PASSWORD_POLICY_LIMITS.maxLength, `Cannot exceed ${PASSWORD_POLICY_LIMITS.maxLength}`),
  max_length: yup
    .number()
    .integer('Must be a whole number')
    .required('Maximum length is required')
    .min(64, 'Must be at least 64')
    .max(PASSWORD_POLICY_LIMITS.maxLength, `Cannot exceed ${PASSWORD_POLICY_LIMITS.maxLength}`)
    .test(
      'not-less-than-minimum',
      'Maximum length must be greater than or equal to minimum length',
      function (value) {
        return value >= this.parent.min_length
      },
    ),
  require_uppercase: yup.boolean().required(),
  require_lowercase: yup.boolean().required(),
  require_number: yup.boolean().required(),
  require_symbol: yup.boolean().required(),
  reject_common_passwords: yup.boolean().required(),
  check_hibp: yup.boolean().required(),
  password_history_count: yup
    .number()
    .integer('Must be a whole number')
    .required()
    .min(0, 'Cannot be negative')
    .max(PASSWORD_POLICY_LIMITS.maxHistoryCount, `Cannot exceed ${PASSWORD_POLICY_LIMITS.maxHistoryCount}`),
  max_age_days: yup
    .number()
    .integer('Must be a whole number')
    .required()
    .min(0, 'Cannot be negative')
    .max(PASSWORD_POLICY_LIMITS.maxAgeDays, `Cannot exceed ${PASSWORD_POLICY_LIMITS.maxAgeDays} days`),
  temporary_password_validity_hours: yup
    .number()
    .integer('Must be a whole number')
    .required()
    .min(1, 'Must be at least 1 hour')
    .max(
      PASSWORD_POLICY_LIMITS.maxTemporaryPasswordValidityHours,
      `Cannot exceed ${PASSWORD_POLICY_LIMITS.maxTemporaryPasswordValidityHours} hours`,
    ),
  hash_algorithm: yup
    .string()
    .required('Hashing algorithm is required')
    .oneOf(['argon2id', 'bcrypt', 'scrypt', 'pbkdf2'], 'Invalid algorithm'),
  min_strength_score: yup
    .number()
    .integer('Must be a whole number')
    .required()
    .min(0, 'Must be 0–4')
    .max(4, 'Must be 0–4'),
}).required()

export type PasswordPoliciesFormData = yup.InferType<typeof passwordPoliciesSchema>
