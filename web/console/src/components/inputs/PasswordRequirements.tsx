import { Check, Info, X } from "lucide-react"
import type { PasswordConfigPublic } from "@/services/api/tenants/types"
import { cn } from "@/lib/utils"
import { buildPasswordRules } from "./passwordRules"

export interface PasswordRequirementsProps {
  password: string
  config?: PasswordConfigPublic
  className?: string
}

export function PasswordRequirements({ password, config, className }: PasswordRequirementsProps) {
  const rules = buildPasswordRules(password, config)

  return (
    <ul className={cn("mt-1 flex flex-col gap-1", className)}>
      {rules.map((rule) => (
        <li
          key={rule.label}
          className={cn(
            "flex items-center gap-2 text-xs transition-colors",
            rule.met ? "text-emerald-600" : "text-muted-foreground",
          )}
        >
          {rule.met ? (
            <Check className="size-3.5 shrink-0" />
          ) : (
            <X className="size-3.5 shrink-0 text-muted-foreground/60" />
          )}
          <span>{rule.label}</span>
        </li>
      ))}
      <li className="mt-1 flex items-start gap-2 text-xs text-muted-foreground">
        <Info className="mt-0.5 size-3.5 shrink-0" />
        <span>
          Additional tenant checks for password strength, common passwords, and known breaches may
          run when you submit.
        </span>
      </li>
    </ul>
  )
}
