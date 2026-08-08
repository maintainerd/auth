import { forwardRef, useState } from "react"
import type { PasswordConfigPublic } from "@/services/api/tenants/types"
import { FormPasswordField, type FormPasswordFieldProps } from "@/components/form"
import { PasswordRequirements } from "./PasswordRequirements"

export interface FormPasswordFieldWithPolicyProps extends FormPasswordFieldProps {
  passwordConfig?: PasswordConfigPublic
}

export const FormPasswordFieldWithPolicy = forwardRef<HTMLInputElement, FormPasswordFieldWithPolicyProps>(
  ({ passwordConfig, ...props }, ref) => {
    const [password, setPassword] = useState("")

    return (
      <div>
        <FormPasswordField
          ref={ref}
          {...props}
          onChange={(e) => {
            setPassword(e.target.value)
            props.onChange?.(e)
          }}
        />
        {password.length > 0 && (
          <PasswordRequirements password={password} config={passwordConfig} />
        )}
      </div>
    )
  }
)

FormPasswordFieldWithPolicy.displayName = "FormPasswordFieldWithPolicy"
