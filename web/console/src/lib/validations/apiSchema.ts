/**
 * API Form Validation Schema
 * Yup validation schema for API-related forms
 */

import * as yup from 'yup'

// Lengths mirror the server rules in APICreateRequestDTO/APIUpdateRequestDTO
// (internal/iam/validation_api.go). They were 2-100 with a required 10-500
// description, so a valid 2-character name was accepted here and then rejected
// by the API, while a blank description was blocked here even though the server
// allows one.
export const API_LIMITS = {
  nameMin: 3,
  nameMax: 50,
  displayNameMin: 3,
  displayNameMax: 50,
  descriptionMax: 200,
} as const

// API Form Schema
export const apiSchema = yup.object({
  name: yup
    .string()
    .required('API name is required')
    .min(API_LIMITS.nameMin, `API name must be at least ${API_LIMITS.nameMin} characters`)
    .max(API_LIMITS.nameMax, `API name must not exceed ${API_LIMITS.nameMax} characters`)
    // Client-only rule with no server counterpart: the field is a FormSlugField
    // that sanitizes keystrokes and pastes down to this alphabet, so the pattern
    // can only ever fire on a value the input itself would not have produced.
    .matches(
      /^[a-z0-9-]+$/,
      'API name must contain only lowercase letters, numbers, and hyphens'
    ),
  displayName: yup
    .string()
    .required('Display name is required')
    .min(API_LIMITS.displayNameMin, `Display name must be at least ${API_LIMITS.displayNameMin} characters`)
    .max(API_LIMITS.displayNameMax, `Display name must not exceed ${API_LIMITS.displayNameMax} characters`),
  // Optional server-side (`validation.Length(0, 200)` with no Required rule), so
  // requiring it here blocked a submission the API would have accepted.
  description: yup
    .string()
    .defined()
    .max(API_LIMITS.descriptionMax, `API description must not exceed ${API_LIMITS.descriptionMax} characters`)
    .default(''),
  status: yup
    .string()
    .oneOf(['active', 'inactive'], 'Invalid status')
    .required('Status is required'),
  // Required only: the value comes from a select of real services, and yup's
  // .uuid() enforces the RFC 4122 version/variant nibbles, which would reject
  // legitimate UUIDs the server accepts (the backend uses a plain is.UUID check).
  serviceId: yup
    .string()
    .required('Service is required')
})

export type ApiFormData = yup.InferType<typeof apiSchema>
