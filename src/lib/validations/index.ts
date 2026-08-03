/**
 * Validation Schemas Index
 * Central export point for all validation schemas
 */

// Setup schemas
export {
  setupTenantSchema,
  setupAdminSchema,
  setupProfileContactSchema,
  setupProfileLocationSchema,
  type SetupTenantFormData,
  type SetupAdminFormData,
  type SetupProfileContactFormData,
  type SetupProfileLocationFormData,
} from './setupSchema'

// Auth schemas
// (removed) login/register/forgot/reset schemas — credential auth now lives in
// the hosted identity app; the console authenticates via OAuth.

// Service schemas
export {
  serviceSchema,
  type ServiceFormData
} from './serviceSchema'

// API schemas
export {
  apiSchema,
  type ApiFormData
} from './apiSchema'

// Permission schemas
export {
  permissionSchema,
  type PermissionFormData
} from './permissionSchema'

// Policy schemas
export {
  policySchema,
  type PolicyFormData,
} from './policySchema'

// Identity Provider schemas
export {
  identityProviderSchema,
  type IdentityProviderFormData
} from './identityProviderSchema'

// Client schemas
export {
  clientSchema,
  validateClientOAuthConfig,
  validateRedirectUri,
  isPublicClientType,
  authMethodRequiresSecret,
  CLIENT_TYPES,
  PUBLIC_CLIENT_TYPES,
  CLIENT_AUTH_METHODS,
  SECRET_BASED_AUTH_METHODS,
  GRANT_TYPES,
  RESPONSE_TYPES,
  REQUIRED_ACR_VALUES,
  type ClientFormData,
  type ClientTypeValue,
  type ClientAuthMethod,
  type ClientOAuthConfigValues
} from './clientSchema'

// Workload identity federation schemas
export {
  workloadIdentitySchema,
  validateIssuerUrl,
  validateSubjectPattern,
  validateAttributeMapping,
  validateAllowedScopes,
  parseAllowedScopes,
  isReservedClaimName,
  RESERVED_CLAIM_NAMES,
  MAX_ATTRIBUTE_MAPPING_ENTRIES,
  WORKLOAD_LIMITS,
  type WorkloadIdentityFormData
} from './workloadIdentitySchema'

// Role schemas
export {
  roleSchema,
  type RoleFormData
} from './roleSchema'

// Branding schemas
export {
  brandingSchema,
  type BrandingFormData
} from './brandingSchema'

// User schemas
export {
  userSchema,
  buildUserSchema,
} from './userSchema'

// User Profile schemas
export {
  userProfileSchema,
  type UserProfileFormData
} from './userProfileSchema'

// Tenant schemas
export {
  createTenantSchema,
  type CreateTenantFormData
} from './tenantSchema'

// Registration Flow schemas
export {
  registrationFlowSchema,
  type RegistrationFlowFormData
} from './registrationFlowSchema'

// Password Policies schemas
export {
  PASSWORD_POLICY_DEFAULTS,
  PASSWORD_POLICY_LIMITS,
  passwordPoliciesSchema,
  type PasswordPoliciesFormData
} from './passwordPoliciesSchema'

// Session Settings schemas
export {
  sessionSettingsSchema,
  type SessionSettingsFormData
} from './sessionSettingsSchema'

// Token Config schemas
export {
  tokenConfigSchema,
  type TokenConfigFormData
} from './tokenConfigSchema'

// Lockout Config schemas
export {
  lockoutConfigSchema,
  type LockoutConfigFormData
} from './lockoutConfigSchema'

// Registration Config schemas
export {
  registrationConfigSchema,
  type RegistrationConfigFormData
} from './registrationConfigSchema'

// Threat Detection Settings schemas
export {
  threatDetectionSettingsSchema,
  type ThreatDetectionSettingsFormData
} from './threatDetectionSettingsSchema'

// Webhook schemas
export {
  webhookSchema,
  type WebhookFormData
} from './webhookSchema'

// Tenant settings schemas
export { rateLimitConfigSchema, type RateLimitConfigFormData } from './rateLimitConfigSchema'
export { auditConfigSchema, type AuditConfigFormData } from './auditConfigSchema'
export { maintenanceConfigSchema, type MaintenanceConfigFormData } from './maintenanceConfigSchema'
