import { forwardRef } from "react"
import { Field, FieldError } from "@/components/ui/field"
import { cn } from "@/lib/utils"

export interface FormConsentCheckboxProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "type"> {
  error?: string
  termsUrl?: string
  privacyUrl?: string
}

/**
 * Required Terms-of-Service / Privacy acceptance for account-creation forms.
 * Works with react-hook-form via a forwarded ref (`{...register("acceptTerms")}`).
 * When a branding URL is configured the policy name links out (new tab);
 * otherwise it renders as plain text so the sentence still reads correctly.
 */
export const FormConsentCheckbox = forwardRef<HTMLInputElement, FormConsentCheckboxProps>(
  ({ error, termsUrl, privacyUrl, id = "accept-terms", ...props }, ref) => {
    const linkClass = "font-medium text-primary underline-offset-4 hover:underline"
    return (
      <Field>
        <label htmlFor={id} className="flex items-start gap-2.5 text-sm text-muted-foreground">
          <input
            ref={ref}
            id={id}
            type="checkbox"
            aria-invalid={error ? "true" : "false"}
            aria-describedby={error ? `${id}-error` : undefined}
            className={cn(
              "mt-0.5 size-4 shrink-0 rounded-[3px] border accent-primary",
              error ? "border-destructive" : "border-input"
            )}
            {...props}
          />
          <span>
            I agree to the{" "}
            {termsUrl ? (
              <a href={termsUrl} target="_blank" rel="noopener noreferrer" className={linkClass}>
                Terms of Service
              </a>
            ) : (
              <span className="font-medium text-foreground">Terms of Service</span>
            )}{" "}
            and{" "}
            {privacyUrl ? (
              <a href={privacyUrl} target="_blank" rel="noopener noreferrer" className={linkClass}>
                Privacy Policy
              </a>
            ) : (
              <span className="font-medium text-foreground">Privacy Policy</span>
            )}
            .
          </span>
        </label>
        {error && <FieldError id={`${id}-error`}>{error}</FieldError>}
      </Field>
    )
  },
)

FormConsentCheckbox.displayName = "FormConsentCheckbox"
