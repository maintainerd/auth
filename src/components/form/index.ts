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
export { FormTextareaField, type FormTextareaFieldProps } from './FormTextareaField'
export { FormPasswordField, type FormPasswordFieldProps } from './FormPasswordField'
export { FormSelectField, type FormSelectFieldProps, type SelectOption } from './FormSelectField'
export { FormCheckboxField, type FormCheckboxFieldProps } from './FormCheckboxField'
export { FormSwitchField, type FormSwitchFieldProps } from './FormSwitchField'
export { FormDateField, type FormDateFieldProps } from './FormDateField'
export { FormFileUploadField, type FormFileUploadFieldProps } from './FormFileUploadField'
export { default as FormSubmitButton } from './FormSubmitButton'
export { default as FormSetupCard } from './FormSetupCard'

