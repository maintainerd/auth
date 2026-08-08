import { describe, expect, it } from 'vitest'
import { PASSWORD_POLICY_LIMITS, passwordPoliciesSchema } from './passwordPoliciesSchema'

const validPolicy = {
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
}

describe('passwordPoliciesSchema', () => {
  it('accepts the default policy', async () => {
    await expect(passwordPoliciesSchema.validate(validPolicy)).resolves.toMatchObject(validPolicy)
  })

  it('rejects a maximum length below the minimum length', async () => {
    await expect(
      passwordPoliciesSchema.validate({ ...validPolicy, min_length: 100, max_length: 64 }),
    ).rejects.toThrow('Maximum length must be greater than or equal to minimum length')
  })

  it.each([
    ['password_history_count', PASSWORD_POLICY_LIMITS.maxHistoryCount + 1],
    ['max_age_days', PASSWORD_POLICY_LIMITS.maxAgeDays + 1],
    [
      'temporary_password_validity_hours',
      PASSWORD_POLICY_LIMITS.maxTemporaryPasswordValidityHours + 1,
    ],
  ])('rejects %s above its safe upper bound', async (field, value) => {
    await expect(passwordPoliciesSchema.validate({ ...validPolicy, [field]: value })).rejects.toThrow()
  })

  it('rejects fractional numeric settings', async () => {
    await expect(
      passwordPoliciesSchema.validate({ ...validPolicy, password_history_count: 1.5 }),
    ).rejects.toThrow('Must be a whole number')
  })
})
