import * as yup from 'yup'

export const roleSchema = yup.object({
  name: yup
    .string()
    .required('Role name is required')
    .min(3, 'Role name must be at least 3 characters')
    .max(20, 'Role name must not exceed 20 characters')
    .matches(
      /^[a-z0-9:-]+$/,
      'Role name must contain only lowercase letters, numbers, hyphens, and colons'
    ),
  description: yup
    .string()
    .optional()
    .max(100, 'Description must not exceed 100 characters'),
  status: yup
    .string()
    .oneOf(['active', 'inactive'], 'Invalid status')
    .required('Status is required'),
})

export type RoleFormData = yup.InferType<typeof roleSchema>
