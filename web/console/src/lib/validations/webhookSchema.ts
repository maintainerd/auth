/**
 * Webhook Form Validation Schema
 * Yup validation schema for webhook-endpoint forms.
 */

import * as yup from 'yup'

export const webhookSchema = yup.object({
  url: yup
    .string()
    .required('URL is required')
    .url('Must be a valid URL (e.g. https://example.com/webhook)')
    .max(2048, 'URL must not exceed 2048 characters'),
  description: yup
    .string()
    .max(500, 'Description must not exceed 500 characters')
    .default(''),
  subscribeAll: yup
    .boolean()
    .required()
    .default(true),
  maxRetries: yup
    .number()
    .typeError('Max retries must be a number')
    .required('Max retries is required')
    .integer('Max retries must be a whole number')
    .min(0, 'Max retries must be at least 0')
    .max(10, 'Max retries must not exceed 10'),
  timeoutSeconds: yup
    .number()
    .typeError('Timeout must be a number')
    .required('Timeout is required')
    .integer('Timeout must be a whole number')
    .min(1, 'Timeout must be at least 1 second')
    .max(120, 'Timeout must not exceed 120 seconds'),
  status: yup
    .string()
    .oneOf(['active', 'inactive'], 'Invalid status')
    .required('Status is required'),
})

export type WebhookFormData = {
  url: string
  description: string
  subscribeAll: boolean
  maxRetries: number
  timeoutSeconds: number
  status: 'active' | 'inactive'
}
