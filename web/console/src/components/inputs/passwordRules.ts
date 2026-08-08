import type { PasswordConfigPublic } from "@/services/api/tenants/types"
import { PASSWORD_SPECIAL_REGEX } from "@/lib/validations/regex"

export interface PasswordRule {
  label: string
  met: boolean
}

const UPPERCASE_REGEX = /[A-Z]/
const LOWERCASE_REGEX = /[a-z]/
const DIGIT_REGEX = /[0-9]/

export function buildPasswordRules(password: string, config?: PasswordConfigPublic): PasswordRule[] {
  const minLen = config?.min_length ?? 8
  const maxLen = config?.max_length ?? 128
  const passwordLength = Array.from(password).length

  const rules: PasswordRule[] = [
    { label: `At least ${minLen} characters`, met: passwordLength >= minLen },
  ]
  if (maxLen > 0) {
    rules.push({ label: `No more than ${maxLen} characters`, met: passwordLength <= maxLen })
  }
  if (config?.require_uppercase ?? true) {
    rules.push({ label: 'One uppercase letter', met: UPPERCASE_REGEX.test(password) })
  }
  if (config?.require_lowercase ?? true) {
    rules.push({ label: 'One lowercase letter', met: LOWERCASE_REGEX.test(password) })
  }
  if (config?.require_number ?? true) {
    rules.push({ label: 'One number', met: DIGIT_REGEX.test(password) })
  }
  if (config?.require_symbol ?? true) {
    rules.push({ label: 'One special character', met: PASSWORD_SPECIAL_REGEX.test(password) })
  }
  return rules
}
