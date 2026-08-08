/**
 * Validation Schemas Index
 * Central export point for all validation schemas
 */

// Auth schemas
export {
  buildLoginSchema,
  buildRegisterSchema,
  buildResetPasswordSchema,
  forgotPasswordSchema,
  type LoginFormData,
  type RegisterFormData,
  type ForgotPasswordFormData,
  type ResetPasswordFormData
} from './authSchema'
