/**
 * Password rule evaluation, derived from the tenant's password_config.
 *
 * Kept in its own (non-component) module so it can be shared and unit-tested.
 * The rules here mirror buildPasswordValidation() in authSchema.ts exactly —
 * same defaults and symbol set — so the live checklist matches what the schema
 * actually enforces.
 */
import type { PasswordConfigPublic } from '@/services/api/tenants/types'

// Keep this identical to the symbol matcher in buildPasswordValidation.
export const PASSWORD_SYMBOL_REGEX = /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]/

export interface PasswordRule {
  label: string
  met: boolean
}

export function buildPasswordRules(password: string, config?: PasswordConfigPublic): PasswordRule[] {
  const minLen = config?.min_length ?? 12
  const maxLen = config?.max_length ?? 128
  const passwordLength = Array.from(password).length

  const rules: PasswordRule[] = [
    { label: `At least ${minLen} characters`, met: passwordLength >= minLen },
  ]
  if (maxLen > 0) {
    rules.push({ label: `No more than ${maxLen} characters`, met: passwordLength <= maxLen })
  }
  if (config?.require_uppercase) {
    rules.push({ label: 'One uppercase letter', met: /[A-Z]/.test(password) })
  }
  if (config?.require_lowercase) {
    rules.push({ label: 'One lowercase letter', met: /[a-z]/.test(password) })
  }
  if (config?.require_number) {
    rules.push({ label: 'One number', met: /[0-9]/.test(password) })
  }
  if (config?.require_symbol) {
    rules.push({ label: 'One special character', met: PASSWORD_SYMBOL_REGEX.test(password) })
  }
  return rules
}
