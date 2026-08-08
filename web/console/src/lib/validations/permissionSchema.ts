/**
 * Permission Form Validation Schema
 * Yup validation schema for API permission forms
 */

import * as yup from 'yup'

// Mirrors validation.Length(3, 50) / Length(0, 200) on
// PermissionCreateRequestDTO and PermissionUpdateRequestDTO
// (internal/iam/validation_permission.go). Without the maxes the form happily
// submitted a 200-character name and surfaced the failure as a server error.
export const PERMISSION_LIMITS = {
  nameMin: 3,
  nameMax: 50,
  descriptionMax: 200,
} as const

/**
 * A permission name: two or more colon-separated lowercase segments, each an
 * alphanumeric run with single `-`/`_` separators (`reports:read`,
 * `users:read:own`, `billing-api:invoice_line:write`).
 *
 * The previous pattern was `^[a-z0-9-]+:[a-z0-9-]+$`, which mandated EXACTLY one
 * colon and no underscores, so it rejected `users:read:own` — a name the server
 * accepts — before the request was ever sent.
 *
 * Deliberately open-ended on segment count where the server currently caps at
 * four (`{1,3}` colon groups). A client that is looser than the server turns a
 * future cap change into a server-side message the form already knows how to
 * display, whereas a client that is tighter silently blocks valid names — which
 * is the bug being fixed here. Reserved first-segment namespaces are likewise
 * left to the server: that list is security policy that moves independently, and
 * a stale copy here would block names the API would have minted.
 */
export const PERMISSION_NAME_REGEX = /^[a-z0-9]+([_-][a-z0-9]+)*(:[a-z0-9]+([_-][a-z0-9]+)*)+$/

export const permissionSchema = yup.object({
  name: yup
    .string()
    .required("Permission name is required")
    .min(PERMISSION_LIMITS.nameMin, `Permission name must be at least ${PERMISSION_LIMITS.nameMin} characters`)
    .max(PERMISSION_LIMITS.nameMax, `Permission name must not exceed ${PERMISSION_LIMITS.nameMax} characters`)
    .matches(
      PERMISSION_NAME_REGEX,
      "Permission name must be lowercase colon-separated segments, e.g. users:read or users:read:own"
    ),
  description: yup
    .string()
    .required("Description is required")
    .max(PERMISSION_LIMITS.descriptionMax, `Description must not exceed ${PERMISSION_LIMITS.descriptionMax} characters`),
  status: yup.string().oneOf(["active", "inactive"]).required("Status is required"),
})

export type PermissionFormData = yup.InferType<typeof permissionSchema>

/**
 * The same permission, created from somewhere that has no owning API in context
 * (the permissions listing). `api_id` is Required on the server's
 * PermissionCreateRequestDTO, so a standalone create form has to collect it or
 * it cannot submit at all.
 */
export const permissionWithApiSchema = permissionSchema.shape({
  // Required only, for the same reason as the other pickers in this app: the
  // value comes from a select of real APIs, and yup's .uuid() rejects UUIDs the
  // backend's plain is.UUID check accepts.
  apiId: yup.string().required("API is required"),
})

export type PermissionWithApiFormData = yup.InferType<typeof permissionWithApiSchema>
