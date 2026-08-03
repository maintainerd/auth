/**
 * FormSwitchSubContainer
 *
 * A FormSwitchField wrapped in a themed sub-container box. The box follows the
 * card sub-container theme config (key/value surfaces, linked clients, etc.),
 * so switches that live in their own bordered surface pick up the theme instead
 * of a hardcoded card box.
 *
 * Props match FormSwitchField; containerClassName styles the sub-container box.
 */

import { FormSwitchField, type FormSwitchFieldProps } from "@/components/form"
import { cn } from "@/lib/utils"

export type FormSwitchSubContainerProps = FormSwitchFieldProps

export function FormSwitchSubContainer({
  containerClassName,
  ...props
}: FormSwitchSubContainerProps) {
  return (
    <div data-md-switch-sub-container className={cn("rounded-md border p-4", containerClassName)}>
      <FormSwitchField {...props} />
    </div>
  )
}

FormSwitchSubContainer.displayName = "FormSwitchSubContainer"
