/**
 * User Profile Form Validation Schema
 *
 * Mirrors the backend `ProfileRequestDTO` validation (internal/user/validation_profile.go):
 * only `first_name` is required; every other field is optional but, when present,
 * must satisfy the same length / format / enum rules. Empty inputs are normalised
 * to `undefined` so we never send empty strings (the backend uses NilOrNotEmpty).
 */

import * as yup from 'yup'

export const GENDER_VALUES = ['male', 'female', 'other', 'prefer_not_to_say'] as const

// An optional, trimmed string that becomes `undefined` when blank.
const optionalString = () =>
  yup
    .string()
    .transform((value: unknown) => {
      if (typeof value !== 'string') return undefined
      const trimmed = value.trim()
      return trimmed === '' ? undefined : trimmed
    })
    .optional()

export const userProfileSchema = yup.object({
  first_name: yup
    .string()
    .transform((value: unknown) => (typeof value === 'string' ? value.trim() : value))
    .required('First name is required')
    .max(100, 'First name must be at most 100 characters'),
  middle_name: optionalString().max(100, 'Middle name must be at most 100 characters'),
  last_name: optionalString().max(100, 'Last name must be at most 100 characters'),
  display_name: optionalString().max(100, 'Display name must be at most 100 characters'),
  birthdate: optionalString().matches(/^\d{4}-\d{2}-\d{2}$/, 'Birthdate must be in YYYY-MM-DD format'),
  gender: optionalString().oneOf(GENDER_VALUES as unknown as string[], 'Invalid gender'),
  profile_url: optionalString()
    .test('is-http-url', 'Enter a valid URL starting with http:// or https://', (value) => {
      if (!value) return true
      try {
        const url = new URL(value)
        return url.protocol === 'http:' || url.protocol === 'https:'
      } catch {
        return false
      }
    })
    .max(1000, 'Profile URL must be at most 1000 characters'),
})

export type UserProfileFormData = yup.InferType<typeof userProfileSchema>
