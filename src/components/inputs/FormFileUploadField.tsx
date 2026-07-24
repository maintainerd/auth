import { useRef } from "react"
import { Upload } from "lucide-react"
import { Field, FieldLabel, FieldDescription, FieldError } from "@/components/ui/field"
import { cn } from "@/lib/utils"

export interface FormFileUploadFieldProps {
  label: string
  description?: string
  accept?: string
  disabled?: boolean
  error?: string
  onChange: (file: File | null) => void
  containerClassName?: string
}

export function FormFileUploadField({
  label,
  description,
  accept,
  disabled = false,
  error,
  onChange,
  containerClassName,
}: FormFileUploadFieldProps) {
  const ref = useRef<HTMLInputElement | null>(null)
  const fieldId = label.toLowerCase().replace(/\s+/g, "-")

  return (
    <Field className={cn(containerClassName)}>
      <FieldLabel htmlFor={fieldId}>{label}</FieldLabel>
      {description && <FieldDescription>{description}</FieldDescription>}
      <div
        className={cn(
          "flex items-center gap-2 rounded-md border px-3 py-2 text-sm cursor-pointer",
          error ? "border-destructive" : "border-input",
          disabled && "cursor-not-allowed opacity-50",
        )}
        onClick={() => !disabled && ref.current?.click()}
      >
        <Upload className="size-4 text-muted-foreground shrink-0" />
        <span className="truncate text-muted-foreground">
          Click to choose a file
        </span>
      </div>
      <input
        ref={ref}
        id={fieldId}
        type="file"
        accept={accept}
        disabled={disabled}
        onChange={(e) => onChange(e.target.files?.[0] ?? null)}
        className="sr-only"
      />
      {error && <FieldError>{error}</FieldError>}
    </Field>
  )
}
