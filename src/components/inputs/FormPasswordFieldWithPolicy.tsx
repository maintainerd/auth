import { forwardRef, useState } from 'react'
import type { PasswordConfigPublic } from '@/services/api/tenants/types'
import { FormPasswordField, PasswordRequirements, type FormPasswordFieldProps } from '@/components/form'

export interface FormPasswordFieldWithPolicyProps extends FormPasswordFieldProps {
  passwordConfig?: PasswordConfigPublic
}

export const FormPasswordFieldWithPolicy = forwardRef<HTMLInputElement, FormPasswordFieldWithPolicyProps>(
  ({ passwordConfig, onChange, ...props }, ref) => {
    const [password, setPassword] = useState('')

    return (
      <div>
        <FormPasswordField
          ref={ref}
          {...props}
          onChange={(event) => {
            setPassword(event.target.value)
            onChange?.(event)
          }}
        />
        {password.length > 0 && (
          <PasswordRequirements password={password} config={passwordConfig} />
        )}
      </div>
    )
  },
)

FormPasswordFieldWithPolicy.displayName = 'FormPasswordFieldWithPolicy'
