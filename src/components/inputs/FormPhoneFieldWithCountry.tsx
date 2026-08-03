import { forwardRef, useState, useCallback, useEffect, useRef, useMemo } from "react"
import { Check, ChevronsUpDown } from "lucide-react"
import { Field, FieldLabel, FieldDescription, FieldError } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Command, CommandInput, CommandList, CommandEmpty, CommandGroup, CommandItem } from "@/components/ui/command"
import { cn } from "@/lib/utils"
import { formatPhone } from "@/lib/validations/regex"
import { COUNTRY_CODES, type CountryCode } from "./countryCodes"

export interface FormPhoneFieldWithCountryProps {
  label: string
  error?: string
  description?: string
  required?: boolean
  disabled?: boolean
  containerClassName?: string
  labelClassName?: string
  errorClassName?: string
  descriptionClassName?: string
  value?: string
  onChange?: (value: string) => void
  onBlur?: () => void
}

const SORTED_COUNTRIES = [...COUNTRY_CODES].sort((a, b) => b.dial.length - a.dial.length)

function parseCountryCode(digits: string): { country: CountryCode; localDigits: string } {
  const match = SORTED_COUNTRIES.find((c) => {
    const dialDigits = c.dial.replace(/\D/g, '')
    return digits.startsWith(dialDigits) && dialDigits.length > 0
  })

  if (match) {
    const dialDigits = match.dial.replace(/\D/g, '')
    let country = match
    if (dialDigits === '1' && country.code !== 'US') {
      country = COUNTRY_CODES.find((c) => c.code === 'US') ?? match
    }
    return { country, localDigits: digits.slice(dialDigits.length) }
  }

  return {
    country: COUNTRY_CODES.find((c) => c.code === 'US') ?? COUNTRY_CODES[0],
    localDigits: digits,
  }
}

const MAX_LOCAL_DIGITS_BY_COUNTRY: Record<string, number> = {
  US: 10,
  CA: 10,
}

function getMaxLocalDigits(country: CountryCode): number {
  return MAX_LOCAL_DIGITS_BY_COUNTRY[country.code] ?? 10
}

export const FormPhoneFieldWithCountry = forwardRef<HTMLDivElement, FormPhoneFieldWithCountryProps>(
  (
    {
      label,
      error,
      description,
      required = false,
      disabled = false,
      containerClassName,
      labelClassName,
      errorClassName,
      descriptionClassName,
      value: externalValue,
      onChange,
      onBlur,
    },
    ref
  ) => {
    const [countryCode, setCountryCode] = useState('US')
    const [localDigits, setLocalDigits] = useState('')
    const [open, setOpen] = useState(false)
    const inputRef = useRef<HTMLInputElement>(null)
    const lastValueRef = useRef<string | undefined>(undefined)

    const selectedCountry = useMemo(
      () => COUNTRY_CODES.find((c) => c.code === countryCode) ?? COUNTRY_CODES[0],
      [countryCode]
    )

    useEffect(() => {
      if (externalValue === lastValueRef.current) return
      lastValueRef.current = externalValue

      if (externalValue) {
        const digits = externalValue.replace(/\D/g, '')
        const { country, localDigits: parsedLocal } = parseCountryCode(digits)
        setCountryCode(country.code)
        setLocalDigits(parsedLocal)
      } else {
        setCountryCode('US')
        setLocalDigits('')
      }
    }, [externalValue])

    const emitChange = useCallback(
      (code: string, digits: string) => {
        const country = COUNTRY_CODES.find((c) => c.code === code) ?? COUNTRY_CODES[0]
        const formatted = digits ? `${country.dial} ${formatPhone(digits)}` : country.dial
        onChange?.(formatted)
      },
      [onChange]
    )

    const handleCountryChange = useCallback(
      (code: string) => {
        setCountryCode(code)
        setOpen(false)
        const newDigits = ''
        setLocalDigits(newDigits)
        emitChange(code, newDigits)
        inputRef.current?.focus()
      },
      [emitChange]
    )

    const handlePhoneChange = useCallback(
      (e: React.ChangeEvent<HTMLInputElement>) => {
        const digits = e.target.value.replace(/\D/g, '').slice(0, getMaxLocalDigits(selectedCountry))
        setLocalDigits(digits)
        emitChange(countryCode, digits)
      },
      [countryCode, selectedCountry, emitChange]
    )

    const localFormatted = useMemo(() => {
      if (!localDigits) return ''
      return formatPhone(localDigits)
    }, [localDigits])

    const fieldId = label.toLowerCase().replace(/\s+/g, '-')

    return (
      <Field ref={ref} className={cn(containerClassName)}>
        <FieldLabel htmlFor={fieldId} className={cn(labelClassName)}>
          {label}
          {required && <span className="text-red-500 ml-1">*</span>}
        </FieldLabel>

        <div className="flex">
          <Popover open={open} onOpenChange={setOpen}>
            <PopoverTrigger asChild>
              <Button
                data-md-phone-country-trigger
                variant="outline"
                role="combobox"
                aria-expanded={open}
                disabled={disabled}
                className={cn(
                  "rounded-r-none border-r-0 w-[100px] shrink-0 justify-between gap-1.5 focus:z-10",
                  error && "border-red-500"
                )}
              >
                <span className="text-base leading-none">{selectedCountry.flag}</span>
                <span className="text-sm">{selectedCountry.dial}</span>
                <ChevronsUpDown className="size-3 opacity-50 shrink-0" />
              </Button>
            </PopoverTrigger>
            <PopoverContent className="p-0 w-[200px]" align="start">
              <Command>
                <CommandInput placeholder="Search countries..." />
                <CommandList>
                  <CommandEmpty>No country found</CommandEmpty>
                  <CommandGroup>
                    {COUNTRY_CODES.map((c) => (
                      <CommandItem
                        key={c.code}
                        value={`${c.code3} ${c.name} ${c.dial}`}
                        onSelect={() => handleCountryChange(c.code)}
                      >
                        <span className="text-base leading-none mr-1">{c.flag}</span>
                        <span className="font-medium text-xs tracking-wider w-8">{c.code3}</span>
                        <span className="text-muted-foreground text-xs">{c.dial}</span>
                        {c.code === countryCode && (
                          <Check className="ml-auto size-3.5" />
                        )}
                      </CommandItem>
                    ))}
                  </CommandGroup>
                </CommandList>
              </Command>
            </PopoverContent>
          </Popover>

          <Input
            data-md-phone-local-input
            ref={inputRef}
            id={fieldId}
            className={cn(
              "rounded-l-none flex-1",
              error && "border-red-500 focus-visible:ring-red-500/20"
            )}
            value={localFormatted}
            onChange={handlePhoneChange}
            onBlur={onBlur}
            placeholder="Phone number"
            autoComplete="tel-national"
            inputMode="numeric"
            disabled={disabled}
            aria-invalid={error ? "true" : "false"}
            aria-describedby={
              error ? `${fieldId}-error` :
              description ? `${fieldId}-description` :
              undefined
            }
          />
        </div>

        {description && !error && (
          <FieldDescription
            id={`${fieldId}-description`}
            className={cn("text-muted-foreground", descriptionClassName)}
          >
            {description}
          </FieldDescription>
        )}

        {error && (
          <FieldError id={`${fieldId}-error`} className={cn(errorClassName)}>
            {error}
          </FieldError>
        )}
      </Field>
    )
  }
)

FormPhoneFieldWithCountry.displayName = "FormPhoneFieldWithCountry"
