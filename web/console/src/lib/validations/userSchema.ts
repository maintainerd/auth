import * as yup from 'yup'
import { isValidEmail, isValidPhone } from './regex'
import type { PasswordConfigPublic } from '@/services/api/tenants/types'

function buildPasswordValidation(cfg?: PasswordConfigPublic) {
  let schema = yup.string().required('Password is required')

  const minLen = cfg?.min_length ?? 8
  const maxLen = cfg?.max_length ?? 128
  schema = schema
    .min(minLen, `Password must be at least ${minLen} characters`)
  if (maxLen > 0) {
    schema = schema.max(maxLen, `Password must not exceed ${maxLen} characters`)
  }
  if (cfg?.require_uppercase ?? true) {
    schema = schema.matches(/[A-Z]/, 'Password must contain at least one uppercase letter')
  }
  if (cfg?.require_lowercase ?? true) {
    schema = schema.matches(/[a-z]/, 'Password must contain at least one lowercase letter')
  }
  if (cfg?.require_number ?? true) {
    schema = schema.matches(/[0-9]/, 'Password must contain at least one number')
  }
  if (cfg?.require_symbol ?? true) {
    schema = schema.matches(
      /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]/,
      'Password must contain at least one special character',
    )
  }
  return schema
}

export function buildUserSchema(cfg?: PasswordConfigPublic) {
  return yup.object({
    username: yup
      .string()
      .required('Username is required')
      .min(3, 'Username must be at least 3 characters')
      .max(50, 'Username must not exceed 50 characters')
      .matches(
        /^[a-zA-Z0-9._@-]+$/,
        'Username must contain only letters, numbers, dots, underscores, @ and hyphens'
      ),
    email: yup
      .string()
      .required('Email is required')
      .max(254, 'Email must not exceed 254 characters')
      .test('email-format', 'Invalid email address', (value) =>
        !!value && isValidEmail(value)
      ),
    phone: yup
      .string()
      .transform((value) => (value === '' ? undefined : value))
      .optional()
      .test('phone-format', 'Invalid phone number format', (value) =>
        !value || isValidPhone(value)
      ),
    password: yup
      .string()
      .when('$isCreating', {
        is: true,
        then: () => buildPasswordValidation(cfg),
        otherwise: (s) => s.optional()
      }),
    status: yup
      .string()
      .oneOf(['active', 'inactive', 'pending', 'suspended'], 'Invalid status')
      .required('Status is required'),
  })
}

export const userSchema = buildUserSchema()

export type UserFormData = {
  username: string
  email: string
  phone: string | undefined
  password: string | undefined
  status: 'active' | 'inactive' | 'pending' | 'suspended'
}
