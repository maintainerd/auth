/**
 * Service Form Validation Schema
 *
 * Mirrors the backend rules in `internal/iam/validation_service.go` and the column
 * widths in migration 006. Every bound here previously disagreed with the server,
 * so the form accepted values that came back as a 422 (and rejected one the server
 * allows).
 */

import * as yup from 'yup'

/** Backend limits: validation_service.go + migration 006 column widths. */
export const SERVICE_LIMITS = {
  nameMin: 3,
  nameMax: 50,
  displayNameMin: 3,
  displayNameMax: 100,
  descriptionMax: 255,
  versionMax: 20,
} as const

export const SERVICE_STATUSES = ['active', 'maintenance', 'deprecated', 'inactive'] as const

// Service Form Schema
export const serviceSchema = yup.object({
  name: yup
    .string()
    .required('Service name is required')
    .min(SERVICE_LIMITS.nameMin, `Service name must be at least ${SERVICE_LIMITS.nameMin} characters`)
    .max(SERVICE_LIMITS.nameMax, `Service name must not exceed ${SERVICE_LIMITS.nameMax} characters`)
    .matches(
      /^[a-z0-9-]+$/,
      'Service name must contain only lowercase letters, numbers, and hyphens'
    ),
  displayName: yup
    .string()
    .required('Display name is required')
    .min(
      SERVICE_LIMITS.displayNameMin,
      `Display name must be at least ${SERVICE_LIMITS.displayNameMin} characters`,
    )
    .max(
      SERVICE_LIMITS.displayNameMax,
      `Display name must not exceed ${SERVICE_LIMITS.displayNameMax} characters`,
    ),
  // Optional, matching the backend (Length(0, 255)) and the NOT NULL DEFAULT ''
  // column. The form used to require 10+ characters, blocking a create the server
  // would have accepted.
  description: yup
    .string()
    .default('')
    .max(
      SERVICE_LIMITS.descriptionMax,
      `Description must not exceed ${SERVICE_LIMITS.descriptionMax} characters`,
    ),
  // The column is VARCHAR(20); the form allowed 50, so a longer value reached
  // Postgres as a truncation error and surfaced as a 500.
  version: yup
    .string()
    .required('Version is required')
    .max(SERVICE_LIMITS.versionMax, `Version must not exceed ${SERVICE_LIMITS.versionMax} characters`),
  status: yup
    .string()
    .oneOf(SERVICE_STATUSES, 'Invalid status')
    .required('Status is required')
})

export type ServiceFormData = yup.InferType<typeof serviceSchema>
