/**
 * Policy Validation Schema
 * Validation rules for policy forms using Yup
 */

import * as yup from 'yup'

/**
 * Policy statement schema
 */
export const policyStatementSchema = yup.object({
  effect: yup
    .string()
    .oneOf(['allow', 'deny'], 'Effect must be either allow or deny')
    .required('Effect is required'),
  action: yup
    .array()
    .of(yup.string().required('Action cannot be empty'))
    .min(1, 'At least one action is required')
    .required('Actions are required'),
  resource: yup
    .array()
    .of(yup.string().required('Resource cannot be empty'))
    .min(1, 'At least one resource is required')
    .required('Resources are required'),
})

/**
 * Policy document schema
 * Note: version field is not included here as it's synced from the root version field
 */
export const policyDocumentSchema = yup.object({
  statement: yup
    .array()
    .of(policyStatementSchema)
    .min(1, 'At least one statement is required')
    .required('Statements are required'),
})

/**
 * Policy name character set, mirroring the server's `policyNamePattern`
 * (internal/iam/validation_policy.go). Forward and back slashes are part of the
 * server's set — omitting them rejected valid ARN-style names client-side.
 */
const POLICY_NAME_PATTERN = /^[a-z0-9_:/\\-]+$/

/**
 * Policy form validation schema
 *
 * Every rule below mirrors PolicyCreateRequestDTO/PolicyUpdateRequestDTO
 * validation in internal/iam/validation_policy.go. Rules that are stricter than
 * the server reject payloads the API would have accepted; rules that are looser
 * surface as a 422 after submit instead of inline.
 */
export const policySchema = yup.object({
  name: yup
    .string()
    .required('Policy name is required')
    .min(3, 'Policy name must be at least 3 characters')
    .max(150, 'Policy name must not exceed 150 characters')
    .matches(
      POLICY_NAME_PATTERN,
      'Policy name can only contain lowercase letters, numbers, underscores, colons, forward slashes, backslashes, and hyphens'
    ),
  // Optional server-side (`*string`, Length(0, 500)) and stored as TEXT NOT NULL
  // DEFAULT '' — so an empty description is valid, not a validation failure.
  description: yup
    .string()
    .defined()
    .max(500, 'Description must not exceed 500 characters'),
  // The server accepts any 1-20 char string, not semver. The backend's own
  // seeder writes "v1" (internal/setup/seeder/013_control_policy.go), which a
  // semver rule rejects — making seeded policies uneditable in the console.
  version: yup
    .string()
    .required('Version is required')
    .max(20, 'Version must not exceed 20 characters')
    // The policies table has CHECK (btrim(version) <> ''), so a whitespace-only
    // version passes `required` here but fails at insert time.
    .matches(/\S/, 'Version is required'),
  status: yup
    .string()
    .oneOf(['active', 'inactive'], 'Status must be either active or inactive')
    .required('Status is required'),
  document: policyDocumentSchema.required('Policy document is required'),
})

/**
 * TypeScript type inferred from the schema
 */
export type PolicyFormData = yup.InferType<typeof policySchema>
export type PolicyStatementFormData = yup.InferType<typeof policyStatementSchema>
export type PolicyDocumentFormData = yup.InferType<typeof policyDocumentSchema>

