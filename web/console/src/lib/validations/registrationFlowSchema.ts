/**
 * Registration Flow Form Validation Schema
 *
 * IMPORTANT: these rules are kept in lock-step with the backend validation in
 * internal/idp/validation_registration_flow.go, so the two never disagree about
 * what a valid registration flow is:
 *
 *   - name             required, up to 100 chars, SLUG-shaped — it is the public
 *                      registration-link selector (?registration_flow=<name>)
 *   - description      OPTIONAL, max 500 (the backend has no minimum)
 *   - status           'active' | 'inactive' only — there is no 'draft'
 *   - clientId         required
 *   - requiredFields   subset of fullname/email/phone
 *
 * There is deliberately NO `identifier` field: the flow name IS the selector.
 * `verificationRequired`, `requiredFields` and `roleIds` live here (rather than in
 * component state) so one form reset hydrates them on edit and the
 * unsaved-changes guard can see them change.
 */

import * as yup from 'yup'

/**
 * Mirrors registrationFlowNamePattern in internal/idp/validation_registration_flow.go.
 * The name appears in a URL, so no spaces, no uppercase, no colons.
 */
export const REGISTRATION_FLOW_NAME_PATTERN = /^[a-z0-9][a-z0-9]*([-_][a-z0-9]+)*$/

/** Mirrors the backend allow-list in validateRequiredFieldsJSON. */
export const REGISTRATION_FLOW_REQUIRED_FIELDS = ['fullname', 'email', 'phone'] as const

// Registration Flow Form Schema
export const registrationFlowSchema = yup.object({
  name: yup
    .string()
    .required('Name is required')
    .min(2, 'Name must be at least 2 characters')
    // Length cap mirrors backend validation.Length(1, 100).
    .max(100, 'Name must not exceed 100 characters')
    .matches(REGISTRATION_FLOW_NAME_PATTERN, {
      message:
        'Name must be lowercase letters, numbers, hyphens and underscores (e.g. partner-signup)',
      excludeEmptyString: true,
    }),
  description: yup
    .string()
    // Optional, matching the backend's validation.Length(0, 500) — it never
    // required a description, let alone a 10-character minimum.
    .max(500, 'Description must not exceed 500 characters')
    .default(''),
  status: yup
    .string()
    .oneOf(['active', 'inactive'], 'Invalid status')
    .required('Status is required'),
  clientId: yup
    .string()
    .required('Client is required'),
  verificationRequired: yup
    .boolean()
    .defined()
    .default(false),
  requiredFields: yup
    .array()
    .of(yup.string().oneOf(REGISTRATION_FLOW_REQUIRED_FIELDS).required())
    .defined()
    .default([]),
  roleIds: yup
    .array()
    .of(yup.string().required())
    .defined()
    .default([]),
})

export type RegistrationFlowFormData = yup.InferType<typeof registrationFlowSchema>
