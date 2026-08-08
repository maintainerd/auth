/**
 * IP Rule Dialog Component
 */

import { useEffect } from 'react'
import { useForm, Controller } from 'react-hook-form'
import { yupResolver } from '@hookform/resolvers/yup'
import * as yup from 'yup'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { FormInputField } from '@/components/form/FormInputField'
import { FormSelectField } from '@/components/form/FormSelectField'
import type { IpRestrictionRule } from '@/services/api/ip-restriction-rules/types'

const ipRuleSchema = yup.object({
  description: yup.string().required('Description is required'),
  type: yup.string().oneOf(['allow', 'deny'] as const).required('Type is required'),
  ipAddress: yup
    .string()
    .required('IP address is required')
    .matches(
      /^(\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?$/,
      'Invalid IP address format (e.g., 192.168.1.1 or 192.168.1.0/24)'
    ),
  status: yup.string().oneOf(['active', 'inactive'] as const).required('Status is required'),
})

type IpRuleFormData = yup.InferType<typeof ipRuleSchema>

interface IpRuleDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (data: IpRuleFormData) => Promise<void>
  rule?: IpRestrictionRule
  isLoading?: boolean
}

export function IpRuleDialog({ open, onOpenChange, onSubmit, rule, isLoading }: IpRuleDialogProps) {
  const { handleSubmit, reset, register, control, formState: { errors } } = useForm<IpRuleFormData>({
    resolver: yupResolver(ipRuleSchema),
    defaultValues: {
      description: '',
      type: 'allow',
      ipAddress: '',
      status: 'active',
    },
  })

  useEffect(() => {
    if (rule) {
      reset({
        description: rule.description,
        type: rule.type,
        ipAddress: rule.ipAddress,
        status: rule.status,
      })
    } else {
      reset({
        description: '',
        type: 'allow',
        ipAddress: '',
        status: 'active',
      })
    }
  }, [rule, reset, open])

  const handleFormSubmit = async (data: IpRuleFormData) => {
    await onSubmit(data)
  }

  const handleFormSubmitWithStopPropagation = async (e: React.FormEvent<HTMLFormElement>) => {
    e.stopPropagation() // Prevent event from bubbling to parent form
    await handleSubmit(handleFormSubmit)(e)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>{rule ? 'Edit IP Rule' : 'Add IP Rule'}</DialogTitle>
          <DialogDescription>
            {rule ? 'Update the IP restriction rule details.' : 'Add a new IP address or range to allow or deny.'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleFormSubmitWithStopPropagation} className="space-y-4">
          <FormInputField
            label="IP Address/Range"
            placeholder="e.g., 192.168.1.1 or 192.168.1.0/24"
            error={errors.ipAddress?.message}
            required
            {...register('ipAddress')}
          />

          <Controller
            name="type"
            control={control}
            render={({ field }) => (
              <FormSelectField
                label="Type"
                placeholder="Select type"
                error={errors.type?.message}
                required
                options={[
                  { value: 'allow', label: 'Allow (Whitelist)' },
                  { value: 'deny', label: 'Deny (Blacklist)' },
                ]}
                value={field.value}
                onValueChange={field.onChange}
              />
            )}
          />

          <FormInputField
            label="Description"
            placeholder="e.g., Office network"
            error={errors.description?.message}
            required
            {...register('description')}
          />

          <Controller
            name="status"
            control={control}
            render={({ field }) => (
              <FormSelectField
                label="Status"
                placeholder="Select status"
                error={errors.status?.message}
                required
                options={[
                  { value: 'active', label: 'Active' },
                  { value: 'inactive', label: 'Inactive' },
                ]}
                value={field.value}
                onValueChange={field.onChange}
              />
            )}
          />

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? 'Saving...' : rule ? 'Update' : 'Add Rule'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
