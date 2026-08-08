/**
 * Branding Form Validation Schema
 * Yup validation schema for branding template forms. Theme tokens (colors, font)
 * are managed separately as a fixed key/value set, so they are not validated here.
 */

import * as yup from 'yup'
import { AUTH_UI_TEMPLATE_IDS } from "@/lib/branding/authUiTemplates"

const optionalUrl = (label: string) =>
  yup
    .string()
    .trim()
    .max(2048, `${label} must not exceed 2048 characters`)
    .matches(/^https?:\/\//, {
      excludeEmptyString: true,
      message: `${label} must start with http:// or https://`,
    })
    .default('')

export const brandingSchema = yup.object({
  name: yup
    .string()
    .trim()
    .required('Name is required')
    .min(2, 'Name must be at least 2 characters')
    .max(100, 'Name must not exceed 100 characters'),
  layout: yup
    .string()
    .oneOf(['centered', 'full_page', 'split'], 'Select a valid login layout')
    .required('Login layout is required')
    .default('centered'),
  ui_template: yup
    .string()
    .oneOf([...AUTH_UI_TEMPLATE_IDS], 'Select a valid login template')
    .required('Login template is required')
    .default('centered-card'),
  company_name: yup
    .string()
    .trim()
    .max(255, 'Company name must not exceed 255 characters')
    .default(''),
  logo_label: yup
    .string()
    .trim()
    .max(255, 'Logo label must not exceed 255 characters')
    .default('Maintainerd-IAM'),
  logo_detail: yup
    .string()
    .trim()
    .max(255, 'Logo detail must not exceed 255 characters')
    .default(''),
  show_logo_label: yup
    .boolean()
    .default(true),
  identity_logo_label: yup
    .string()
    .trim()
    .max(255, 'Identity logo label must not exceed 255 characters')
    .default(''),
  identity_show_logo_label: yup
    .boolean()
    .default(true),
  logo_url: optionalUrl('Logo URL'),
  favicon_url: optionalUrl('Favicon URL'),
  support_url: optionalUrl('Support URL'),
  privacy_policy_url: optionalUrl('Privacy policy URL'),
  terms_of_service_url: optionalUrl('Terms of service URL'),
})

export type BrandingFormData = yup.InferType<typeof brandingSchema>
