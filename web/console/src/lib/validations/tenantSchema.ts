/**
 * Tenant Form Validation Schemas
 * Yup validation schemas for tenant-related forms
 *
 * Every rule below mirrors a rule the maintainerd-auth backend actually
 * enforces, so the form never rejects input the server would accept and never
 * lets through input the server will bounce. Sources:
 *   internal/tenant/validation_tenant.go:14 (slug pattern), :19-33 (reserved
 *   slugs), :64-81 (create DTO), :83-100 (update DTO)
 *   internal/shared/constants.go:8-11 (status vocabulary)
 *   internal/platform/database/migration/001_create_tenants_table.go:13-16
 */

import * as yup from 'yup'

/**
 * Names that would shadow a platform host and are rejected outright by the
 * server. Checked client-side so the user is told before they submit rather
 * than after a round trip. Mirrors reservedTenantSlugs in
 * maintainerd-auth internal/tenant/validation_tenant.go:19-33 — keep in sync.
 */
export const RESERVED_TENANT_SLUGS = [
  'system',
  'console',
  'api',
  'control-api',
  'control',
  'auth',
  'www',
  'admin',
  'root',
  'rabbitmq',
  'prometheus',
  'grafana',
  'signoz',
] as const

/**
 * The tenant name doubles as the DNS subdomain ({tenant}.auth.maintainerd.*),
 * so it must be a DNS-safe label. The previous `^[a-z0-9-]+$` accepted leading
 * and trailing hyphens that the server's anchored pattern rejects.
 */
const TENANT_SLUG_PATTERN = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/

// Create Tenant Form Schema
export const createTenantSchema = yup.object({
  name: yup
    .string()
    .required('Tenant name is required')
    // Length(3, 63) server-side — min(2) let a 2-character name through the
    // form only to be rejected by the API.
    .min(3, 'Tenant name must be at least 3 characters')
    .max(63, 'Tenant name must not exceed 63 characters')
    .matches(
      TENANT_SLUG_PATTERN,
      'Tenant name must start and end with a letter or number, and can only contain lowercase letters, numbers, and hyphens'
    )
    .notOneOf(
      RESERVED_TENANT_SLUGS as unknown as string[],
      'This tenant name is reserved by the platform. Choose a different name.'
    ),
  display_name: yup
    .string()
    // DELIBERATELY STRICTER THAN THE SERVER — reviewed, not an oversight.
    // TenantCreateRequestDTO/TenantUpdateRequestDTO declare no DisplayName rule
    // (internal/tenant/validation_tenant.go:64-100) and the column is nullable,
    // so the API would store "" without complaint. The console renders
    // display_name as the tenant's label everywhere with no fallback to `name`
    // — the tenants table, the details header, the "Edit {display_name}" page
    // title — so an empty one leaves those surfaces blank and the tenant
    // unidentifiable. Requiring it costs one field and cannot reject anything a
    // user would actually want to submit.
    .required('Display name is required')
    // VARCHAR(255) (001_create_tenants_table.go:14) is the only real bound; the
    // former 100-character cap was a client invention.
    .max(255, 'Display name must not exceed 255 characters'),
  description: yup
    .string()
    .required('Description is required')
    // Length(8, 200) server-side — the client demanded 10 and allowed 500.
    .min(8, 'Description must be at least 8 characters')
    .max(200, 'Description must not exceed 200 characters'),
  status: yup
    .string()
    .required('Status is required')
    .oneOf(['active', 'inactive', 'pending', 'suspended'], 'Invalid status'),
})

export type CreateTenantFormData = yup.InferType<typeof createTenantSchema>
