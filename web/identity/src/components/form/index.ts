/**
 * Form Components Export Index
 * Central export point for all reusable form components
 */

// The shared field scaffold. Build new field components on this rather than
// re-implementing label/description/error/aria markup.
export { FieldShell } from './FieldShell'
export {
  fieldControlProps,
  resolveFieldId,
  FIELD_INVALID_CONTROL_CLASS,
  type FieldShellOwnProps,
} from './fieldControl'
export { FormInputField, type FormInputFieldProps } from './FormInputField'
export { FormPasswordField, type FormPasswordFieldProps } from './FormPasswordField'
export { FormConsentCheckbox, type FormConsentCheckboxProps } from './FormConsentCheckbox'
export { PasswordRequirements, type PasswordRequirementsProps } from './PasswordRequirements'
export { buildPasswordRules, PASSWORD_SYMBOL_REGEX, type PasswordRule } from './passwordRules'
export { FormSelectField, type FormSelectFieldProps, type SelectOption } from './FormSelectField'
export { default as FormSubmitButton } from './FormSubmitButton'
export { default as FormSetupCard } from './FormSetupCard'
