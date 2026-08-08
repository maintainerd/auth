/**
 * Authentication Form Validation Schemas
 * Yup validation schemas for authentication-related forms
 */

import * as yup from 'yup'
import type { PasswordConfigPublic } from '@/services/api/tenants/types'

const CONSENT_REQUIRED_MESSAGE = 'You must accept the Terms of Service and Privacy Policy'

// Required Terms-of-Service / Privacy acceptance. Used on account-creation forms
// so the consent the backend records at registration reflects an explicit,
// user-driven agreement (compliance) rather than a silent default.
export function acceptTermsValidation() {
  return yup
    .boolean()
    .oneOf([true], CONSENT_REQUIRED_MESSAGE)
    .required(CONSENT_REQUIRED_MESSAGE)
}

export function buildPasswordValidation(cfg?: PasswordConfigPublic) {
  let schema = yup.string().required('Password is required')

  const minLen = cfg?.min_length ?? 12
  const maxLen = cfg?.max_length ?? 128
  schema = schema.test(
    'minimum-character-length',
    `Password must be at least ${minLen} characters`,
    (value) => !value || Array.from(value).length >= minLen,
  )
  if (maxLen > 0) {
    schema = schema.test(
      'maximum-character-length',
      `Password must not exceed ${maxLen} characters`,
      (value) => !value || Array.from(value).length <= maxLen,
    )
  }
  if (cfg?.require_uppercase) {
    schema = schema.matches(/[A-Z]/, 'Password must contain at least one uppercase letter')
  }
  if (cfg?.require_lowercase) {
    schema = schema.matches(/[a-z]/, 'Password must contain at least one lowercase letter')
  }
  if (cfg?.require_number) {
    schema = schema.matches(/[0-9]/, 'Password must contain at least one digit')
  }
  if (cfg?.require_symbol) {
    schema = schema.matches(/[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]/, 'Password must contain at least one special character')
  }
  return schema
}

export function buildLoginSchema() {
  return yup.object({
    // Sign-in accepts a USERNAME or an email: the backend looks the identifier
    // up by username first and falls back to email (authn.service_login). This
    // enforced .email(), so any account whose username was not email-shaped —
    // every admin-created user, since usernames are only length-checked — could
    // never sign in through this app. The client rejected the input before it
    // was ever sent, and the field label said "Email", so there was no clue.
    email: yup
      .string()
      .required('Email or username is required')
      .max(255, 'Must not exceed 255 characters'),
    // Existing passwords remain valid after an administrator strengthens the
    // tenant policy. Complexity rules apply only when a password is created or
    // changed; login should validate presence and let the server authenticate.
    password: yup.string().required('Password is required'),
  })
}

export interface LoginFormData {
  email: string
  password: string
}

// Register Form Schema (config-driven)
// Mirrors buildLoginSchema: the password rules come from the tenant's
// password_config so registration enforces the same policy the backend does.
// Which extra fields the resolved registration context demands. Client-side
// conditionality is UX only — the server enforces required_fields regardless.
export interface RegisterFieldRequirements {
  fullname?: boolean
  phone?: boolean
}

// Mirrors internal/platform/valid/valid.go IsValidPhoneNumber: the regex plus the
// 7-15 digit count. A leading zero is rejected there, so the message says how to
// satisfy it rather than just declaring the value invalid.
const PHONE_SHAPE = /^[+]?[1-9][\d\s\-().]{6,20}$/
const PHONE_MESSAGE = 'Enter a phone number with country code, e.g. +1 212 555 1234'

function isBackendValidPhone(value: string): boolean {
  const digits = value.replace(/\D/g, '')
  if (digits.length < 7 || digits.length > 15) return false
  return PHONE_SHAPE.test(value)
}

export function buildRegisterSchema(cfg?: PasswordConfigPublic, required?: RegisterFieldRequirements) {
  return yup.object({
    fullname: required?.fullname
      ? yup.string().trim().required('Full name is required').max(255, 'Full name must not exceed 255 characters')
      : yup.string().trim().max(255, 'Full name must not exceed 255 characters').optional(),
    phone: required?.phone
      ? yup
          .string()
          .trim()
          .required('Phone number is required')
          .test('phone-format', PHONE_MESSAGE, (v) => !!v && isBackendValidPhone(v))
      : yup
          .string()
          .trim()
          .optional()
          .test('phone-format', PHONE_MESSAGE, (v) => !v || isBackendValidPhone(v)),
    email: yup
      .string()
      .required('Email is required')
      .email('Please enter a valid email address')
      .max(255, 'Email must not exceed 255 characters'),
    password: buildPasswordValidation(cfg),
    confirmPassword: yup
      .string()
      .required('Please confirm your password')
      .oneOf([yup.ref('password')], 'Passwords must match'),
    acceptTerms: acceptTermsValidation(),
  })
}

// Register Form Schema (static fallback — kept for back-compat / typing)
export const registerSchema = yup.object({
  email: yup
    .string()
    .required('Email is required')
    .email('Please enter a valid email address')
    .max(255, 'Email must not exceed 255 characters'),
  password: yup
    .string()
    .required('Password is required')
    .min(8, 'Password must be at least 8 characters')
    .matches(
      /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]/,
      'Password must contain uppercase, lowercase, number, and special character'
    ),
  confirmPassword: yup
    .string()
    .required('Please confirm your password')
    .oneOf([yup.ref('password')], 'Passwords must match'),
  acceptTerms: acceptTermsValidation(),
})

// Declared explicitly rather than inferred from `registerSchema`: the static
// schema is only the back-compat fallback, so inference would silently omit the
// conditional fields that buildRegisterSchema adds.
export type RegisterFormData = yup.InferType<typeof registerSchema> & {
  fullname?: string
  phone?: string
}

// Forgot Password Form Schema
export const forgotPasswordSchema = yup.object({
  email: yup
    .string()
    .required('Email is required')
    .email('Please enter a valid email address')
    .max(255, 'Email must not exceed 255 characters')
})

export type ForgotPasswordFormData = yup.InferType<typeof forgotPasswordSchema>

export function buildResetPasswordSchema(cfg?: PasswordConfigPublic) {
  return yup.object({
    password: buildPasswordValidation(cfg),
    confirmPassword: yup
      .string()
      .required('Please confirm your password')
      .oneOf([yup.ref('password')], 'Passwords must match'),
  })
}

export const resetPasswordSchema = buildResetPasswordSchema()

export type ResetPasswordFormData = yup.InferType<typeof resetPasswordSchema>
