/**
 * FormScopeField
 *
 * A reusable comma-separated token input for OAuth2 / OIDC scopes. The text
 * input holds the canonical comma-separated value; a "Browse" popover offers a
 * searchable, checkbox list of common scopes — toggling one appends/removes it
 * from the end of the value. Tokens are validated (see SCOPE_TOKEN_REGEX) so
 * rubbish characters can't be entered.
 *
 * Controlled: pass `value` (comma-separated string) and `onValueChange`.
 */

import { useMemo, useState } from "react"
import { ChevronsUpDown, Plus } from "lucide-react"
import { Field, FieldLabel, FieldDescription, FieldError } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import { cn } from "@/lib/utils"
import { SCOPE_TOKEN_REGEX } from "@/lib/validations/regex"

export interface FormScopeFieldProps {
  label: string
  value: string
  onValueChange: (value: string) => void
  suggestions?: string[]
  error?: string
  description?: string
  placeholder?: string
  required?: boolean
  disabled?: boolean
  id?: string
  containerClassName?: string
}

function parseTokens(value: string): string[] {
  return value.split(/[\n,]/).map((t) => t.trim()).filter(Boolean)
}

export function FormScopeField({
  label,
  value,
  onValueChange,
  suggestions = [],
  error,
  description,
  placeholder,
  required = false,
  disabled = false,
  id,
  containerClassName,
}: FormScopeFieldProps) {
  const fieldId = id || label.toLowerCase().replace(/\s+/g, "-")
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState("")

  const tokens = useMemo(() => parseTokens(value), [value])
  const tokenSet = useMemo(() => new Set(tokens), [tokens])

  // Surface a live error for malformed tokens even before submit; the parent's
  // `error` (from validation on submit) takes precedence when present.
  const invalidTokens = useMemo(() => tokens.filter((t) => !SCOPE_TOKEN_REGEX.test(t)), [tokens])
  const localError =
    invalidTokens.length > 0
      ? `Invalid scope${invalidTokens.length > 1 ? "s" : ""}: ${invalidTokens.join(", ")}`
      : undefined
  const shownError = error ?? localError

  const commit = (next: string[]) => {
    // De-dupe preserving order.
    const seen = new Set<string>()
    const deduped = next.filter((t) => (seen.has(t) ? false : (seen.add(t), true)))
    onValueChange(deduped.join(", "))
  }

  const toggle = (scope: string) => {
    if (tokenSet.has(scope)) commit(tokens.filter((t) => t !== scope))
    else commit([...tokens, scope]) // appended to the end
  }

  const searchTrimmed = search.trim()
  const filtered = useMemo(
    () => suggestions.filter((s) => s.toLowerCase().includes(searchTrimmed.toLowerCase())),
    [suggestions, searchTrimmed],
  )
  const canAddCustom =
    searchTrimmed !== "" && !suggestions.includes(searchTrimmed) && SCOPE_TOKEN_REGEX.test(searchTrimmed)

  return (
    <Field className={cn(containerClassName)}>
      <FieldLabel htmlFor={fieldId}>
        {label}
        {required && <span className="text-red-500 ml-1">*</span>}
      </FieldLabel>

      <div className="flex gap-2">
        <Input
          id={fieldId}
          value={value}
          placeholder={placeholder}
          disabled={disabled}
          onChange={(e) => onValueChange(e.target.value)}
          className={cn("flex-1", shownError && "border-red-500 focus-visible:ring-red-500/20")}
          aria-invalid={shownError ? "true" : "false"}
          aria-describedby={
            shownError ? `${fieldId}-error` : description ? `${fieldId}-description` : undefined
          }
        />
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <Button
              type="button"
              variant="outline"
              data-md-input-button
              disabled={disabled}
              className="shrink-0 gap-1.5"
            >
              Browse
              <ChevronsUpDown className="h-3.5 w-3.5 opacity-60" />
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-[280px] p-0" align="end">
            <Command shouldFilter={false}>
              <CommandInput
                placeholder="Search or type a scope..."
                value={search}
                onValueChange={setSearch}
              />
              <CommandList>
                {filtered.length === 0 && !canAddCustom && <CommandEmpty>No scopes found.</CommandEmpty>}
                <CommandGroup>
                  {filtered.map((scope) => (
                    <CommandItem key={scope} value={scope} onSelect={() => toggle(scope)} className="gap-2">
                      <Checkbox checked={tokenSet.has(scope)} className="pointer-events-none" />
                      <span className="truncate">{scope}</span>
                    </CommandItem>
                  ))}
                  {canAddCustom && (
                    <CommandItem
                      value={searchTrimmed}
                      onSelect={() => {
                        toggle(searchTrimmed)
                        setSearch("")
                      }}
                      className="gap-2"
                    >
                      <Plus className="h-4 w-4" />
                      Add &ldquo;{searchTrimmed}&rdquo;
                    </CommandItem>
                  )}
                </CommandGroup>
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>
      </div>

      {description && !shownError && (
        <FieldDescription id={`${fieldId}-description`} className="text-muted-foreground">
          {description}
        </FieldDescription>
      )}
      {shownError && <FieldError id={`${fieldId}-error`}>{shownError}</FieldError>}
    </Field>
  )
}

FormScopeField.displayName = "FormScopeField"
